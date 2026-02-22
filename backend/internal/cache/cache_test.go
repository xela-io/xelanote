package cache

import (
	"testing"
	"time"
)

func TestCache_SetGet(t *testing.T) {
	c := New(50 * time.Millisecond)
	defer c.Close()

	c.Set("k1", "value")
	if v, ok := c.Get("k1"); !ok || v.(string) != "value" {
		t.Fatalf("expected cached value")
	}
}

func TestCache_Expires(t *testing.T) {
	c := New(10 * time.Millisecond)
	defer c.Close()

	c.Set("k1", "value")
	time.Sleep(20 * time.Millisecond)
	if _, ok := c.Get("k1"); ok {
		t.Fatalf("expected expired value")
	}
}

func TestCache_DeleteByPrefix(t *testing.T) {
	c := New(1 * time.Minute)
	defer c.Close()

	c.Set("a:1", 1)
	c.Set("a:2", 2)
	c.Set("b:1", 3)

	c.DeleteByPrefix("a:")
	if _, ok := c.Get("a:1"); ok {
		t.Fatalf("expected key a:1 removed")
	}
	if _, ok := c.Get("a:2"); ok {
		t.Fatalf("expected key a:2 removed")
	}
	if _, ok := c.Get("b:1"); !ok {
		t.Fatalf("expected key b:1 to remain")
	}
}
