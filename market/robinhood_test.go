package market

import "testing"

func TestTopMoversInstrumentIds_Success_ReturnsInstrumentURIs(t *testing.T) {
	r := &robinhood{}
	result, err := r.getTopMoversInstrumentIds()

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if result == nil || len(result) != 20 {
		t.Fatalf("Failed with unexpected response: %v", result)
	}
}

func TestGetSymbol_UnknownInstrumentURI_ReturnsEmptyString(t *testing.T) {
	const instrumentURI string = "https://api.robinhood.com/instruments/unknown-instrumentID/"
	r := &robinhood{}
	result, err := r.getSymbol(instrumentURI)

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if result != "" {
		t.Fatalf("Failed with unexpected response: %v", result)
	}
}
