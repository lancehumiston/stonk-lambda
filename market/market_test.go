package market

import "testing"

func TestGetAnalysis_KnownSymbol_ShouldNotBeEmpty(t *testing.T) {
	symbol := "FB"

	gain, rating, data, err := GetAnalysis(symbol)

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

	gain, rating, data, err := GetAnalysis(symbol)

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
