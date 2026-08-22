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

func TestTask0019IdempotencyKeysAreTenantScoped(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	a := Command{TenantID: "fleet-a", Method: "POST", Path: "/voyages", Key: "satellite-retry", RequestHash: "hash-a", Response: []byte("voyage-a")}
	b := Command{TenantID: "fleet-b", Method: "POST", Path: "/voyages", Key: "satellite-retry", RequestHash: "hash-b", Response: []byte("voyage-b")}
	must(t, s.SaveCommand(ctx, a))
	if err := s.SaveCommand(ctx, b); err != nil {
		t.Errorf("second tenant save failed: %v", err)
	}
	for _, item := range []struct{ tenant, hash, want string }{{"fleet-a", "hash-a", "voyage-a"}, {"fleet-b", "hash-b", "voyage-b"}} {
		body, err := s.ReplayCommand(ctx, item.tenant, "POST", "/voyages", "satellite-retry", item.hash)
		if err != nil || string(body) != item.want {
			t.Errorf("replay %s=%q err=%v", item.tenant, body, err)
		}
	}
	if err := s.SaveCommand(ctx, Command{TenantID: "fleet-a", Method: "POST", Path: "/voyages", Key: "satellite-retry", RequestHash: "different", Response: []byte("bad")}); !IsConflict(err) {
		t.Errorf("same-tenant conflict=%v", err)
	}
}
