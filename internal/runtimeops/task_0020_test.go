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

func TestTask0020IdempotencyKeyIncludesOperationIdentity(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	commands := []Command{{TenantID: "fleet-r", Method: "POST", Path: "/permits", Key: "link-42", RequestHash: "create-hash", Response: []byte("created")}, {TenantID: "fleet-r", Method: "DELETE", Path: "/voyages/v1", Key: "link-42", RequestHash: "close-hash", Response: []byte("closed")}}
	for _, command := range commands {
		if err := s.SaveCommand(ctx, command); err != nil {
			t.Errorf("save %s %s: %v", command.Method, command.Path, err)
		}
	}
	for _, command := range commands {
		body, err := s.ReplayCommand(ctx, command.TenantID, command.Method, command.Path, command.Key, command.RequestHash)
		if err != nil || string(body) != string(command.Response) {
			t.Errorf("replay %s %s=%q err=%v", command.Method, command.Path, body, err)
		}
	}
	if err := s.SaveCommand(ctx, Command{TenantID: "fleet-r", Method: "POST", Path: "/permits", Key: "link-42", RequestHash: "changed", Response: []byte("bad")}); !IsConflict(err) {
		t.Errorf("same operation conflict=%v", err)
	}
}
