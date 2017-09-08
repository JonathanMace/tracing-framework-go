package gokitutil

import (
	"context"
	"fmt"
	xtr "github.com/JonathanMace/tracing-framework-go/xtrace/client"
	http "net/http"
)

func XTraceClientPreSendInterceptor(ctx context.Context, req *http.Request) context.Context {
	fmt.Print("XTraceClientPreSendInterceptor")
	xtr.Log("XTraceClientPreSendInterceptor Adding Baggage to HTTP header")
	// TODO: baggage
	return ctx
}

func XTraceServerPreHandleInterceptor(ctx context.Context, req *http.Request) context.Context {
	fmt.Printf("XTraceServerPreHandleInterceptor Received HTTP request with Baggage: %s", req.Header.Get("Baggage"))
	xtr.Log(fmt.Sprintf("Received HTTP request with Baggage %s", req.Header.Get("Baggage")))
	return ctx
}

func XTraceServerPostHandleInterceptor(ctx context.Context, rsp http.ResponseWriter) context.Context {
	fmt.Print("XTraceServerPostHandleInterceptor")
	xtr.Log(fmt.Sprintf("Responding to HTTP request"))
	// TODO: baggage
	return ctx
}

func XTraceClientPostReceiveInterceptor(ctx context.Context, rsp *http.Response) context.Context {
	fmt.Printf("XTraceClientPostReceiveInterceptor Received HTTP response with Baggage: %s", rsp.Header.Get("Baggage"))
	xtr.Log(fmt.Sprintf("Received HTTP response with Baggage %s", rsp.Header.Get("Baggage")))
	return ctx
}