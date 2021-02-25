package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var (
	archiveTableName string
)

func init() {
	archiveTableName = os.Getenv("ARCHIVE_TABLE_NAME")
}

// insertArchive - Inserts a record for the stock into the long-lived archive data store
func insertArchive(symbol string, price float64) error {
	item := struct {
		Symbol       string
		Price        float64
		CreatedAtUtc int64
	}{
		Symbol:       symbol,
		Price:        price,
		CreatedAtUtc: time.Now().UTC().Unix(),
	}

	svc := dynamodb.New(session.New())

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:                av,
		TableName:           aws.String(archiveTableName),
		ConditionExpression: aws.String("attribute_not_exists(Symbol)"), // don't overwrite if `Symbol` exists
	}

	_, err = svc.PutItem(input)
	if err == nil {
		return nil
	}
	aerr, ok := err.(awserr.Error)
	if !ok || aerr.Code() != dynamodb.ErrCodeConditionalCheckFailedException {
		return err
	}

	return nil
}

// lambdaHandler - Entry point
func lambdaHandler(ctx context.Context, record events.DynamoDBEvent) error {
	for _, v := range record.Records {
		i := v.Change.NewImage

		s, ok := i["Symbol"]
		if !ok {
			log.Printf("%v did not contain valid 'Symbol'", i)
			continue
		}
		symbol := s.String()

		p, ok := i["Price"]
		if !ok {
			log.Printf("%v did not contain valid 'Price'", i)
			continue
		}
		price, err := strconv.ParseFloat(p.Number(), 64)
		if err != nil {
			log.Print(err) // log and continue to process batch
			continue
		}

		if err = insertArchive(symbol, price); err != nil {
			log.Print(err) // log and continue to process batch
		}
	}

	return nil
}

func main() {
	lambda.Start(lambdaHandler)
}
