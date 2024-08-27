package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type Customer struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	TotalSpent float64 `json:"total_amount_spent"`
	Orders     []Order `json:"orders"`
}

type Order struct {
	ID      string   `json:"id"`
	ItemIDs []string `json:"item_ids"`
	Amount  float64  `json:"amount"`
}

func handler(ctx context.Context, s3Event events.S3Event) {
	sess := session.Must(session.NewSession())
	s3svc := s3.New(sess)
	sqsSvc := sqs.New(sess)
	queueUrl := os.Getenv("SQS_QUEUE")

	customers := make(map[string]*Customer)

	for _, record := range s3Event.Records {
		s3Record := record.S3
		result, err := s3svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(s3Record.Bucket.Name),
			Key:    aws.String(s3Record.Object.Key),
		})
		if err != nil {
			log.Printf("Unable to download file, %v", err)
			sendErrorMessage(sqsSvc, queueUrl, fmt.Sprintf("Failed to download file: %v", err))
			continue
		}

		csvReader := csv.NewReader(result.Body)
		switch {
		case s3Record.Object.Key == "customers.csv":
			processCustomers(csvReader, customers)
		case s3Record.Object.Key == "orders.csv":
			processOrders(csvReader, customers)
		case s3Record.Object.Key == "items.csv":
			processItems(csvReader, customers)
		default:
			sendErrorMessage(sqsSvc, queueUrl, fmt.Sprintf("Unexpected file: %s", s3Record.Object.Key))
		}
	}

	// Send customer data to SQS
	for _, customer := range customers {
		jsonData, _ := json.Marshal(customer)
		_, err := sqsSvc.SendMessage(&sqs.SendMessageInput{
			MessageBody: aws.String(string(jsonData)),
			QueueUrl:    &queueUrl,
		})
		if err != nil {
			log.Printf("Failed to send customer message: %v", err)
		}
	}
}

func processCustomers(reader *csv.Reader, customers map[string]*Customer) {
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading customer data: %v", err)
			continue
		}

		id := line[0]
		name := line[1]
		customers[id] = &Customer{ID: id, Name: name, Orders: []Order{}}
	}
}

func processOrders(reader *csv.Reader, customers map[string]*Customer) {
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading order data: %v", err)
			continue
		}

		customerID := line[1]
		orderID := line[0]
		amount, err := strconv.ParseFloat(line[2], 64)
		if err != nil {
			log.Printf("Error parsing order amount: %v", err)
			continue
		}

		if customer, exists := customers[customerID]; exists {
			customer.Orders = append(customer.Orders, Order{ID: orderID, Amount: amount})
			customer.TotalSpent += amount
		}
	}
}

func processItems(reader *csv.Reader, customers map[string]*Customer) {
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading item data: %v", err)
			continue
		}

		orderID := line[1]
		itemID := line[0]

		for _, customer := range customers {
			for i := range customer.Orders {
				if customer.Orders[i].ID == orderID {
					customer.Orders[i].ItemIDs = append(customer.Orders[i].ItemIDs, itemID)
				}
			}
		}
	}
}

func sendErrorMessage(sqsSvc *sqs.SQS, queueUrl, message string) {
	_, err := sqsSvc.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(message),
		QueueUrl:    &queueUrl,
	})
	if err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}

func main() {
	lambda.Start(handler)
}
