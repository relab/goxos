package event

import (
	"fmt"
	"time"
)

type Event struct {
	Type    Type
	Time    time.Time
	EndTime time.Time
	Value   uint64
}

type Type uint8

const (
	// General: 0-15
	Unknown       Type = 0
	Start         Type = 1
	Running       Type = 2
	Processing    Type = 3
	ShutdownStart Type = 4
	Exit          Type = 5

	// Throughput: 16-23
	ThroughputSample Type = 16

	// Common-Standby: 24-31
	InitListening     Type = 24
	InitTransferStart Type = 25
	InitTransferDone  Type = 26
	InitInitialized   Type = 27

	// LR-Standby: 32-39
	LRWaitForActivation Type = 32
	LRActivated         Type = 33

	// Reconfig-Standby: 40-47
	ReconfigFirstSlotReceived Type = 40
	ReconfigJoined            Type = 41

	// Failure Handling Common: 48-55
	FailureHandlingSuspect   Type = 48
	FailureHandlingInitStart Type = 49
	FailureHandlingInitDone  Type = 50

	// Catch-up: 56-63
	CatchUpMakeReq          Type = 56
	CatchUpSentReq          Type = 57
	CatchUpRecvReq          Type = 58
	CatchUpSentResp         Type = 59
	CatchUpRecvResp         Type = 60
	CatchUpDoneHandlingResp Type = 61

	// Live Replacement: 64-71
	LRStart            Type = 64
	LRPrepareEpochSent Type = 65
	LRPrepareEpochRecv Type = 66
	LRActivatedFromPE  Type = 67
	LRPreConnectSleep  Type = 68

	// Reconfiguration: 72-79
	ReconfigStart         Type = 72
	ReconfigPropose       Type = 73
	ReconfigExecReconfCmd Type = 74
	ReconfigDone          Type = 75

	// ARec: 80-87
	ARecStart            Type = 80
	ARecRMSent           Type = 81
	ARecStopPaxos        Type = 82
	ARecActivatedFromCPs Type = 83
	ARecRestart          Type = 84

	// Client Request Latency: 88-95
	ClientRequestLatency Type = 88
)

//go:generate stringer -type=Type

func NewEvent(t Type) Event {
	return Event{
		Type: t,
		Time: time.Now(),
	}
}

func NewEventWithMetric(t Type, v uint64) Event {
	return Event{
		Type:  t,
		Time:  time.Now(),
		Value: v,
	}
}

func NewTimedEvent(t Type, start time.Time) Event {
	return Event{
		Type:    t,
		Time:    start,
		EndTime: time.Now(),
	}
}

const layout = "2006-01-02 15:04:05.999999999"

func (e Event) String() string {
	switch e.Type {
	case FailureHandlingSuspect, ThroughputSample:
		return fmt.Sprintf("%v:\t%30v %3d",
			e.Time.Format(layout), e.Type, e.Value)
	case ClientRequestLatency:
		return fmt.Sprintf("%v:\t%30v Latency: %v",
			e.EndTime.Format(layout), e.Type, e.EndTime.Sub(e.Time))
	default:
		if e.EndTime.IsZero() {
			return fmt.Sprintf("%v:\t%30v",
				e.Time.Format(layout), e.Type)
		}
		return fmt.Sprintf("%v:\t%30v %v",
			e.Time.Format(layout), e.Type, e.EndTime.Format(layout))
	}
}
