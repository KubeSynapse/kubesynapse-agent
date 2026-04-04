package dedup

import (
	"context"
	"testing"
	"time"
)

func TestCache_ShouldAlert(t *testing.T) {
	ctx := context.Background()
	cache := NewInMemory(2) // 2-second cooldown

	// First alert should fire
	if !cache.ShouldAlert(ctx, "default", "my-pod") {
		t.Error("First alert should return true")
	}

	// Immediate second alert should be blocked
	if cache.ShouldAlert(ctx, "default", "my-pod") {
		t.Error("Duplicate alert within cooldown should return false")
	}

	// Different pod should still fire
	if !cache.ShouldAlert(ctx, "default", "other-pod") {
		t.Error("Different pod should return true")
	}

	// Wait for cooldown to expire
	time.Sleep(3 * time.Second)

	// Same pod should fire again after cooldown
	if !cache.ShouldAlert(ctx, "default", "my-pod") {
		t.Error("Alert after cooldown should return true")
	}
}

func TestCache_DifferentNamespaces(t *testing.T) {
	ctx := context.Background()
	cache := NewInMemory(300)

	// Same pod name in different namespaces should be independent
	if !cache.ShouldAlert(ctx, "production", "api-server") {
		t.Error("First namespace alert should fire")
	}
	if !cache.ShouldAlert(ctx, "staging", "api-server") {
		t.Error("Different namespace alert should fire")
	}

	// Same namespace+pod should be blocked
	if cache.ShouldAlert(ctx, "production", "api-server") {
		t.Error("Same namespace+pod should be blocked")
	}
}

func TestCache_Reset(t *testing.T) {
	ctx := context.Background()
	cache := NewInMemory(300)

	cache.ShouldAlert(ctx, "default", "pod-1")
	cache.Reset(ctx, "default", "pod-1")

	// After reset, should alert again
	if !cache.ShouldAlert(ctx, "default", "pod-1") {
		t.Error("After reset, alert should fire")
	}
}

