package runtimeops

import (
	"context"
	"errors"
	"testing"
	"time"
)

var (
	_ = errors.Is
	_ = time.Second
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestTask0021ReplayRejectsDifferentRequestPayload(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	must(t, s.SaveCommand(ctx, Command{TenantID: "fleet-s", Method: "POST", Path: "/landings", Key: "uplink-9", RequestHash: "payload-v1", Response: []byte("accepted-v1")}))
	body, err := s.ReplayCommand(ctx, "fleet-s", "POST", "/landings", "uplink-9", "payload-v2")
	if !IsConflict(err) || body != nil {
		t.Errorf("mismatched replay body=%q err=%v", body, err)
	}
	body, err = s.ReplayCommand(ctx, "fleet-s", "POST", "/landings", "uplink-9", "payload-v1")
	if err != nil || string(body) != "accepted-v1" {
		t.Errorf("matching replay body=%q err=%v", body, err)
	}
	body[0] = 'X'
	again, err := s.ReplayCommand(ctx, "fleet-s", "POST", "/landings", "uplink-9", "payload-v1")
	if err != nil || string(again) != "accepted-v1" {
		t.Errorf("stored response was polluted: %q err=%v", again, err)
	}
}
