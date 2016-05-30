package main

import (
	e "github.com/relab/goxos/elog/event"
)

type timelineEvent struct {
	Event     e.Event
	Replica   string
	EventDesc string
}

type timeline []timelineEvent

func (tl timeline) Len() int {
	return len(tl)
}

func (tl timeline) Swap(i, j int) {
	tl[i], tl[j] = tl[j], tl[i]
}

func (tl timeline) Less(i, j int) bool {
	return tl[i].Event.Time.Before(tl[j].Event.Time)
}
