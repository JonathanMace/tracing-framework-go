package local

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"runtime"
)

type testLocal int

func (v *testLocal) Derive() runtime.Local {
	newv := testLocal(int(*v)+1)
	return &newv
}

func current() (hasLocal bool, isValid bool, value int) {
	local := runtime.GetLocal()
	hasLocal = local != nil
	valuePtr, ok := local.(*testLocal)
	isValid = ok && valuePtr != nil
	if isValid { value = int(*valuePtr) }
	return
}

func expect(t *testing.T, expectedValue int, value interface {}) {
	assert.NotNil(t, value)
	ptr, ok := value.(*testLocal)
	assert.True(t, ok)
	assert.NotNil(t, ptr)
	assert.Equal(t, expectedValue, int(*ptr))
}

func TestInterpretAtoms(t *testing.T) {
	assert.Equal(t, 0, len(gls.deriveCallbacks))
	assert.Equal(t, nil, runtime.GetLocal())
}

func TestSetLocal(t *testing.T) {
	assert.Equal(t, nil, runtime.GetLocal())

	hasLocal, _, _ := current()
	assert.False(t, hasLocal)

	v1 := testLocal(5)
	runtime.SetLocal(&v1)
	expect(t, 5, runtime.GetLocal())

	v2 := testLocal(10)
	runtime.SetLocal(&v2)
	expect(t, 10, runtime.GetLocal())

	vchan := make(chan interface{})
	go func() {
		vchan <- runtime.GetLocal()
	}()

	expect(t, 11, <- vchan)
	expect(t, 10, runtime.GetLocal())

	go func() {
		vchan <- runtime.GetLocal()
	}()

	expect(t, 11, <- vchan)
	expect(t, 10, runtime.GetLocal())

	go func() {
		vchan <- runtime.GetLocal()
		go func() {
			vchan <- runtime.GetLocal()
			go func() {
				vchan <- runtime.GetLocal()
				go func() {
					vchan <- runtime.GetLocal()
					go func() {
						vchan <- runtime.GetLocal()
					}()
				}()
			}()
		}()
	}()

	expect(t, 11, <- vchan)
	expect(t, 12, <- vchan)
	expect(t, 13, <- vchan)
	expect(t, 14, <- vchan)
	expect(t, 15, <- vchan)
	expect(t, 10, runtime.GetLocal())

}