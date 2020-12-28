package url

import "testing"

func TestGetShortenedAlias_Success_ReturnsShortenedUrlAlias(t *testing.T) {
	url := "https://google.com"

	result, err := GetShortenedAlias(url)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if result == "" {
		t.Fatal("Failed with unexpected empty response")
	}
}
