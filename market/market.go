package market

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var (
	newsAPIKey          string
	companySuffixRegexp *regexp.Regexp
)

func init() {
	newsAPIKey = os.Getenv("NEWS_API_KEY")
	companySuffixRegexp = regexp.MustCompile(`(?i)inc\.|(?i)Incorporated|(?i)plc|(?i)corporation|(?i)corp\.|(?i)limited|(?i)ltd\.`)
}

// TopMoversProvider - Provides a list of "top mover" stock symbols for the day
type TopMoversProvider interface {
	GetTopMovers() ([]string, error)
}

// GetTopMoversProviders - Returns a collection of TopMoversProvider
func GetTopMoversProviders() []TopMoversProvider {
	return []TopMoversProvider{
		&robinhood{},
		&financialModelingPrep{},
	}
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

// Currency - Currency data
type Currency struct {
	USD float64 `json:"raw"`
}

// Percent - Percent data
type Percent struct {
	Percent float64 `json:"raw"`
}

// FinancialData - Stock financial data
type FinancialData struct {
	CurrentPrice    Currency `json:"currentPrice"`
	TargetHighPrice Currency `json:"targetHighPrice"`
	TargetLowPrice  Currency `json:"targetLowPrice"`
	TargetMeanPrice Currency `json:"targetMeanPrice"`
}

// Price - Price data
type Price struct {
	MarketChange   Percent  `json:"regularMarketChangePercent"`
	PreMarketPrice Currency `json:"preMarketPrice"`
}

type quoteResponse struct {
	Summary struct {
		Result []struct {
			RecommendationTrend struct {
				Trend []RecommendationRating `json:"trend"`
			} `json:"recommendationTrend"`
			Price         Price         `json:"price"`
			FinancialData FinancialData `json:"financialData"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"quoteSummary"`
}

// GetAnalysis - Calculates the gain percentage from previous close to current and fetches analyst recommendation rating
func GetAnalysis(symbol string) (Price, RecommendationRating, FinancialData, error) {
	var rating RecommendationRating
	var data FinancialData
	var price Price

	resp, err := http.Get(fmt.Sprintf("https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?region=US&modules=recommendationTrend%%2Cprice%%2CfinancialData", symbol))
	if err != nil {
		return price, rating, data, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return price, rating, data, err
	}

	if resp.StatusCode == 404 {
		log.Printf("Symbol not found:%s", symbol)
		return price, rating, data, err
	}

	var q quoteResponse
	json.Unmarshal(body, &q)
	log.Println(q)

	if q.Summary.Error != nil {
		return price, rating, data, fmt.Errorf("%v", q.Summary.Error)
	}
	if len(q.Summary.Result) < 1 {
		return price, rating, data, nil
	}
	result := q.Summary.Result[0]

	result.Price.MarketChange.Percent = result.Price.MarketChange.Percent * 100
	data.CurrentPrice = result.FinancialData.CurrentPrice
	if len(result.RecommendationTrend.Trend) > 0 {
		rating = result.RecommendationTrend.Trend[0]
		data.TargetLowPrice = result.FinancialData.TargetLowPrice
		data.TargetHighPrice = result.FinancialData.TargetHighPrice
		data.TargetMeanPrice = result.FinancialData.TargetMeanPrice
	}

	return result.Price, rating, data, nil
}

type companyResponse struct {
	ResultSet struct {
		Result []struct {
			Symbol string `json:"symbol"`
			Name   string `json:"name"`
		} `json:"Result"`
	} `json:"ResultSet"`
}

// GetCompanyName - Gets the company name that the symbol is associated with
func GetCompanyName(symbol string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://autoc.finance.yahoo.com/autoc?lang=en&query=%s", symbol))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var c companyResponse
	json.Unmarshal(body, &c)
	log.Println(c)

	for _, v := range c.ResultSet.Result {
		if v.Symbol == symbol {
			return v.Name, nil
		}
	}

	return "", nil
}

// GetNews - Returns URL of news articles related to the company
func GetNews(companyName string) (string, error) {
	if companyName == "" {
		return "", errors.New("GetNews failed due to empty companyName")
	}

	noSuffix := companySuffixRegexp.ReplaceAllString(companyName, "")
	formattedQuery := "+" + strings.Replace(noSuffix, " ", "+", -1)
	return fmt.Sprintf("https://news.google.com/search?q=%s", formattedQuery), nil
}
