package client

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/JonathanMace/tracing-framework-go/xtrace/client/internal"
	"github.com/JonathanMace/tracing-framework-go/xtrace/internal/pubsub"
	"github.com/golang/protobuf/proto"
	"runtime"
	"math"
	"github.com/JonathanMace/tracing-framework-go/localbaggage"
)

var client *pubsub.Client

var connectOnce sync.Once = sync.Once{}
var disconnectOnce sync.Once = sync.Once{}

var DefaultServerString string = "localhost:5563"

// Connect initializes a connection to the X-Trace
// server. Connect must be called (and must complete
// successfully) before Log can be called.
func Connect(server string) (err error) {
	connectOnce.Do(func() {
		client, err = pubsub.NewClient(server)
		if err != nil {
			client = nil
		}
	})
	return
}

// Disconnect removes the existing connection to the X-Trace server
func Disconnect() {
	disconnectOnce.Do(func() {
		if client != nil {
			client.Close()
			client = nil
		}
	})
}

type xtraceWriter struct{}

func (x xtraceWriter) Write(p []byte) (n int, err error) {
	Log(string(p))
	return len(p), nil
}

func MakeWriter(wrapped ...io.Writer) io.Writer {
	return io.MultiWriter(append(wrapped, xtraceWriter{})...)
}

var topic = []byte("xtpb")
var processName = strings.Join(os.Args, " ")

var pnameOnce = sync.Once{}

func SetProcessName(pname string) {
	pnameOnce.Do(func() {
		processName = pname
	})
}

// Log a given message with the extra preceding events given
// adds a ParentEventId for all precedingEvents _in addition_ to the recorded parent of this event
func logWithTags(str string, tags ...string) {
	if client == nil {
		//fail silently
		return
	}

	valid, taskID, parentEventIDs, eventID := Event()

	if !valid {
		return
	}

	var report internal.XTraceReportv4
	report.TaskId = &taskID
	report.EventId = &eventID
	report.ParentEventId = parentEventIDs

	hrt := time.Now().UnixNano()
	ts := hrt / 1000000
	report.Timestamp = &ts
	report.Hrt = &hrt

	pid := int32(os.Getpid())
	fakeTid := int32(uint32(uint64(runtime.GetGoID()) % math.MaxUint32))
	report.ProcessId = &pid
	report.ProcessName = &processName
	report.ThreadId = &fakeTid

	host, err := os.Hostname()
	if err != nil {
		report.Host = &host
	}

	report.Label = &str
	report.Agent = &str

	if len(tags) > 0 {
		report.Tags = tags
	}

	report.Key = append(report.Key, "Baggage")
	report.Value = append(report.Value, fmt.Sprint(localbaggage.Get().Atoms))

	buf, err := proto.Marshal(&report)
	if err != nil {
		fmt.Fprintf(os.Stderr, "internal error: %v", err)
	}

	// NOTE(joshlf): Currently, Log blocks until the log message
	// has been written to the TCP connection to the X-Trace server.
	// This makes testing easier, but ideally we should optimize
	// so that the program can block before it quits, but each
	// call to Log is not blocking.
	client.PublishBlock(topic, buf)
}

// Log logs the given message. Log must not be
// called before Connect has been called successfully.
func Log(str string) {
	logWithTags(str)
}

func LogWithTags(str string, tags ...string) {
	logWithTags(str, tags...)
}

func Logf(format string, args ...interface{}) {
	logWithTags(fmt.Sprintf(format, args...))
}
