package auth_test

import (
	"testing"
	"time"

	"github.com/untappedtech/conduit/internal/auth/cache"
)

func TestCache_SetGet(t *testing.T) {
	c := cache.NewTTLLRUCache(10, 5)

	c.Set("token1", "role1")
	role, ok := c.Get("token1")
	if !ok || role != "role1" {
		t.Fatalf("expected role1")
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	c := cache.NewTTLLRUCache(10, 1)

	c.Set("token1", "role1")
	time.Sleep(2 * time.Second)

	_, ok := c.Get("token1")
	if ok {
		t.Fatalf("expected expired token")
	}
}

func TestCache_LRUEviction(t *testing.T) {
	c := cache.NewTTLLRUCache(2, 10)

	c.Set("a", "ra")
	c.Set("b", "rb")
	c.Set("c", "rc") // evicts "a"

	_, ok := c.Get("a")
	if ok {
		t.Fatalf("expected a evicted")
	}
}

func TestCache_SentinelInvalidToken(t *testing.T) {
	c := cache.NewTTLLRUCache(10, 10)

	c.Set("bad", cache.SentinelInvalidToken)
	role, ok := c.Get("bad")
	if !ok || role != cache.SentinelInvalidToken {
		t.Fatalf("expected sentinel")
	}
}
