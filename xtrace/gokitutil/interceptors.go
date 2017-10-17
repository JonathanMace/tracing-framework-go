package gokitutil

import (
	"context"
	"fmt"
	xtr "github.com/JonathanMace/tracing-framework-go/xtrace/client"
	http "net/http"
	tp "github.com/tracingplane/tracingplane-go/tracingplane"
	"github.com/JonathanMace/tracing-framework-go/localbaggage"
)

func XTraceClientPreSendInterceptor(ctx context.Context, req *http.Request) context.Context {
	defer localbaggage.Clear()
	xtr.Log("XTraceClientPreSendInterceptor Adding Baggage to HTTP header")
	req.Header.Set("Baggage", tp.EncodeBase64(localbaggage.Get()))
	return ctx
}

func XTraceServerPreHandleInterceptor(ctx context.Context, req *http.Request) context.Context {
	baggageStr := req.Header.Get("Baggage")
	baggage, err := tp.DecodeBase64(baggageStr)
	localbaggage.Set(baggage)
	var msg string
	switch {
	case err == nil: msg = fmt.Sprintf("XTraceServerPreHandleInterceptor received HTTP request")
	case err != nil: msg = fmt.Sprintf("XTraceServerPreHandleInterceptor received HTTP request -- error decoding localbaggage %s", baggageStr)
	}
	fmt.Println(msg)
	xtr.Log(msg)
	return ctx
}

func XTraceServerPostHandleInterceptor(ctx context.Context, rsp http.ResponseWriter) context.Context {
	defer localbaggage.Clear()
	xtr.Log("XTraceServerPostHandleInterceptor Adding Baggage to HTTP header")
	rsp.Header().Set("Baggage", tp.EncodeBase64(localbaggage.Get()))
	return ctx
}

func XTraceClientPostReceiveInterceptor(ctx context.Context, rsp *http.Response) context.Context {
	baggageStr := rsp.Header.Get("Baggage")
	baggage, err := tp.DecodeBase64(baggageStr)
	localbaggage.Set(baggage)
	var msg string
	switch {
	case err == nil: msg = fmt.Sprintf("XTraceClientPostReceiveInterceptor received HTTP response")
	case err != nil: msg = fmt.Sprintf("XTraceClientPostReceiveInterceptor received HTTP response -- error decoding localbaggage %s", baggageStr)
	}
	fmt.Println(msg)
	xtr.Log(msg)
	return ctx
}