package url

import (
	"strings"
	"testing"
)

func TestGetShortenedAlias_Success_ReturnsShortenedUrlAlias(t *testing.T) {
	url := "https://google.com"
	expectedPrefix := "https://cutt.ly/"

	result, err := GetShortenedAlias(url)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if !strings.HasPrefix(result, expectedPrefix) {
		t.Fatalf("Failed with unexpected response:%s", result)
	}
}
