package runtimeops

import "time"

type Permit struct {
	ID, TenantID, Slot, State string
	Version                   int64
}

type Event struct {
	ID, TenantID, BatchID, Status string
	Magnitude                     float64
}

type Job struct {
	ID, TenantID, State, Payload string
	Attempts, MaxAttempts        int
	AvailableAt                  time.Time
	LeaseUntil                   *time.Time
}

type Command struct {
	TenantID, Method, Path, Key, RequestHash string
	Response                                 []byte
}

type Checkpoint struct {
	TenantID, Stream string
	Generation       int64
	Payload          []byte
}

type EventPage struct {
	Items []Event
	Total int
}
