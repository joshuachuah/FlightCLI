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
