package local

import (
	"runtime"
)

// A goroutine-local variable
type GoLocal goroutinelocal
type goroutinelocal int

var noderive = func(local interface{}) interface{} { return nil }

// Registers a new goroutine-local variable.
// Register should only be called during initialization and in the main goroutine
func GoLocalSimple() GoLocal {
	return GoLocalDerivable(noderive)
}

// Registers a new goroutine-local variable.  When new goroutines are spun off with 'go func', the goroutine-local value
// in the new goroutine will be derived by calling deriveFunction.
// Register should only be called during initialization and in the main goroutine
func GoLocalDerivable(deriveFunction func(local interface{}) interface{}) GoLocal {
	gls.deriveCallbacks = append(gls.deriveCallbacks, deriveFunction)
	return GoLocal(len(gls.deriveCallbacks)-1)
}

// The global goroutine-local registry
type g struct {
	deriveCallbacks []func(local interface{}) interface{}  // The callbacks to derive new values for goroutines
}
var gls g

// The actual variable that's attached to the goroutine struct
type goroutinelocals []interface{}

// Implementation of the Derive function called whenever we spin off new goroutines
func (current goroutinelocals) Derive() runtime.Local {
	if current == nil { return nil }

	derived := make(goroutinelocals, len(gls.deriveCallbacks))

	for i, v := range current {
		if v != nil {
			derived[i] = gls.deriveCallbacks[i](v)
		}
	}

	return derived
}

func (local GoLocal) Set(value interface{}) {
	current, ok := runtime.GetLocal().(goroutinelocals)
	if !ok || current == nil {
		current = make(goroutinelocals, len(gls.deriveCallbacks))
	}
	if len(current) < len(gls.deriveCallbacks) {
		extension := make(goroutinelocals, len(gls.deriveCallbacks) - len(current))
		current = append(current, extension...)
	}
	current[local] = value
	runtime.SetLocal(current)
}

func (local GoLocal) Get() interface{} {
	current, ok := runtime.GetLocal().(goroutinelocals)
	if !ok || len(current) <= int(local) {
		return nil
	} else {
		return current[local]
	}
}