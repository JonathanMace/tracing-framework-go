package gokitutil

import (
	"context"
	"fmt"
	xtr "github.com/JonathanMace/tracing-framework-go/xtrace/client"
	http "net/http"
	tp "github.com/tracingplane/tracingplane-go/tracingplane"
)

func XTraceClientPreSendInterceptor(ctx context.Context, req *http.Request) context.Context {
	xtr.Log("XTraceClientPreSendInterceptor Adding Baggage to HTTP header")
	req.Header.Set("Baggage", tp.EncodeBase64(xtr.GetLocalBaggage()))
	return ctx
}

func XTraceServerPreHandleInterceptor(ctx context.Context, req *http.Request) context.Context {
	baggageStr := req.Header.Get("Baggage")
	baggage, err := tp.DecodeBase64(baggageStr)
	xtr.SetLocalBaggage(baggage)
	var msg string
	switch {
	case err == nil: msg = fmt.Sprintf("XTraceServerPreHandleInterceptor received HTTP response with Baggage %s", baggageStr)
	case err != nil: msg = fmt.Sprintf("XTraceServerPreHandleInterceptor received HTTP response -- error decoding baggage %s", baggageStr)
	}
	fmt.Println(msg)
	xtr.Log(msg)
	return ctx
}

func XTraceServerPostHandleInterceptor(ctx context.Context, rsp http.ResponseWriter) context.Context {
	xtr.Log("XTraceServerPostHandleInterceptor Adding Baggage to HTTP header")
	rsp.Header().Set("Baggage", tp.EncodeBase64(xtr.GetLocalBaggage()))
	return ctx
}

func XTraceClientPostReceiveInterceptor(ctx context.Context, rsp *http.Response) context.Context {
	baggageStr := rsp.Header.Get("Baggage")
	baggage, err := tp.DecodeBase64(baggageStr)
	xtr.SetLocalBaggage(baggage)
	var msg string
	switch {
	case err == nil: msg = fmt.Sprintf("XTraceClientPostReceiveInterceptor received HTTP response with Baggage %s", baggageStr)
	case err != nil: msg = fmt.Sprintf("XTraceClientPostReceiveInterceptor received HTTP response -- error decoding baggage %s", baggageStr)
	}
	fmt.Println(msg)
	xtr.Log(msg)
	return ctx
}