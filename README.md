# Simple scalable data processing pipeline for AWS Environment

### Background


### Description
This project creates a serverless data processing pipeline. It reads CSV files from S3 bucket, processes them, and outputs the data as JSON messages to a queue

### Underlying tech
The pipeline utilizes AWS Lambda and is orchestrated using Pulumi with Golang


## Prerequisites
Before you can run this project, you need to have the following tools installed:
- [Pulumi](https://www.pulumi.com/docs/get-started/install/)
- [Golang](https://golang.org/dl/)
- [AWS CLI](https://aws.amazon.com/cli/)


#### Cloning the repository
```
git clone 
```

#### Authenticate to Providers
- use aws credentials to authenticate
```
export AWS_ACCESS_KEY_ID="<YOUR_ACCESS_KEY_ID>"
export AWS_SECRET_ACCESS_KEY="<YOUR_SECRET_ACCESS_KEY>"
```

- or you can configure your aws cli with a profile
```
aws configure
```
provide your access and secret credentials in configuration process

- then use this profile whenever you need
```
export AWS_PROFILE="<YOUR_PROFILE_NAME"
```

that was all for aws side, now for the pulumi part,

```
pulumi login
```

and after you authenticated to pulumi, we are all good

## Provisioning Infrastructure

1. **Clone the repository**
```
git clone https://github.com/ugurcancaykara/aws-lambda-fullstack
cd aws-lambda-fullstack
```

2.**Build the lambda function**: Run make command and compile Go lambda for linux env:
```
make build
```

3. **Deploy the infrastructure**: Navigate to `infrastack` directory and run the following command
```
cd infrastack
pulumi up
```

## Producing Messages

### Assign values to variables

```
# Retrieve the S3 bucket name
S3_BUCKET_NAME=$(pulumi stack output s3BucketName)

# Retrieve the SQS queue URL
SQS_QUEUE_URL=$(pulumi stack output sqsQueueUrl)
```

### Triggering lamba invocation with s3 notifications

```
# Upload customers_20230828.csv
aws s3 cp customers_20230828.csv s3://$S3_BUCKET_NAME/customers_20230828.csv

# Upload orders_20230828.csv
aws s3 cp orders_20230828.csv s3://$S3_BUCKET_NAME/orders_20230828.csv

# Upload items_20230828.csv
aws s3 cp items_20230828.csv s3://$S3_BUCKET_NAME/items_20230828.csv
```

## Checking Messages

### Verifying SQS Message
```
aws sqs receive-message --queue-url $SQS_QUEUE_URL --max-number-of-messages 10 --wait-time-seconds 10 --region eu-west-1
```

### Clean-Up Commands

Delete files from S3
```
aws s3 rm s3://$S3_BUCKET_NAME/customers_20230828.csv
aws s3 rm s3://$S3_BUCKET_NAME/orders_20230828.csv
aws s3 rm s3://$S3_BUCKET_NAME/items_20230828.csv
```


Purge SQS queue
```
aws sqs purge-queue --queue-url $SQS_QUEUE_URL --region eu-west-1
```



## Components

### Infrastack


### Lambda Function

The Lambda function is implemented in Go. It is triggered by events in the S3 bucket, processes the CSV files, and sends the processed data as JSON messages.

### Improvements and Recommendations

- Implement schema validation for input CSV files to ensure data integrity
- Consider integrating a more permanent storage solution like Amazon RDS or DynamoDB for processed data
- Enhance error handling to manage and log unexpected inputs effectively



##### References
[Deploying Go Lambda](https://docs.aws.amazon.com/lambda/latest/dg/golang-package.html)
[best practices](https://docs.aws.amazon.com/lambda/latest/dg/best-practices.html)
[migrating from 1.x to os-only](https://aws.amazon.com/blogs/compute/migrating-aws-lambda-functions-from-the-go1-x-runtime-to-the-custom-runtime-on-amazon-linux-2/)

