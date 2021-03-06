package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lancehumiston/stonk-lambda/market"
)

const gainThreshold float64 = 50

func TestUnique_Duplicates_ReturnsUniqueItems(t *testing.T) {
	items := []string{
		"a",
		"b",
		"c",
		"b",
		"c",
		"d",
	}
	expected := []string{
		"a",
		"b",
		"c",
		"d",
	}

	result := unique(items)

	if !IsEqual(result, expected) {
		t.Fatalf("Failed expected:%v actual:%v", expected, result)
	}
}

func IsEqual(a, b []string) bool {
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

type testCase struct {
	name          string
	symbol        string
	price         market.Price
	rating        market.RecommendationRating
	financialData market.FinancialData
	expected      error
}

var validateAgainstTresholdsTCs = []testCase{
	{
		name:   "Success targetMeanPrice > 1.5*currentPrice w/ buy rating",
		symbol: "GNOG",
		price: market.Price{
			MarketChange: market.Percent{
				Percent: gainThreshold,
			},
			PreMarketPrice: market.Currency{
				USD: 9,
			},
		},
		rating: market.RecommendationRating{
			Buy: 1,
		},
		financialData: market.FinancialData{
			CurrentPrice: market.Currency{
				USD: 10,
			},
			TargetMeanPrice: market.Currency{
				USD: 15,
			},
		},
		expected: nil,
	},
	{
		name:   "Success targetMeanPrice > 1.5*currentPrice w/ strongBuy rating",
		symbol: "GNOG",
		price: market.Price{
			MarketChange: market.Percent{
				Percent: gainThreshold,
			},
			PreMarketPrice: market.Currency{
				USD: 9,
			},
		},
		rating: market.RecommendationRating{
			StrongBuy: 1,
		},
		financialData: market.FinancialData{
			CurrentPrice: market.Currency{
				USD: 10,
			},
			TargetMeanPrice: market.Currency{
				USD: 15,
			},
		},
		expected: nil,
	},
	{
		name:   "Fail gainPercentage below threshold",
		symbol: "GNOG",
		price: market.Price{
			MarketChange: market.Percent{
				Percent: gainThreshold - 1,
			},
			PreMarketPrice: market.Currency{
				USD: 10,
			},
		},
		rating: market.RecommendationRating{
			StrongBuy: 1,
		},
		financialData: market.FinancialData{
			CurrentPrice: market.Currency{
				USD: 15,
			},
			TargetMeanPrice: market.Currency{},
		},
		expected: errors.New("GNOG gain:49.00 is not above threshold:50.00"),
	},
	{
		name:   "Fail preMarketPrice is above currentPrice",
		symbol: "GNOG",
		price: market.Price{
			MarketChange: market.Percent{
				Percent: gainThreshold,
			},
			PreMarketPrice: market.Currency{
				USD: 15,
			},
		},
		rating: market.RecommendationRating{
			StrongBuy: 1,
		},
		financialData: market.FinancialData{
			CurrentPrice: market.Currency{
				USD: 10,
			},
			TargetMeanPrice: market.Currency{},
		},
		expected: errors.New("GNOG preMarketPrice:15.00 is above currentPrice:10.00"),
	},
	{
		name:   "Fail no targetMeanPrice",
		symbol: "GNOG",
		price: market.Price{
			MarketChange: market.Percent{
				Percent: gainThreshold,
			},
			PreMarketPrice: market.Currency{
				USD: 10,
			},
		},
		rating: market.RecommendationRating{
			Buy: 1,
		},
		financialData: market.FinancialData{
			CurrentPrice: market.Currency{
				USD: 15,
			},
			TargetMeanPrice: market.Currency{},
		},
		expected: errors.New("GNOG targetMeanPrice:0.00 is less than fiftyPercentGainPrice:22.50 currentPrice:15.00"),
	},
	{
		name:   "Fail sell",
		symbol: "GNOG",
		price: market.Price{
			MarketChange: market.Percent{
				Percent: gainThreshold,
			},
			PreMarketPrice: market.Currency{
				USD: 9,
			},
		},
		rating: market.RecommendationRating{
			Sell: 1,
		},
		financialData: market.FinancialData{
			CurrentPrice: market.Currency{
				USD: 10,
			},
			TargetMeanPrice: market.Currency{
				USD: 15,
			},
		},
		expected: errors.New("GNOG has sell:1 strongSell:0 rating"),
	},
	{
		name:   "Fail strongSell",
		symbol: "GNOG",
		price: market.Price{
			MarketChange: market.Percent{
				Percent: gainThreshold,
			},
			PreMarketPrice: market.Currency{
				USD: 9,
			},
		},
		rating: market.RecommendationRating{
			StrongSell: 1,
		},
		financialData: market.FinancialData{
			CurrentPrice: market.Currency{
				USD: 10,
			},
			TargetMeanPrice: market.Currency{
				USD: 15,
			},
		},
		expected: errors.New("GNOG has sell:0 strongSell:1 rating"),
	},
	{
		name:   "Fail no buy or strongBuy",
		symbol: "GNOG",
		price: market.Price{
			MarketChange: market.Percent{
				Percent: gainThreshold,
			},
			PreMarketPrice: market.Currency{
				USD: 9,
			},
		},
		rating: market.RecommendationRating{},
		financialData: market.FinancialData{
			CurrentPrice: market.Currency{
				USD: 10,
			},
			TargetMeanPrice: market.Currency{
				USD: 15,
			},
		},
		expected: errors.New("GNOG has buy:0 strongBuy:0 rating"),
	},
}

func TestValidateAgainstTresholds(t *testing.T) {
	for _, tc := range validateAgainstTresholdsTCs {
		actual := validateAgainstTresholds(tc.symbol, tc.price, tc.rating, tc.financialData)

		if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", tc.expected) {
			t.Fatalf("%s expected: %v, actual: %v", tc.name, tc.expected, actual)
		}
	}
}
