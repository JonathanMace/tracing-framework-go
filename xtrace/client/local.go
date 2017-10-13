package client

import (
	"github.com/JonathanMace/tracing-framework-go/local"
	"github.com/tracingplane/tracingplane-go/tracingplane"
)

// Goroutine-local Baggage

var token local.Token

func init() {
	token = local.Register(&tracingplane.BaggageContext{}, local.Callbacks{
		func(l interface{}) interface{} {
			baggage := *(l.(*tracingplane.BaggageContext))
			branchedBaggage := baggage.Branch()
			return &branchedBaggage
		},
	})
}

// Runs the given function in a new goroutine, but copies the
// local vars from the current goroutine first.
func XGo(f func()) {
	go func(f1 func(), f2 func()) {
		f1()
		f2()
	}(local.GetSpawnCallback(), f)
}

func SetLocalBaggage(baggage tracingplane.BaggageContext) {
	*local.GetLocal(token).(*tracingplane.BaggageContext) = baggage
}

func GetLocalBaggage() tracingplane.BaggageContext {
	return *local.GetLocal(token).(*tracingplane.BaggageContext)
}

func BranchLocalBaggage() tracingplane.BaggageContext {
	return GetLocalBaggage().Branch()
}

// Merges the current goroutine-local baggage with the provided baggage
func MergeLocalWith(baggage tracingplane.BaggageContext) {
	SetLocalBaggage(GetLocalBaggage().MergeWith(baggage))
}