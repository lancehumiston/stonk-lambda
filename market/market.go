package market

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type movers struct {
	InstrumentURIs []string `json:"instruments"`
}

// GetTopMovers - Returns a list of URIs associated with Robinhood's "Top Movers" list
func GetTopMovers() ([]string, error) {
	resp, err := http.Get("https://api.robinhood.com/midlands/tags/tag/top-movers/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var m movers
	json.Unmarshal(body, &m)
	log.Println(m)

	return m.InstrumentURIs, nil
}

type instrument struct {
	Symbol string `json:"symbol"`
}

// GetSymbol - Returns ticker symbol associated with the instrumentURI
func GetSymbol(instrumentURI string) (string, error) {
	resp, err := http.Get(instrumentURI)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var i instrument
	json.Unmarshal(body, &i)
	log.Println(i)

	return i.Symbol, nil
}

// RecommendationRating - Stock analyst recommendation rating
type RecommendationRating struct {
	Period     string `json:"period"`
	StrongBuy  int64  `json:"strongBuy"`
	Buy        int64  `json:"buy"`
	Hold       int64  `json:"hold"`
	Sell       int64  `json:"sell"`
	StrongSell int64  `json:"strongSell"`
}

// FinancialData - Stock financial data
type FinancialData struct {
	CurrentPrice struct {
		USD float64 `json:"raw"`
	} `json:"currentPrice"`
	TargetHighPrice struct {
		USD float64 `json:"raw"`
	} `json:"targetHighPrice"`
	TargetLowPrice struct {
		USD float64 `json:"raw"`
	} `json:"targetLowPrice"`
	TargetMeanPrice struct {
		USD float64 `json:"raw"`
	} `json:"targetMeanPrice"`
}

type quoteSummary struct {
	Summary struct {
		Result []struct {
			RecommendationTrend struct {
				Trend []RecommendationRating `json:"trend"`
			} `json:"recommendationTrend"`
			Price struct {
				MarketChange struct {
					Percent float64 `json:"raw"`
				} `json:"regularMarketChangePercent"`
			} `json:"price"`
			FinancialData FinancialData `json:"financialData"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"quoteSummary"`
}

// GetAnalysis - Calculates the gain percentage from previous close to current and fetches analyst recommendation rating
func GetAnalysis(symbol string) (float64, RecommendationRating, FinancialData, error) {
	var rating RecommendationRating
	var data FinancialData

	resp, err := http.Get(fmt.Sprintf("https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?region=US&modules=recommendationTrend%%2Cprice%%2CfinancialData", symbol))
	if err != nil {
		return 0, rating, data, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, rating, data, err
	}

	if resp.StatusCode == 404 {
		log.Printf("Symbol not found:%s", symbol)
		return 0, rating, data, err
	}

	var q quoteSummary
	json.Unmarshal(body, &q)
	log.Println(q)

	if q.Summary.Error != nil {
		return 0, rating, data, fmt.Errorf("%v", q.Summary.Error)
	}
	if len(q.Summary.Result) < 1 {
		return 0, rating, data, nil
	}
	result := q.Summary.Result[0]

	gain := result.Price.MarketChange.Percent
	data.CurrentPrice = result.FinancialData.CurrentPrice
	if len(result.RecommendationTrend.Trend) > 0 {
		rating = result.RecommendationTrend.Trend[0]
		data.TargetLowPrice = result.FinancialData.TargetLowPrice
		data.TargetHighPrice = result.FinancialData.TargetHighPrice
		data.TargetMeanPrice = result.FinancialData.TargetMeanPrice
	}

	return gain * 100, rating, data, nil
}
