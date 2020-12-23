package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/sns"
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

// stockExists - Determines if a record for the stock exists in the data store
func stockExists(symbol string) (bool, error) {
	svc := dynamodb.New(session.New())
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Symbol": {
				S: aws.String(symbol),
			},
		},
		TableName: aws.String(tableName),
	}

	result, err := svc.GetItem(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				log.Print(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Print(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeRequestLimitExceeded:
				log.Print(dynamodb.ErrCodeRequestLimitExceeded, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				log.Print(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				log.Print(aerr.Error())
			}
		} else {
			log.Print(err.Error())
		}
	}
	if result.Item == nil {
		return false, err
	}

	if *result.Item["Symbol"].S == symbol {
		return true, nil
	}
	return false, fmt.Errorf("Symbol %s returned wrong record %v", symbol, result.Item)
}

type movers struct {
	InstrumentURIs []string `json:"instruments"`
}

// getTopMovers - Returns a list of URIs associated with Robinhood's "Top Movers" list
func getTopMovers() ([]string, error) {
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

// getSymbol - Returns ticker symbol associated with the instrumentURI
func getSymbol(instrumentURI string) (string, error) {
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

type recommendationRating struct {
	Period     string `json:"period"`
	StrongBuy  int64  `json:"strongBuy"`
	Buy        int64  `json:"buy"`
	Hold       int64  `json:"hold"`
	Sell       int64  `json:"sell"`
	StrongSell int64  `json:"strongSell"`
}

type financialData struct {
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
				Trend []recommendationRating `json:"trend"`
			} `json:"recommendationTrend"`
			Price struct {
				MarketChange struct {
					Percent float64 `json:"raw"`
				} `json:"regularMarketChangePercent"`
			} `json:"price"`
			FinancialData financialData `json:"financialData"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"quoteSummary"`
}

// getAnalysis - Calculates the gain percentage from previous close to current and fetches analyst recommendation rating
func getAnalysis(symbol string) (float64, recommendationRating, financialData, error) {
	var rating recommendationRating
	var data financialData

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

// getItemTTL - Returns the epoch value for 2am tomorrow in UTC
func getItemTTL(t time.Time) int64 {
	year, month, day := t.Date()
	return time.Date(year, month, day+1, 2, 0, 0, 0, time.UTC).Unix()
}

// insertStock - Inserts the stock into the data store
func insertStock(symbol string, percentage float64, ttl int64) error {
	item := struct {
		Symbol     string
		Percentage float64
		TTL        int64
	}{
		Symbol:     symbol,
		Percentage: percentage,
		TTL:        ttl,
	}

	svc := dynamodb.New(session.New())

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	}

	result, err := svc.PutItem(input)
	if err != nil {
		return err
	}

	log.Println(result)

	return nil
}

type notification struct {
	Symbol          string  `json:"symbol"`
	Gain            float64 `json:"gain"`
	CurrentPrice    float64 `json:"currentPrice"`
	TargetHighPrice float64 `json:"targetHighPrice"`
	TargetLowPrice  float64 `json:"targetLowPrice"`
	TargetMeanPrice float64 `json:"targetMeanPrice"`
	StrongBuy       int64   `json:"strongBuy"`
	Buy             int64   `json:"buy"`
	Hold            int64   `json:"hold"`
	Sell            int64   `json:"sell"`
	StrongSell      int64   `json:"strongSell"`
}

// getFilteredNotifications - Filters the collection of stockURIs to those that meet the notificaiton criteria
// and returns a filtered collection of notification structs
func getFilteredNotifications(stockURIs []string) ([]notification, error) {
	var notifications []notification
	ttl := getItemTTL(time.Now().UTC())

	for _, stock := range stockURIs {
		symbol, err := getSymbol(stock)
		if err != nil {
			return nil, err
		}

		gain, rating, data, err := getAnalysis(symbol)
		if err != nil {
			return nil, err
		}

		if gain < gainThresholdPercentage {
			continue
		}

		exists, err := stockExists(symbol)
		if err != nil {
			return nil, err
		}
		if exists {
			log.Printf("dynamodb record exists for %s", symbol)
			continue
		}

		if err := insertStock(symbol, gain, ttl); err != nil {
			return nil, err
		}

		notification := notification{
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

// sendNotification - Sends the collection of notification structs as a sms via sns
func sendNotification(notifications []notification) error {
	if len(notifications) == 0 {
		return nil
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})

	if err != nil {
		return err
	}

	client := sns.New(sess)

	var sb strings.Builder
	sb.WriteString("🚀🚀🚀\n\n")
	for _, n := range notifications {
		sb.WriteString(fmt.Sprintf(
			"Symbol: %s\nGainz: %.2f%%\nCurrentPrice: %.2f\nTargetHigh: %.2f\nTargetLow: %.2f\nTargetMean: %.2f\nStrongBuy: %d\nBuy: %d\nHold: %d\nSell: %d\nStrongSell: %d\n\n",
			n.Symbol, n.Gain, n.CurrentPrice, n.TargetHighPrice, n.TargetLowPrice, n.TargetMeanPrice, n.StrongBuy, n.Buy, n.Hold, n.Sell, n.StrongSell))
	}
	input := &sns.PublishInput{
		Message:  aws.String(sb.String()),
		TopicArn: aws.String(snsTopicArn),
	}

	result, err := client.Publish(input)
	if err != nil {
		return err
	}

	log.Println(result)

	return nil
}

// lambdaHandler - Entry point
func lambdaHandler(ctx context.Context, event events.CloudWatchEvent) error {
	log.Println(event)

	stockURIs, err := getTopMovers()
	if err != nil {
		return err
	}

	filteredStocks, err := getFilteredNotifications(stockURIs)
	if err != nil {
		return err
	}

	if err = sendNotification(filteredStocks); err != nil {
		return err
	}

	return nil
}

func main() {
	lambda.Start(lambdaHandler)
}
