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
	archiveTableName        string
	snsTopicArn             string
	gainThresholdPercentage float64
)

func init() {
	var err error

	tableName = os.Getenv("TABLE_NAME")
	archiveTableName = os.Getenv("ARCHIVE_TABLE_NAME")
	snsTopicArn = os.Getenv("SNS_TOPIC_ARN")
	threshold := os.Getenv("GAIN_THRESHOLD")
	if gainThresholdPercentage, err = strconv.ParseFloat(threshold, 64); err != nil {
		log.Println(err)
		gainThresholdPercentage = 50
	}
}

// getFilteredStocks - Filters the collection of symbols to those that meet the notificaiton criteria
// and returns a filtered collection of Stock structs
func getFilteredStocks(symbols []string) ([]notification.Stock, error) {
	var notifications []notification.Stock
	stockDataStore := data.New(tableName, archiveTableName)

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

			gainPercentage := price.MarketChange.Percent
			if gainPercentage < gainThresholdPercentage {
				errCh <- fmt.Errorf("%s gain:%.2f is not above threshold:%.2f", symbol, gainPercentage, gainThresholdPercentage)
				return
			}
			preMarketPrice := price.PreMarketPrice.USD
			if preMarketPrice > data.CurrentPrice.USD {
				errCh <- fmt.Errorf("%s preMarketPrice:%.2f is above currentPrice:%.2f", symbol, preMarketPrice, data.CurrentPrice.USD)
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
