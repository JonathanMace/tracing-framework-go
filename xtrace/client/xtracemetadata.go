package client

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

var bagIndex = uint64(5)  // Hard code bagIndex 5 for now

// Get XTraceMetadata from the goroutine-local baggage
func getMetadata() (xmd XTraceMetadata) {
	GetLocalBaggage().ReadBag(bagIndex, &xmd)
	return
}

// Set XTraceMetadata in the goroutine-local baggage
func setMetadata(xmd XTraceMetadata) {
	baggage := GetLocalBaggage()
	baggage.Set(bagIndex, &xmd)
	SetLocalBaggage(baggage)
}

// Starts a new X-Trace task by generating and saving random TaskID
func NewTask(tags ...string) {
	var xmd XTraceMetadata
	xmd.SetTaskID(randInt64())
	setMetadata(xmd)
	LogWithTags("Starting X-Trace Task", tags...)
}

// Returns true if there is a non-zero TaskID, indicating that tracing is active
func HasTask() bool {
	xmd := getMetadata()
	return xmd.taskID != nil
}

// Stops the current X-Trace task, by dropping the TaskID and ParentEventIDs
func StopTask() {
	setMetadata(XTraceMetadata{})
}

// Generates an X-Trace event.
// Returns:
//	 valid
//			true if we are tracing an X-Trace task, false otherwise.  If false, the event should be ignored
//	 taskID
//			the ID of the current task
//	 parentEventIDs
//			IDs of causal predecessor events
//	 eventID
//			the unique ID of this event
func Event() (valid bool, taskID int64, parentEventIDs []int64, eventID int64) {
	xmd := getMetadata()

	if !xmd.HasTaskID() {
		return false, 0, nil, 0
	}

	// Get return values
	valid = true
	taskID = xmd.GetTaskID()
	parentEventIDs = xmd.GetParentEventIDs()
	eventID = randInt64()

	// Update metadata
	xmd.SetParentEventID(eventID)
	setMetadata(xmd)

	return
}

func randInt64() int64 {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(fmt.Errorf("could not read random bytes: %v", err))
	}
	// shift to guarantee high bit is 0 and thus
	// int64 version is non-negative
	return int64(binary.BigEndian.Uint64(b[:]) >> 1)
}
