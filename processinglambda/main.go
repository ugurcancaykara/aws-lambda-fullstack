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
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type Customer struct {
	ID         string  `json:"id" dynamodbav:"ID"`
	Name       string  `json:"name" dynamodbav:"Name"`
	TotalSpent float64 `json:"total_amount_spent" dynamodbav:"TotalSpent"`
	Orders     []Order `json:"orders" dynamodbav:"Orders"`
}
type Order struct {
	ID      string   `json:"id" dynamodbav:"ID"`            // Maps to order_id in CSV
	Amount  float64  `json:"amount" dynamodbav:"Amount"`    // Maps to amount in CSV
	ItemIDs []string `json:"item_ids" dynamodbav:"ItemIDs"` // This field will be updated when processing items
}

func handler(ctx context.Context, s3Event events.S3Event) {
	sess := session.Must(session.NewSession())
	s3svc := s3.New(sess)
	sqsSvc := sqs.New(sess)
	dynamoDBSvc := dynamodb.New(sess)
	queueUrl := os.Getenv("SQS_QUEUE")
	tableName := os.Getenv("DYNAMODB_TABLE")

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
		case strings.HasPrefix(s3Record.Object.Key, "customers_"):
			processCustomers(csvReader, dynamoDBSvc, tableName)
		case strings.HasPrefix(s3Record.Object.Key, "orders_"):
			processOrders(csvReader, dynamoDBSvc, tableName)
		case strings.HasPrefix(s3Record.Object.Key, "items_"):
			processItems(csvReader, dynamoDBSvc, tableName)
			// After processing, send customer data to SQS
			customers, err := getAllCustomersFromDynamoDB(dynamoDBSvc, tableName)
			if err != nil {
				log.Printf("Failed to retrieve customers: %v", err)
				return
			}

			for _, customer := range customers {
				sendCustomerToSQS(sqsSvc, queueUrl, customer)
			}

		default:
			sendErrorMessage(sqsSvc, queueUrl, fmt.Sprintf("Unexpected file: %s", s3Record.Object.Key))
		}
	}

}

func sendCustomerToSQS(sqsSvc *sqs.SQS, queueUrl string, customer Customer) {
	jsonData, err := json.Marshal(customer)
	if err != nil {
		log.Printf("Failed to marshal customer data to JSON: %v", err)
		return
	}

	_, err = sqsSvc.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(jsonData)),
		QueueUrl:    aws.String(queueUrl),
	})
	if err != nil {
		log.Printf("Failed to send customer data to SQS: %v", err)
	} else {
		log.Printf("Successfully sent customer data to SQS: %s", customer.ID)
	}
}

func processCustomers(reader *csv.Reader, dynamoDBSvc *dynamodb.DynamoDB, tableName string) {
	// Skip the header row
	_, err := reader.Read()
	if err != nil {
		log.Printf("Error reading header row: %v", err)
		return
	}

	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading customer data: %v", err)
			continue
		}

		id := line[0]   // Assuming the first column is customer_id
		name := line[1] // Assuming the second column is name

		customer := &Customer{ID: id, Name: name, Orders: []Order{}}

		av, err := dynamodbattribute.MarshalMap(customer)
		if err != nil {
			log.Printf("Failed to marshal customer data: %v", err)
			continue
		}

		// Log the marshaled attribute value
		log.Printf("Marshalled attribute value: %v", av)

		fmt.Println("this is customer id: ", customer.ID)
		fmt.Println("this is customer name: ", customer.Name)
		_, err = dynamoDBSvc.PutItem(&dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      av,
		})
		if err != nil {
			log.Printf("Failed to save customer data to DynamoDB: %v", err)
		}
	}
}
func processOrders(reader *csv.Reader, dynamoDBSvc *dynamodb.DynamoDB, tableName string) {
	// Skip the header row
	_, err := reader.Read()
	if err != nil {
		log.Printf("Error reading header row: %v", err)
		return
	}

	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading order data: %v", err)
			continue
		}

		customerID := line[1] // Assuming the second column is customer_id
		orderID := line[0]    // Assuming the first column is order_id
		amount, err := strconv.ParseFloat(line[2], 64)
		if err != nil {
			log.Printf("Error parsing order amount: %v", err)
			continue
		}

		customer, err := getCustomerFromDynamoDB(customerID, dynamoDBSvc, tableName)
		if err != nil {
			log.Printf("Failed to retrieve customer data: %v", err)
			continue
		}

		customer.Orders = append(customer.Orders, Order{ID: orderID, Amount: amount})
		customer.TotalSpent += amount

		err = saveCustomerToDynamoDB(customer, dynamoDBSvc, tableName)
		if err != nil {
			log.Printf("Failed to update customer data in DynamoDB: %v", err)
		}
	}
}
func processItems(reader *csv.Reader, dynamoDBSvc *dynamodb.DynamoDB, tableName string) {
	// Skip the header row
	_, err := reader.Read()
	if err != nil {
		log.Printf("Error reading header row: %v", err)
		return
	}

	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading item data: %v", err)
			continue
		}

		orderID := line[1] // Assuming the second column is order_id
		itemID := line[0]  // Assuming the first column is item_id

		// Retrieve customers and update orders
		customers, err := getAllCustomersFromDynamoDB(dynamoDBSvc, tableName)
		if err != nil {
			log.Printf("Failed to retrieve customers: %v", err)
			continue
		}

		for _, customer := range customers {
			for i := range customer.Orders {
				if customer.Orders[i].ID == orderID {
					customer.Orders[i].ItemIDs = append(customer.Orders[i].ItemIDs, itemID)
					err = saveCustomerToDynamoDB(&customer, dynamoDBSvc, tableName)
					if err != nil {
						log.Printf("Failed to update customer data in DynamoDB: %v", err)
					}
				}
			}
		}
	}
}
func getCustomerFromDynamoDB(customerID string, svc *dynamodb.DynamoDB, tableName string) (*Customer, error) {
	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(customerID),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	customer := new(Customer)
	err = dynamodbattribute.UnmarshalMap(result.Item, customer)
	if err != nil {
		return nil, err
	}

	return customer, nil
}

func getAllCustomersFromDynamoDB(svc *dynamodb.DynamoDB, tableName string) ([]Customer, error) {
	var customers []Customer

	result, err := svc.Scan(&dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return nil, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &customers)
	if err != nil {
		return nil, err
	}

	return customers, nil
}

func saveCustomerToDynamoDB(customer *Customer, svc *dynamodb.DynamoDB, tableName string) error {
	if customer.ID == "" {
		return fmt.Errorf("customer ID is missing")
	}

	av, err := dynamodbattribute.MarshalMap(customer)
	if err != nil {
		return fmt.Errorf("got error marshalling map: %v", err)
	}

	log.Printf("Saving customer with ID: %s", customer.ID)
	_, err = svc.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      av,
	})
	return err
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
