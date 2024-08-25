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

#### Setup

1. **Clone the repository**
```
git clone <repository-url>
cd <repository-name>
```

2.**Build the lambda function**: Navigate to the `lambda` directory and compile Go lambda for linux env:
```
cd lambda
GOOS=linux go build -o main main.go
zip deployment.zip main
```

3. **Deploy the infrastructure**: Navigate to `infrastack` directory and run the following command
```
cd ../infrastack
pulumi up
```


## Components

### Infrastack


### Lambda Function

The Lambda function is implemented in Go. It is triggered by events in the S3 bucket, processes the CSV files, and sends the processed data as JSON messages.

### Improvements and Recommendations

- Implement schema validation for input CSV files to ensure data integrity
- Consider integrating a more permanent storage solution like Amazon RDS or DynamoDB for processed data
- Enhance error handling to manage and log unexpected inputs effectively

# aws-lambda-fullstack
