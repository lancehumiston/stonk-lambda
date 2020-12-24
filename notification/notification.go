package notification

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

// Stock - Stock overview for messaging
type Stock struct {
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

type notification struct {
	SnsTopicArn string
}

// New - Public constructor for notification
func New(snsTopicArn string) *notification {
	if snsTopicArn == "" {
		log.Panic("snsTopicArn cannot be empty")
	}

	return &notification{
		SnsTopicArn: snsTopicArn,
	}
}

// Send - Sends the collection of stocks as a SMS via AWS's SNS
func (n *notification) Send(stocks []Stock) error {
	if len(stocks) == 0 {
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
	for _, s := range stocks {
		sb.WriteString(fmt.Sprintf(`
Symbol: %s
Gainz: %.2f%%
CurrentPrice: %.2f
TargetHigh: %.2f
TargetLow: %.2f
TargetMean: %.2f
StrongBuy: %d
Buy: %d
Hold: %d
Sell: %d
StrongSell: %d
https://robinhood.com/stocks/%s
`,
			s.Symbol,
			s.Gain,
			s.CurrentPrice,
			s.TargetHighPrice,
			s.TargetLowPrice,
			s.TargetMeanPrice,
			s.StrongBuy,
			s.Buy,
			s.Hold,
			s.Sell,
			s.StrongSell,
			s.Symbol))
	}
	input := &sns.PublishInput{
		Message:  aws.String(sb.String()),
		TopicArn: aws.String(n.SnsTopicArn),
	}

	result, err := client.Publish(input)
	if err != nil {
		return err
	}

	log.Println(result)

	return nil
}
