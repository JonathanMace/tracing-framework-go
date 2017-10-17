package localbaggage

import (
	"github.com/JonathanMace/tracing-framework-go/local"
	"github.com/tracingplane/tracingplane-go/tracingplane"
)

// Maintains goroutine-local localbaggage

var golocal = local.GoLocalDerivable(branchBaggage)

func branchBaggage(local interface{}) interface{} {
	baggage, ok := local.(*tracingplane.BaggageContext)
	if !ok || baggage == nil { return nil }
	branched := baggage.Branch()
	return &branched
}

// Get the current goroutine-local baggage
func Get() tracingplane.BaggageContext {
	baggage, ok := golocal.Get().(*tracingplane.BaggageContext)
	if !ok || baggage == nil { return tracingplane.BaggageContext{} }
	return *baggage
}

// Set the goroutine-local baggage to the provided baggage.
// Baggage will be automatically branched with any go f() statements
func Set(baggage tracingplane.BaggageContext) {
	golocal.Set(&baggage)
}

// Branch a new baggagecontext from the current goroutine-local baggage
func Branch() tracingplane.BaggageContext {
	return Get().Branch()
}

// Merge a baggagecontext with the current goroutine-local baggage
func Merge(other tracingplane.BaggageContext) {
	Set(Get().MergeWith(other))
}