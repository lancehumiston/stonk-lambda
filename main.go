package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/lancehumiston/stonk-lambda/data"
	"github.com/lancehumiston/stonk-lambda/market"
	"github.com/lancehumiston/stonk-lambda/notification"
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

// getFilteredStocks - Filters the collection of stockURIs to those that meet the notificaiton criteria
// and returns a filtered collection of Stock structs
func getFilteredNotifications(stockURIs []string) ([]notification.Stock, error) {
	var notifications []notification.Stock
	stockRepo := data.New(tableName)

	for _, stock := range stockURIs {
		symbol, err := market.GetSymbol(stock)
		if err != nil {
			return nil, err
		}

		gain, rating, data, err := market.GetAnalysis(symbol)
		if err != nil {
			return nil, err
		}

		if gain < gainThresholdPercentage {
			continue
		}

		exists, err := stockRepo.Exists(symbol)
		if err != nil {
			return nil, err
		}
		if exists {
			log.Printf("dynamodb record exists for %s", symbol)
			continue
		}

		if err := stockRepo.Insert(symbol, gain); err != nil {
			return nil, err
		}

		notification := notification.Stock{
			Symbol:          symbol,
			Gain:            gain,
			CurrentPrice:    data.CurrentPrice.USD,
			TargetLowPrice:  data.TargetLowPrice.USD,
			TargetHighPrice: data.TargetHighPrice.USD,
			TargetMeanPrice: data.TargetMeanPrice.USD,
			StrongBuy:       rating.StrongBuy,
			Buy:             rating.Buy,
			Hold:            rating.Hold,
			Sell:            rating.Sell,
			StrongSell:      rating.StrongSell,
		}
		notifications = append(notifications, notification)
	}

	return notifications, nil
}

// lambdaHandler - Entry point
func lambdaHandler(ctx context.Context, event events.CloudWatchEvent) error {
	log.Println(event)

	stockURIs, err := market.GetTopMovers()
	if err != nil {
		return err
	}

	filteredStocks, err := getFilteredNotifications(stockURIs)
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
