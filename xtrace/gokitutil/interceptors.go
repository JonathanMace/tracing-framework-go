package gokitutil

import (
	"context"
	"fmt"
	xtr "github.com/JonathanMace/tracing-framework-go/xtrace/client"
	http "net/http"
	tp "github.com/tracingplane/tracingplane-go/tracingplane"
)

func setXTraceFromBaggage(baggage tp.BaggageContext) {
	// TODO: hacking this in here for POC
	var xmd xtr.XTraceMetadata
	baggage.ReadBag(5, &xmd)
	if xmd.HasTaskID() {
		xtr.SetTaskID(xmd.GetTaskID())
		xtr.SetEventIDs(xmd.GetParentEventIDs()...)
	} else {
		xmd.SetTaskID(0)
	}
}

func setXTraceToBaggage() tp.BaggageContext {
	var xmd xtr.XTraceMetadata
	taskID := xtr.GetTaskID()
	if taskID != 0 {
		xmd.SetTaskID(taskID)
		xmd.AddParentEventID(xtr.GetEventIDs()...)
	}
	var baggage tp.BaggageContext
	baggage.Set(5, &xmd)
	return baggage
}

func XTraceClientPreSendInterceptor(ctx context.Context, req *http.Request) context.Context {
	xtr.Log("XTraceClientPreSendInterceptor Adding Baggage to HTTP header")
	baggage := setXTraceToBaggage()
	req.Header.Set("Baggage", tp.EncodeBase64(baggage))
	return ctx
}

func XTraceServerPreHandleInterceptor(ctx context.Context, req *http.Request) context.Context {
	baggageStr := req.Header.Get("Baggage")
	baggage, err := tp.DecodeBase64(baggageStr)
	setXTraceFromBaggage(baggage)
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
	baggage := setXTraceToBaggage()
	rsp.Header().Set("Baggage", tp.EncodeBase64(baggage))
	return ctx
}

func XTraceClientPostReceiveInterceptor(ctx context.Context, rsp *http.Response) context.Context {
	baggageStr := rsp.Header.Get("Baggage")
	baggage, err := tp.DecodeBase64(baggageStr)
	setXTraceFromBaggage(baggage)
	var msg string
	switch {
	case err == nil: msg = fmt.Sprintf("XTraceClientPostReceiveInterceptor received HTTP response with Baggage %s", baggageStr)
	case err != nil: msg = fmt.Sprintf("XTraceClientPostReceiveInterceptor received HTTP response -- error decoding baggage %s", baggageStr)
	}
	fmt.Println(msg)
	xtr.Log(msg)
	return ctx
}