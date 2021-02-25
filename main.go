package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/lancehumiston/stonk-lambda/data"
	"github.com/lancehumiston/stonk-lambda/market"
	"github.com/lancehumiston/stonk-lambda/notification"
	"github.com/lancehumiston/stonk-lambda/url"
)

var (
	tableName               string
	snsTopicArn             string
	gainThresholdPercentage float64
)

func init() {
	var err error

	tableName = os.Getenv("TABLE_NAME")
	snsTopicArn = os.Getenv("SNS_TOPIC_ARN")
	threshold := os.Getenv("GAIN_THRESHOLD")
	if gainThresholdPercentage, err = strconv.ParseFloat(threshold, 64); err != nil {
		log.Println(err)
		gainThresholdPercentage = 50
	}
}

// validateAgainstTresholds - Verifies that the symbol meets notification thresholds based on price and financialData
func validateAgainstTresholds(symbol string, price market.Price, rating market.RecommendationRating, financialData market.FinancialData) error {
	gainPercentage := price.MarketChange.Percent
	if gainPercentage < gainThresholdPercentage {
		return fmt.Errorf("%s gain:%.2f is not above threshold:%.2f", symbol, gainPercentage, gainThresholdPercentage)
	}
	log.Printf("%s gain:%.2f is above threshold:%.2f", symbol, gainPercentage, gainThresholdPercentage)

	preMarketPrice := price.PreMarketPrice.USD
	currentPrice := financialData.CurrentPrice.USD
	if preMarketPrice > currentPrice {
		return fmt.Errorf("%s preMarketPrice:%.2f is above currentPrice:%.2f", symbol, preMarketPrice, financialData.CurrentPrice.USD)
	}
	log.Printf("%s currentPrice:%.2f is above preMarketPrice:%.2f", symbol, financialData.CurrentPrice.USD, preMarketPrice)

	if rating.Sell > 0 || rating.StrongSell > 0 {
		return fmt.Errorf("%s has sell:%d strongSell:%d rating", symbol, rating.Sell, rating.StrongSell)
	}

	if rating.Buy == 0 && rating.StrongBuy == 0 {
		return fmt.Errorf("%s has buy:%d strongBuy:%d rating", symbol, rating.Buy, rating.StrongBuy)
	}

	targetMeanPrice := financialData.TargetMeanPrice.USD
	fiftyPercentGainPrice := currentPrice * 1.5
	if targetMeanPrice < fiftyPercentGainPrice {
		return fmt.Errorf("%s targetMeanPrice:%.2f is less than fiftyPercentGainPrice:%.2f currentPrice:%.2f", symbol, targetMeanPrice, fiftyPercentGainPrice, currentPrice)
	}

	return nil
}

// getFilteredStocks - Filters the collection of symbols to those that meet the notificaiton criteria
// and returns a filtered collection of Stock structs
func getFilteredStocks(symbols []string) ([]notification.Stock, error) {
	var notifications []notification.Stock
	stockDataStore := data.New(tableName)

	uniqueSymbols := unique(symbols)
	ch := make(chan notification.Stock, len(uniqueSymbols))
	errCh := make(chan error, cap(ch))
	for _, v := range uniqueSymbols {
		go func(symbol string, ch chan<- notification.Stock, errCh chan<- error) {
			price, rating, data, err := market.GetAnalysis(symbol)
			if err != nil {
				errCh <- err
				return
			}

			if err = validateAgainstTresholds(symbol, price, rating, data); err != nil {
				errCh <- err
				return
			}

			exists, err := stockDataStore.Exists(symbol)
			if err != nil {
				errCh <- err
				return
			}
			if exists {
				errCh <- fmt.Errorf("dynamodb record exists for %s", symbol)
				return
			}

			gainPercentage := price.MarketChange.Percent
			if err := stockDataStore.Insert(symbol, gainPercentage, data.CurrentPrice.USD); err != nil {
				errCh <- err
				return
			}

			companyName, err := market.GetCompanyName(symbol)
			if err != nil {
				errCh <- err
				return
			}

			newsURL, err := market.GetNews(companyName)
			if err != nil {
				errCh <- err
				return
			}
			shortenedNewsURL, err := url.GetShortenedAlias(newsURL)
			if err != nil {
				errCh <- err
				return
			}

			ch <- notification.Stock{
				Symbol:          symbol,
				Gain:            gainPercentage,
				CurrentPrice:    data.CurrentPrice.USD,
				TargetLowPrice:  data.TargetLowPrice.USD,
				TargetHighPrice: data.TargetHighPrice.USD,
				TargetMeanPrice: data.TargetMeanPrice.USD,
				StrongBuy:       rating.StrongBuy,
				Buy:             rating.Buy,
				Hold:            rating.Hold,
				Sell:            rating.Sell,
				StrongSell:      rating.StrongSell,
				NewsURL:         shortenedNewsURL,
			}
		}(v, ch, errCh)
	}

	for i := 0; i < cap(ch); i++ {
		select {
		case notification := <-ch:
			notifications = append(notifications, notification)
		case err := <-errCh:
			log.Println(err) // log and continue with data from other stocks
		}
	}

	return notifications, nil
}

func unique(items []string) []string {
	if items == nil || len(items) == 0 {
		return items
	}

	var uniqueItems []string
	set := make(map[string]struct{})
	for _, v := range items {
		if _, ok := set[v]; ok {
			continue
		}

		uniqueItems = append(uniqueItems, v)
		set[v] = struct{}{}
	}

	return uniqueItems
}

// lambdaHandler - Entry point
func lambdaHandler(ctx context.Context, event events.CloudWatchEvent) error {
	var symbols []string
	providers := market.GetTopMoversProviders()

	ch := make(chan []string, len(providers))
	errCh := make(chan error, cap(ch))
	for _, v := range providers {
		go func(provider market.TopMoversProvider, ch chan<- []string, errCh chan<- error) {
			topMovers, err := provider.GetTopMovers()
			if err != nil {
				errCh <- err
				return
			}

			ch <- topMovers
		}(v, ch, errCh)
	}

	for i := 0; i < cap(ch); i++ {
		select {
		case topMovers := <-ch:
			symbols = append(symbols, topMovers...)
		case err := <-errCh:
			log.Println(err) // log and continue with data from other providers
		}
	}

	filteredStocks, err := getFilteredStocks(symbols)
	if err != nil {
		return err
	}

	notification := notification.New(snsTopicArn)
	if err = notification.Send(filteredStocks); err != nil {
		return err
	}

	return nil
}

func main() {
	lambda.Start(lambdaHandler)
}
