package data

import (
	"testing"
	"time"
)

func TestGetItemTTL(t *testing.T) {
	now := time.Now().UTC()

	ttl := getItemTTL(now)

	if ttl <= now.Unix() {
		t.Fatalf("Expected ttl:%d to be after now:%d", ttl, now.Unix())
	}
}
