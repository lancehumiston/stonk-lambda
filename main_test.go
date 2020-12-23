package main

import (
	"testing"
	"time"
)

func TestGetRecomendationRating_KnownSymbol_ShouldNotBeEmpty(t *testing.T) {
	symbol := "FB"

	result, err := getRecomendationRating(symbol)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if result.Period != "0m" {
		t.Fatalf("Failed with unexpected empty response: %v", result)
	}
}

func TestGetRecomendationRating_UnknownSymbol_ShouldBeEmpty(t *testing.T) {
	symbol := "NOT_A_SYMBOL"

	result, err := getRecomendationRating(symbol)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if result.Period != "" {
		t.Fatalf("Failed with unexpected response: %v", result)
	}
}

func TestGetAnalysis_KnownSymbol_ShouldNotBeEmpty(t *testing.T) {
	symbol := "FB"

	gain, rating, data, err := getAnalysis(symbol)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if gain == 0 {
		t.Fatalf("Failed with unexpected response: %.2f", gain)
	}

	if rating.Period != "0m" {
		t.Fatalf("Failed with unexpected period response: %s", rating.Period)
	}

	if data.CurrentPrice.USD == 0 {
		t.Fatalf("Failed with unexpected response: %.2f", data.CurrentPrice.USD)
	}
}

func TestGetAnalysis_UnknownSymbol_ShouldBeEmpty(t *testing.T) {
	symbol := "NOT_A_SYMBOL"

	gain, rating, data, err := getAnalysis(symbol)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if gain != 0 {
		t.Fatalf("Failed with unexpected response: %.2f", gain)
	}

	if rating.Period != "" {
		t.Fatalf("Failed with unexpected period response: %s", rating.Period)
	}

	if data.CurrentPrice.USD != 0 {
		t.Fatalf("Failed with unexpected response: %.2f", data.CurrentPrice.USD)
	}
}

func TestGetItemTTL(t *testing.T) {
	now := time.Now().UTC()

	ttl := getItemTTL(now)

	if ttl <= now.Unix() {
		t.Fatalf("Expected ttl:%d to be after now:%d", ttl, now.Unix())
	}
}
