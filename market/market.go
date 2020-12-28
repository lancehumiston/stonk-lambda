package market

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	apiKey string
)

func init() {
	apiKey = os.Getenv("NEWS_API_KEY")
}

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

type instrumentResponse struct {
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

	var i instrumentResponse
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

type quoteResponse struct {
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

	var q quoteResponse
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

type newsResponse struct {
	Articles []struct {
		URL string `json:"url"`
	} `json:"articles"`
}

// GetNews - Returns URL of news articles related to the company
func GetNews(companyName string) ([]string, error) {
	var urls = []string{}
	if companyName == "" {
		return urls, nil
	}

	const pageSize int = 3
	noSuffix := regexp.MustCompile(`(?i)inc\.|(?i)Incorporated|(?i)plc|(?i)corporation|(?i)corp\.|(?i)limited|(?i)ltd\.`).ReplaceAllString(companyName, "")
	formattedQuery := "+" + strings.Replace(noSuffix, " ", "+", -1)
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	resp, err := http.Get(fmt.Sprintf("https://newsapi.org/v2/everything?qInTitle=%s&sortBy=publishedAt&pageSize=%d&apiKey=%s&from=%s&language=en", formattedQuery, pageSize, apiKey, yesterday))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GetNews failed with status: %d - %s", resp.StatusCode, body)
	}

	var n newsResponse
	json.Unmarshal(body, &n)
	log.Println(n)

	for _, v := range n.Articles {
		urls = append(urls, v.URL)
	}

	return urls, nil
}
