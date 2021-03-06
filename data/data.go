package data

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type data struct {
	TableName string
}

// New - Public constructor for data
func New(tableName string) *data {
	if tableName == "" {
		log.Panic("tableName cannot be empty")
	}

	return &data{
		TableName: tableName,
	}
}

// Exists - Determines if a record for the symbol exists in the data store
func (d *data) Exists(symbol string) (bool, error) {
	svc := dynamodb.New(session.New())
	input := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Symbol": {
				S: aws.String(symbol),
			},
		},
		TableName: aws.String(d.TableName),
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

// Insert - Inserts the stock into the short-lived cache data store
func (d *data) Insert(symbol string, percentage float64, price float64) error {
	item := struct {
		Symbol       string
		Percentage   float64
		Price        float64
		CreatedAtUtc int64
		TTL          int64
	}{
		Symbol:       symbol,
		Percentage:   percentage,
		Price:        price,
		CreatedAtUtc: time.Now().UTC().Unix(),
		TTL:          getItemTTL(time.Now().UTC()),
	}

	svc := dynamodb.New(session.New())

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(d.TableName),
	}

	if _, err := svc.PutItem(input); err != nil {
		return err
	}

	return nil
}

// getItemTTL - Returns the epoch value for 2am tomorrow in UTC
func getItemTTL(t time.Time) int64 {
	year, month, day := t.Date()
	return time.Date(year, month, day+1, 2, 0, 0, 0, time.UTC).Unix()
}
