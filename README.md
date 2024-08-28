# Basic Data Processing Pipeline

### Description
This project creates a serverless data processing pipeline. It reads CSV files from S3 bucket, processes them, and outputs the data as JSON messages to a queue

### Underlying tech
The pipeline utilizes AWS Lambda, DynamoDB, SQS, S3 and are orchestrated using Pulumi with Golang

## Background

I have been interested with pulumi and wanted to test it for a while :) So this is also an attempt to test pulumi for IaC practices. I just don't want to make up and running IaC but I want to make research and focus on is it really applicable at large scale?, is it mature enough to use? 

To answer those questions, I wanted to make a [workshop](https://github.com/ugurcancaykara/pulumi-workshop). It will cover all the topics that are important for IaC best practices and how you can make those happen with `pulumi`.

So let's continue from where we left :) 

### Serverless Setup Using AWS Lambda

The problem of processing CSV files uploaded to an S3 bucket and generating JSON messages to be sent to an SQS queue can be effectively solved using a serverless architecture with the following AWS services:
 - **AWS S3:** The partner uploads CSV files (e.g., customers, orders, items) to a shared S3 bucket. S3 acts as the storage solution for these files and automatically triggers events when new files are uploaded.

 - **AWS Lambda:** Lambda functions are triggered by the S3 events generated when files are uploaded. The Lambda function reads the contents of these files, processes the data to generate customer messages with aggregated information such as total_amount_spent, and sends these messages to an SQS queue.

 - **AWS SQS:** SQS acts as a decoupling mechanism that receives and queues the JSON messages generated by the Lambda function. This allows downstream systems to process these messages asynchronously and at their own pace.

Flow:

1. The partner uploads CSV files to the S3 bucket.
2. S3 triggers an event for each file upload, which invokes the Lambda function.
3. The Lambda function reads the file from S3, processes the data, and generates JSON messages.
4. The Lambda function sends each JSON message to the SQS queue.
5. Downstream systems can consume messages from the SQS queue as needed.

### Problem and Solution Discussion

The problem involves processing multiple CSV files (customers_*.csv, orders_*.csv, and items_*.csv) that are uploaded to S3, combining this data, and sending aggregated customer information to an SQS queue. Each CSV file upload triggers a separate Lambda function invocation, which must aggregate data across these invocations.

**Solution:**

- AWS Lambda: Each CSV file upload triggers a Lambda function that processes the file and updates customer data.
- DynamoDB: Used as a persistent storage solution to store and aggregate customer data across multiple Lambda invocations. This allows data from different CSV files to be combined. [For more detail, at the end of the README.md](#necessity-of-using-dynamodb-instead-of-in-memory--rocessing-in-lambda)
- SQS: Once all data is aggregated, a final JSON message for each customer is sent to SQ

### Scability:
**Serverless Architecture:** The solution scales automatically with the volume of files due to the serverless nature of AWS Lambda, S3, and DynamoDB.

**DynamoDB:** Ensures efficient and scalable data storage, handling high throughput and large data volumes seamlessly.


### Permanent Storage Solution
**DynamoDB:** Chosen for its durability and scalability. It stores intermediate and final aggregated customer data across multiple Lambda invocations.


Refactored Flow:

1. The partner uploads CSV files to the S3 bucket in order.
  - First, uploads `customer_*.csv` file
  - Second, uploads `orders_*.csv` file
  - Third, uploads `items_*.csv` file
2. S3 triggers an event for each file upload, which invokes the Lambda function.
3. The Lambda function reads the file from S3, processes the data, and aggregates at DynamoDB.
5. When it's the `items_*.csv` file that is uploaded, then it makes a put request onto aggregated data(from `customer_*.csv` and `orders_*.csv`) at DynamoDB.
6. Then It produces JSON message for each customer with aggregated version
7. The Lambda function sends each JSON message to the SQS queue.
8. Downstream systems can consume messages from the SQS queue as needed.



## Deploying the stack
###  Prerequisites
Before you can run this project, you need to have the following tools installed:
- [Pulumi](https://www.pulumi.com/docs/get-started/install/)
- [Golang](https://golang.org/dl/)
- [AWS CLI](https://aws.amazon.com/cli/)


#### Cloning the repository
```
git clone git@github.com:ugurcancaykara/aws-lambda-fullstack.git
```

#### Authenticate to Providers
Before you start to provision resources, you need to authenticate to `aws` and `pulumi`

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

- then export this profile
```
export AWS_PROFILE="<YOUR_PROFILE_NAME"
```

that was all for aws side, now for the pulumi part,

```
pulumi login
```

and it will open a link, after you authenticated to pulumi, we are all good

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
cd deploy
pulumi up
```
and click yes to provision resources
```
yes <-
```

## Producing Messages

First, change your directory to root path folder level where `*.csv` files exists

### Assign values to variables

```
# Retrieve the S3 bucket name
S3_BUCKET_NAME=$(pulumi stack output s3BucketName)

# Retrieve the SQS queue URL
SQS_QUEUE_URL=$(pulumi stack output sqsQueueUrl)
```

### Triggering lamba invocation with s3 notifications
Order is important here:
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
You can check messages at SQS queue by running below command and are in expected structure
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

Delete all resources
```
cd deploy
pulumi destroy -y
```


## Improvements

### 1. **Lambda Function Scaling**
   - **Concurrent Executions**: AWS Lambda automatically scales based on the number of incoming events. Setting a reserved concurrency limit ensures that your function doesn’t overwhelm downstream services like DynamoDB or SQS, preventing throttling and maintaining system stability.
   - **Provisioned Concurrency**: Using provisioned concurrency can help eliminate cold starts, which reduces latency for workloads with predictable traffic patterns, ensuring a consistent user experience.

### 2. **SQS Dead Letter Queue (DLQ)**
   - **Dead Letter Queue (DLQ)**: Setting up a DLQ helps capture messages that can’t be processed successfully after several retries, allowing you to isolate and debug problematic messages without impacting the main processing flow.
   - **Redrive Policy**: Implementing a redrive policy automatically moves failed messages to the DLQ, preventing repeated processing failures from clogging the main queue, which enhances system reliability and debugging efficiency.

### 3. **DynamoDB Optimization**
   - **Provisioned Throughput**: Using provisioned throughput with auto-scaling ensures that your DynamoDB table can handle peak traffic efficiently without over-provisioning, optimizing cost while maintaining performance.
   - **On-Demand Mode**: On-demand mode is ideal for unpredictable traffic, allowing DynamoDB to scale instantly to handle any request load without manual intervention, ensuring seamless user experience.
   - **DAX (DynamoDB Accelerator)**: Implementing DAX reduces latency for read-heavy applications by caching results, which speeds up read operations and reduces load on the DynamoDB table.

### 4. **Monitoring and Tracing**
   - **AWS X-Ray Tracing**: Enabling X-Ray helps monitor and trace the execution flow, identifying performance bottlenecks and debugging issues effectively. It provides visibility into how requests are processed across the system.
   - **CloudWatch Logs**: Using CloudWatch Logs allows you to monitor, store, and access log files from Lambda, helping you quickly detect and troubleshoot issues. Setting up alarms based on logs can alert you to problems before they impact users.
   - **Custom Metrics**: Implementing custom metrics in CloudWatch gives you deeper insight into specific performance indicators, such as processing time and error rates, allowing for proactive performance tuning.

### 5. **Error Handling and Retries**
   - **Lambda Retries**: Lambda automatically retries failed executions twice. Ensuring your function is idempotent avoids duplicate processing and maintains data consistency, even in case of errors.
   - **Custom Error Handling**: Implementing custom error handling enables you to gracefully manage different error types, decide on retry logic, and prevent cascading failures, thereby improving system resilience.

### 6. **Security**
   - **Least Privilege IAM Roles**: Applying the principle of least privilege restricts access to only what is necessary, reducing the attack surface and minimizing potential damage from compromised credentials.
   - **Environment Variables Encryption**: Encrypting environment variables with AWS KMS ensures that sensitive information is securely stored and accessed, protecting against unauthorized access.
   - **VPC Integration**: Running Lambda within a VPC enables secure access to private resources (like DynamoDB endpoints) while minimizing exposure to the public internet, enhancing security.

### 7. **Cost Optimization**
   - **Optimize Lambda Execution Time**: Reducing Lambda execution time lowers costs since you pay based on compute time. This includes optimizing code, minimizing external calls, and efficiently managing resources.
   - **Use ARM-based Graviton2**: Graviton2 processors offer better price-performance for many workloads, helping you reduce costs without sacrificing performance.
   - **Right-Sizing Lambda Memory**: Choosing the appropriate memory size balances cost and performance, ensuring that your function runs efficiently without unnecessary expenses.

### 8. **Data Validation**
   - **Schema Validation**: Implementing schema validation ensures that incoming data meets expected formats before processing, reducing the likelihood of errors and improving data quality.
   - **Sanitize Inputs**: Sanitizing inputs prevents injection attacks and ensures that only valid data is processed, enhancing security and stability.

### 9. **Security Monitoring**
   - **AWS Config and Security Hub**: Using AWS Config and Security Hub to monitor for best practice compliance helps maintain a secure environment, detecting and addressing security risks proactively.
   - **Audit Logs**: Enabling CloudTrail for audit logs provides a detailed record of all API activity, which is essential for forensic analysis and compliance auditing.

By implementing these improvements, you can enhance overall scalability, security, reliability, and cost-effectiveness of your AWS Lambda-based data processing pipeline.



##### References
- [Deploying Go Lambda](https://docs.aws.amazon.com/lambda/latest/dg/golang-package.html)
- [best practices](https://docs.aws.amazon.com/lambda/latest/dg/best-practices.html)
- [Migrating go lambdas from 1.x to os-only](https://aws.amazon.com/blogs/compute/migrating-aws-lambda-functions-from-the-go1-x-runtime-to-the-custom-runtime-on-amazon-linux-2/)

