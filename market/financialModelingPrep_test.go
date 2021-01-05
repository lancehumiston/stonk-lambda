package market

import "testing"

func TestGetTopMovers_Success_ReturnsTopMovers(t *testing.T) {
	f := &financialModelingPrep{}
	r, err := f.GetTopMovers()

	if err != nil {
		t.Fatalf("Failed with unexpected error: %s", err)
	}

	if r == nil {
		t.Fatalf("Failed with unexpected nil response: %v", r)
	}
}
