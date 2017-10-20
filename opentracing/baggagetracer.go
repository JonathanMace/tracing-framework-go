package opentracing

// Proxies to other tracers, but overrides the binary formats used to propagate trace contexts on the wire, in headers, etc.

import (
	ot "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"github.com/tracingplane/tracingplane-go/examples"
	"github.com/JonathanMace/tracing-framework-go/localbaggage"
	"github.com/openzipkin/zipkin-go-opentracing/flag"
)

type BaggageTracer interface {
	zipkin.Tracer
}

type baggageTracer struct {
	wrapped zipkin.Tracer
}

func Wrap(tracer zipkin.Tracer) ot.Tracer {
	return &baggageTracer{tracer}
}

func (tracer *baggageTracer) Options() zipkin.TracerOptions {
	return tracer.wrapped.Options()
}

func (tracer *baggageTracer) StartSpan(operationName string, opts ...ot.StartSpanOption) ot.Span {
	return tracer.wrapped.StartSpan(operationName, opts...)
}

func (tracer *baggageTracer) Inject(spanContext ot.SpanContext, format interface{}, carrier interface{}) error {
	// Bypass format and carrier, write straight to golocal
	sc, ok := spanContext.(zipkin.SpanContext)
	if !ok {
		return ot.ErrInvalidSpanContext
	}

	baggage := localbaggage.Get()

	var zb examples.ZipkinMetadata
	baggage.ReadBag(2, &zb)

	traceID := sc.TraceID.Low
	zb.SetTraceID(int64(traceID))
	zb.SetSpanID(int64(sc.SpanID))
	zb.SetSampled(sc.Sampled)

	if sc.ParentSpanID != nil {
		// we only set ParentSpanID header if there is a parent span
		zb.SetParentSpanID(int64(*sc.ParentSpanID))
	}

	// TODO: flags
	zb.Tags = sc.Baggage

	baggage.Set(2, &zb)
	localbaggage.Set(baggage)

	return nil
}

func (tracer *baggageTracer) Extract(format interface{}, carrier interface{}) (ot.SpanContext, error) {
	baggage := localbaggage.Get()

	var zb examples.ZipkinMetadata
	baggage.ReadBag(2, &zb)

	var sc zipkin.SpanContext
	if zb.HasTraceID() { sc.TraceID.Low = uint64(zb.GetTraceID()) }
	if zb.HasSpanID() { sc.SpanID = uint64(zb.GetSpanID()) }
	if zb.HasParentSpanID() { parentSpanID := uint64(zb.GetParentSpanID()); sc.ParentSpanID = &parentSpanID }
	if zb.HasSampled() && zb.GetSampled() { sc.Sampled = true; sc.Flags = flag.Sampled }
	if zb.HasSampled() && !zb.GetSampled() { sc.Sampled = false; sc.Flags = flag.SamplingSet }
	if !zb.HasSampled() { sc.Sampled = false }
	sc.Baggage = zb.Tags


	return sc, nil
}