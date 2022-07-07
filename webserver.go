package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

func startWebserver(env Env, api *API) string {
	loginUsers, err := json.Marshal(env.LoginUsers)
	if err != nil {
		log.Fatal(err)
	}

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		body := ctx.Request.Body()
		switch path {
		case "/send_cv":
			cvContent := map[string]interface{}{}
			err := json.Unmarshal(body, &cvContent)
			if err != nil {
				errorResp(ctx, 400, "invalid CV")
				return
			}

			referenceNrInterf, ok := cvContent["referenceNumber"]
			if !ok {
				errorResp(ctx, 400, "referenceNumber field does not exists")
				return
			}

			referenceNr, ok := referenceNrInterf.(string)
			if !ok {
				errorResp(ctx, 400, "referenceNumber must be a string")
				return
			}

			cacheEntryExists := api.CacheEntryExists(referenceNr)
			if cacheEntryExists {
				// Cannot send the same cv twice
				ctx.Response.AppendBodyString("false")
				return
			}

			hasMatch := false
			if api.MockMode {
				api.SetCacheEntry(referenceNr, time.Hour*72)
				hasMatch = true
			} else {
				scanCVBody := json.RawMessage(append(append([]byte(`{"cv":`), body...), '}'))

				for idx, conn := range api.connections {
					var response struct {
						HasMatches bool `json:"hasMatches"`
					}

					err = conn.Post("/api/v1/scraper/scanCV", scanCVBody, &response)
					if err != nil {
						errorResp(ctx, 500, err.Error())
						return
					}

					if idx == api.primaryConnection {
						hasMatch = response.HasMatches
						if hasMatch {
							// Only cache the CVs that where matched to something
							api.SetCacheEntry(referenceNr, time.Hour*72) // 3 days
						}
					}
				}
			}

			ctx.Response.AppendBodyString("true")
		case "/users":
			ctx.Response.AppendBody(loginUsers)
		case "/set_cached_reference", "/set_short_cached_reference":
			refNr := string(ctx.Request.Body())
			if refNr == "" {
				errorResp(ctx, 400, "reference number cannot be an empty string")
				return
			}

			if path == "/set_cached_reference" {
				api.SetCacheEntry(refNr, time.Hour*72) // 3 days
			} else {
				api.SetCacheEntry(refNr, time.Hour*12) // 0.5 days
			}

			ctx.Response.AppendBodyString("true")
		case "/get_cached_reference":
			refNr := string(ctx.Request.Body())
			if refNr == "" {
				errorResp(ctx, 400, "reference number cannot be an empty string")
				return
			}

			if api.CacheEntryExists(refNr) {
				ctx.Response.AppendBodyString("true")
			} else {
				ctx.Response.AppendBodyString("false")
			}
		default:
			errorResp(ctx, 404, "404 not found")
			return
		}
		ctx.Response.Header.Set("Content-Type", "application/json")
	}

	s := &fasthttp.Server{Handler: requestHandler}

	portAttempt := 4_000
	for {
		portAttempt++
		if portAttempt > 6_000 {
			// Give up
			log.Fatal("Could not find a free port to start the webserver")
		}

		address := fmt.Sprintf("127.0.0.1:%d", portAttempt)

		l, err := net.Listen("tcp4", address)
		if err != nil {
			if strings.Contains(err.Error(), "address already in use") {
				// Retry with a diffrent port
				continue
			}
			log.Fatal("Error in Listen: " + err.Error())
		}

		go func() {
			err = s.Serve(l)
			if err != nil {
				log.Fatal("Error in Serve: " + err.Error())
			}
		}()

		return "http://" + address
	}
}

func errorResp(ctx *fasthttp.RequestCtx, code int, msg string) {
	ctx.Response.AppendBodyString(msg)
	ctx.Response.SetStatusCode(code)
}
