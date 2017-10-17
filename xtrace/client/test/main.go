// +build !local

package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/JonathanMace/tracing-framework-go/xtrace/client"
	"github.com/tracingplane/tracingplane-go/tracingplane"
	"github.com/JonathanMace/tracing-framework-go/localbaggage"
)

func main() {

	err := client.Connect("localhost:5563")
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to X-Trace server: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Logging")

	client.StartTask("go/test/main.go")

	client.Log("1")
	client.Log("2")
	client.Log("3")

	var wg sync.WaitGroup
	var donebaggage tracingplane.BaggageContext
	wg.Add(1)
	go func() {
		client.Log("4")
		wg.Done()
		donebaggage = localbaggage.Get()
	}()

	client.Log("5")
	wg.Wait()
	localbaggage.Merge(donebaggage)
	client.Log("6")

	time.Sleep(time.Minute)
}
