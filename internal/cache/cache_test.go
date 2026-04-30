package cache

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestSetAndGetRoundTrip(t *testing.T) {
	cache := &Cache{Dir: t.TempDir()}

	if err := cache.Set("status:AA100", map[string]string{"status": "In Flight"}, time.Minute); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	raw, hit, err := cache.Get("status:AA100")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !hit {
		t.Fatalf("expected cache hit after Set")
	}

	var got map[string]string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal cache payload: %v", err)
	}
	if got["status"] != "In Flight" {
		t.Fatalf("unexpected cached payload: %#v", got)
	}
}

func TestGetExpiredEntryTreatsEntryAsMissAndRemovesFile(t *testing.T) {
	cache := &Cache{Dir: t.TempDir()}
	key := "status:AA100"

	if err := cache.Set(key, map[string]string{"status": "Expired"}, -time.Second); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	_, hit, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if hit {
		t.Fatalf("expected expired cache entry to miss")
	}
	if _, err := os.Stat(cache.keyPath(key)); !os.IsNotExist(err) {
		t.Fatalf("expected expired cache file to be removed, stat err=%v", err)
	}
}

func TestCleanupRemovesExpiredEntries(t *testing.T) {
	cache := &Cache{Dir: t.TempDir()}

	// Set one entry that is still valid and one that is expired.
	if err := cache.Set("valid:key", "still-here", 5*time.Minute); err != nil {
		t.Fatalf("Set valid key: %v", err)
	}
	if err := cache.Set("expired:key", "gone", -1*time.Second); err != nil {
		t.Fatalf("Set expired key: %v", err)
	}

	removed, err := cache.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 entry removed, got %d", removed)
	}

	// The valid entry should still be retrievable.
	_, hit, _ := cache.Get("valid:key")
	if !hit {
		t.Fatalf("expected valid entry to remain after cleanup")
	}

	// The expired entry should be gone.
	_, hit, _ = cache.Get("expired:key")
	if hit {
		t.Fatalf("expected expired entry to be removed by Cleanup")
	}
}

func TestCleanupRemovesCorruptFiles(t *testing.T) {
	cache := &Cache{Dir: t.TempDir()}

	// Write a corrupt file directly.
	if err := os.WriteFile(cache.keyPath("bad"), []byte("not-json"), 0600); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	removed, err := cache.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 corrupt file removed, got %d", removed)
	}

	if _, err := os.Stat(cache.keyPath("bad")); !os.IsNotExist(err) {
		t.Fatalf("expected corrupt file to be deleted")
	}
}

func TestCleanupOnNonexistentDirReturnsNil(t *testing.T) {
	cache := &Cache{Dir: "/tmp/flightcli-cache-nonexistent-dir-xyz"}
	removed, err := cache.Cleanup()
	if err != nil {
		t.Fatalf("expected no error for nonexistent dir, got: %v", err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed for nonexistent dir, got %d", removed)
	}
}

func TestGetTreatsCorruptFileAsMiss(t *testing.T) {
	cache := &Cache{Dir: t.TempDir()}
	key := "status:AA100"

	if err := os.WriteFile(cache.keyPath(key), []byte("not-json"), 0600); err != nil {
		t.Fatalf("write corrupt cache file: %v", err)
	}

	_, hit, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if hit {
		t.Fatalf("expected corrupt cache file to be treated as a miss")
	}
}
