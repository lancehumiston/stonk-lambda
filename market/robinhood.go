package market

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type robinhood struct{}

type moversResponse struct {
	InstrumentURIs []string `json:"instruments"`
}

// GetTopMovers - Implementation of the TopMoversProvider interface
func (r *robinhood) GetTopMovers() ([]string, error) {
	urls, err := r.getTopMoversInstrumentIds()
	if err != nil {
		return nil, err
	}

	var symbols []string
	for _, v := range urls {
		symbol, err := r.getSymbol(v)
		if err != nil {
			return nil, err
		}

		symbols = append(symbols, symbol)
	}

	return symbols, nil
}

// getTopMoversInstrumentIds - Returns a collection of URIs associated with Robinhood's "Top Movers" list
func (r *robinhood) getTopMoversInstrumentIds() ([]string, error) {
	resp, err := http.Get("https://api.robinhood.com/midlands/tags/tag/top-movers/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var m moversResponse
	json.Unmarshal(body, &m)
	log.Println(m)

	return m.InstrumentURIs, nil
}

type instrumentResponse struct {
	Symbol string `json:"symbol"`
}

// getSymbol - Returns ticker symbol associated with the instrumentURI
func (r *robinhood) getSymbol(instrumentURI string) (string, error) {
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
