/*
Copyright 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Cache is a file-based key/value store with TTL support.
type Cache struct {
	Dir string
}

type entry struct {
	Data      json.RawMessage `json:"data"`
	ExpiresAt time.Time       `json:"expires_at"`
}

// New returns a Cache rooted at ~/.flightcli/cache/.
// It creates the directory if it does not exist.
func New() (*Cache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".flightcli", "cache")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("could not create cache directory: %w", err)
	}
	return &Cache{Dir: dir}, nil
}

func (c *Cache) keyPath(key string) string {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(key)))
	return filepath.Join(c.Dir, hash+".json")
}

// Get retrieves a cached value. Returns (data, true, nil) on a valid hit,
// (nil, false, nil) on a miss or expiry, and (nil, false, err) on I/O error.
func (c *Cache) Get(key string) (json.RawMessage, bool, error) {
	path := c.keyPath(key)
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading cache: %w", err)
	}

	var e entry
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, false, nil // corrupt file - treat as miss
	}

	if time.Now().After(e.ExpiresAt) {
		os.Remove(path) // clean up expired file silently
		return nil, false, nil
	}

	return e.Data, true, nil
}

// Cleanup removes all expired cache entries from disk.
// It returns the number of entries removed and any error encountered
// while reading the cache directory. Individual file removal errors
// are not reported — only directory-level errors are.
func (c *Cache) Cleanup() (int, error) {
	entries, err := os.ReadDir(c.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("reading cache directory: %w", err)
	}

	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(c.Dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var ent entry
		if err := json.Unmarshal(raw, &ent); err != nil {
			// Corrupt file — remove it
			os.Remove(path)
			removed++
			continue
		}
		if time.Now().After(ent.ExpiresAt) {
			os.Remove(path)
			removed++
		}
	}
	return removed, nil
}

// Set writes a value to cache with the given TTL.
func (c *Cache) Set(key string, data interface{}, ttl time.Duration) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling cache data: %w", err)
	}

	e := entry{
		Data:      raw,
		ExpiresAt: time.Now().Add(ttl),
	}

	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshaling cache entry: %w", err)
	}

	return os.WriteFile(c.keyPath(key), b, 0600)
}
