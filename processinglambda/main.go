package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func handler(ctx context.Context, s3Event events.S3Event) {
	sess := session.Must(session.NewSession())
	s3svc := s3.New(sess)
	sqsSvc := sqs.New(sess)
	queueUrl := os.Getenv("SQS_QUEUE")

	for _, record := range s3Event.Records {
		s3Record := record.S3
		result, err := s3svc.GetObject(&s3.GetObjectInput{Bucket: aws.String(s3Record.Bucket.Name),
			Key: aws.String(s3Record.Object.Key),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to download file, %v", err)
			continue
		}

		csvReader := csv.NewReader(result.Body)
		for {
			line, error := csvReader.Read()
			if error != nil {
				break
			}
			jsonData, _ := json.Marshal(line)
			_, err := sqsSvc.SendMessage(&sqs.SendMessageInput{
				MessageBody: aws.String(string(jsonData)),
				QueueUrl:    &queueUrl,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to send message: %v", err)
			}
		}
	}
}

func main() {
	lambda.Start(handler)
}
