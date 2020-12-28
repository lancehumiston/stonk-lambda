package market

import "testing"

func TestGetTopMovers_Success_ReturnsInstrumentURIs(t *testing.T) {
	result, err := GetTopMovers()

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if result == nil || len(result) != 20 {
		t.Fatalf("Failed with unexpected response: %v", result)
	}
}

func TestGetSymbol_UnknownInstrumentURI_ReturnsEmptyString(t *testing.T) {
	const instrumentURI string = "https://api.robinhood.com/instruments/unknown-instrumentID/"

	result, err := GetSymbol(instrumentURI)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if result != "" {
		t.Fatalf("Failed with unexpected response: %v", result)
	}
}

func TestGetAnalysis_KnownSymbol_ReturnsGainAndRating(t *testing.T) {
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

func TestGetAnalysis_UnknownSymbol_ReturnsEmptyResponse(t *testing.T) {
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

func TestGetCompanyName_KnownSymbol_ReturnsCompanyName(t *testing.T) {
	symbol := "FB"

	name, err := GetCompanyName(symbol)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if name != "Facebook, Inc." {
		t.Fatalf("Failed with unexpected response: %s", name)
	}
}

func TestGetCompanyName_UnknownSymbol_ReturnsEmptyResponse(t *testing.T) {
	symbol := "NOT_A_SYMBOL"

	name, err := GetCompanyName(symbol)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if name != "" {
		t.Fatalf("Failed with unexpected response: %s", name)
	}
}

func TestGetNews_KnownCompanyName_ReturnsNewsArticles(t *testing.T) {
	symbol := "Facebook, Inc."

	uris, err := GetNews(symbol)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if len(uris) < 1 {
		t.Fatal("Failed with unexpected empty response")
	}
}

func TestGetNews_UnknownCompanyName_ReturnsEmptyResponse(t *testing.T) {
	symbol := "NotARealCompany"

	uris, err := GetNews(symbol)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if len(uris) > 0 {
		t.Fatalf("Failed with unexpected response: %v", uris)
	}
}
