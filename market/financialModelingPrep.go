package market

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	financialModelingPrepAPIKey string
)

func init() {
	financialModelingPrepAPIKey = os.Getenv("FIN_MODELING_API_KEY")

	// Check if the key should be rotated out to avoid api rate limit later in the day
	now := time.Now().UTC()
	apiKeyRotationTime := time.Date(now.Year(), now.Month(), now.Day(), 18, 30, 0, 0, time.UTC)
	if now.After(apiKeyRotationTime) {
		log.Println("ApiKey rotation time, using FIN_MODELING_API_KEY_BACKUP")
		financialModelingPrepAPIKey = os.Getenv("FIN_MODELING_API_KEY_BACKUP")
	}
}

type financialModelingPrep struct{}

type tickerResponse struct {
	Ticker string `json:"ticker"`
}

// GetTopMovers - Implementation of the TopMoversProvider interface
func (f *financialModelingPrep) GetTopMovers() ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("https://financialmodelingprep.com/api/v3/gainers?apikey=%s", financialModelingPrepAPIKey))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	a := string(body)
	log.Print(a)

	var r []tickerResponse
	json.Unmarshal(body, &r)
	log.Println(r)

	var symbols []string
	for _, v := range r {
		symbols = append(symbols, v.Ticker)
	}

	return symbols, nil
}
