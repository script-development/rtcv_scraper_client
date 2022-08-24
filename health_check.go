package main

import (
	"fmt"
	"log"
	"net"

	"github.com/valyala/fasthttp"
)

func startHealthCheckServer(port string) {
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		ctx.Response.AppendBody([]byte("true"))
		ctx.Response.Header.Set("Content-Type", "application/json")
	}

	s := &fasthttp.Server{Handler: requestHandler}

	address := "0.0.0.0:" + port
	l, err := net.Listen("tcp4", address)
	if err != nil {
		log.Fatalf("Error, unable to start health check service at \"%s\" error %s", address, err.Error())
	}

	fmt.Println("running health check service at", address)

	err = s.Serve(l)
	if err != nil {
		log.Fatal("Error in health check service Serve: " + err.Error())
	}
}
