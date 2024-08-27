package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/s3"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/sqs"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// Create an S3 Bucket
		bucket, err := s3.NewBucket(ctx, "dataBucket", nil)
		if err != nil {
			return err
		}

		// Create SQS queue
		queue, err := sqs.NewQueue(ctx, "dataQueue", nil)
		if err != nil {
			return err
		}

		// IAM role for Lambda
		lambdaExecRole, err := iam.NewRole(ctx, "lambdaExecutionRole", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Principal": {
						"Service": "lambda.amazonaws.com"
					},
					"Action": "sts:AssumeRole"
				}]
			}`),
		})
		if err != nil {
			return err
		}

		// Attach basic execution role policy to the IAM role
		_, err = iam.NewRolePolicyAttachment(ctx, "lambdaExecPolicyAttachment", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaExecRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
		})
		if err != nil {
			return err
		}

		// Attach SQS role to lambda
		_, err = iam.NewRolePolicyAttachment(ctx, "lambdaSQSPolicyAttachment", &iam.RolePolicyAttachmentArgs{
			Role:      lambdaExecRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonSQSFullAccess"),
		})
		if err != nil {
			return err
		}

		// Lambda function
		// Set arguments for constructing the function resource.
		args := &lambda.FunctionArgs{
			Handler: pulumi.String("main"),
			Role:    lambdaExecRole.Arn,

			Runtime: pulumi.String("provided.al2023"),
			// Runtime: pulumi.String("go1.x"),
			Code: pulumi.NewFileArchive("../processinglambda/deployment.zip"),
			Environment: lambda.FunctionEnvironmentArgs{
				Variables: pulumi.StringMap{
					"S3_BUCKET": bucket.Bucket,
					"SQS_QUEUE": queue.Url,
				},
			},
		}

		// FIX: can't provision lambda, it throws error related with unsupported version of runtime 1.x

		// Create the lambda using the args.
		lambdaFunc, err := lambda.NewFunction(
			ctx,
			"dataProcessor",
			args,
		)
		if err != nil {
			return err
		}

		_, err = s3.NewBucketNotification(ctx, "bucketNotification", &s3.BucketNotificationArgs{
			Bucket: bucket.ID(),
			LambdaFunctions: s3.BucketNotificationLambdaFunctionArray{
				&s3.BucketNotificationLambdaFunctionArgs{
					LambdaFunctionArn: lambdaFunc.Arn,
					Events: pulumi.StringArray{
						pulumi.String("s3:ObjectCreated:*"),
					},
					FilterPrefix: pulumi.String(""),
					FilterSuffix: pulumi.String(".csv"),
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{lambdaFunc}))
		if err != nil {
			return err
		}

		return nil
	})
}
