package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

func startWebserver(env Env, api *API, loginUsers []EnvUser) string {
	loginUsersJSON, err := json.Marshal(loginUsers)
	if err != nil {
		log.Fatal(err)
	}

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		body := ctx.Request.Body()
		switch path {
		case "/send_cv":
			cvForChecking := StrippedCV{}
			err := json.Unmarshal(body, &cvForChecking)
			if err != nil {
				errorResp(ctx, 400, "invalid CV")
				return
			}

			err = cvForChecking.checkRefNr()
			if err != nil {
				errorResp(ctx, 400, err.Error())
				return
			}

			cacheEntryExists := api.CacheEntryExists(cvForChecking.ReferenceNumber)
			if cacheEntryExists {
				// Cannot send the same cv twice
				ctx.Response.AppendBodyString("false")
				return
			}

			hasMatch := false
			if api.MockMode {
				api.SetCacheEntry(cvForChecking.ReferenceNumber, time.Hour*72)
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
							api.SetCacheEntry(cvForChecking.ReferenceNumber, time.Hour*72) // 3 days
						}
					}
				}
			}

			ctx.Response.AppendBodyString("true")
		case "/cvs_list":
			cvs := []StrippedCVWithOriginal{}
			err := json.Unmarshal(body, &cvs)
			if err != nil {
				errorResp(ctx, 400, "invalid CV")
				return
			}

			checkedRefNrs := map[string]struct{}{}
			for idx := 10; idx >= 0; idx-- {
				cv := cvs[idx]
				err := cv.checkRefNr()
				if err != nil {
					errorResp(ctx, 400, fmt.Sprintf("error in cv with index %d, error: %s", idx, err.Error()))
					return
				}

				_, ok := checkedRefNrs[cv.ReferenceNumber]
				if ok {
					// Remove this cv as it's a duplicate of another one in the list
					cvs = append(cvs[:idx], cvs[idx+1:]...)
					continue
				}
				checkedRefNrs[cv.ReferenceNumber] = struct{}{}

				if cv.checkMustHaveValidZip() != nil {
					// Remove this cv from the list as it does not have a valid zip code
					cvs = append(cvs[:idx], cvs[idx+1:]...)
					continue
				}
			}

			if !api.MockMode {
				for _, conn := range api.connections {
					err = conn.Post("/api/v1/scraper/allCVs", map[string]any{"cvs": cvs}, nil)
					if err != nil {
						errorResp(ctx, 500, err.Error())
						return
					}
				}
			}

			ctx.Response.AppendBodyString("true")
		case "/users":
			ctx.Response.AppendBody(loginUsersJSON)
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
		case "/server_response":
			if api.MockMode {
				ctx.Response.AppendBodyString("false")
			}

			api.HandleWebsocketResponse(ctx.Request.Body())
			ctx.Response.AppendBodyString("true")
		case "/server_request":
			// Cancel previous calls to this endpoint
			// There seems to be no way in fasthttp to check if the client has disconnected and thus the channel might be open forever
			// This is a bit of a dirty hack but it works we just cancel all previous calls to this endpoint when a new one is made
			api.CancelPreviouseCommunication()

			select {
			case <-api.CancelPreviouseCommunicationChan:
				errorResp(ctx, 400, "a new request has been opened")
			case req := <-api.WebsocketReq:
				ctx.Response.AppendBody(req)
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

		address := "127.0.0.1:" + strconv.Itoa(portAttempt)

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
