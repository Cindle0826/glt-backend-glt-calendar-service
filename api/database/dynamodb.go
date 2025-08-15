package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
	"glt-calendar-service/settings/env"
	"glt-calendar-service/settings/log"
	"go.uber.org/zap"
	"sync"
)

var (
	cfg          = env.GetConfig()
	dynamoClient *dynamodb.Client
	once         sync.Once
	logger       = log.GetLogger()
)

// InitDynamoDB Reference : https://pkg.go.dev/github.com/aws/aws-sdk-go-v2
func InitDynamoDB() error {
	svc := GetDynamoDBClient()

	// Check table exists
	exists, err := tableExists(svc, "Sessions")
	if err != nil {
		return fmt.Errorf("error checking table existence: %v", err)
	}

	if !exists {
		// createTable
		input := &dynamodb.CreateTableInput{
			TableName: aws.String("Sessions"),
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("session_id"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("session_id"),
					KeyType:       types.KeyTypeHash,
				},
			},
			ProvisionedThroughput: &types.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(5),
				WriteCapacityUnits: aws.Int64(5),
			},
		}

		_, err = svc.CreateTable(context.TODO(), input)
		if err != nil {
			return fmt.Errorf("error creating table: %v", err)
		}
		logger.Info("Created the table Sessions successfully!")
	}

	// enable TTL
	if err := enableTTL(); err != nil {
		return err
	}
	logger.Info("Enabled TTL for the table Sessions successfully!")

	return nil
}

func enableTTL() error {
	svc := GetDynamoDBClient()

	// 先檢查 TTL 是否已經啟用
	ttlResponse, err := svc.DescribeTimeToLive(context.TODO(), &dynamodb.DescribeTimeToLiveInput{
		TableName: aws.String("Sessions"),
	})

	if err != nil {
		return fmt.Errorf("error checking TTL status: %v", err)
	}

	// 檢查 TTL 狀態
	ttlEnabled := false
	if ttlResponse.TimeToLiveDescription != nil {
		status := ttlResponse.TimeToLiveDescription.TimeToLiveStatus
		attributeName := ttlResponse.TimeToLiveDescription.AttributeName

		// 檢查 TTL 是否已啟用，且使用的屬性名稱是 "ttl"
		if status == types.TimeToLiveStatusEnabled && *attributeName == "ttl" {
			ttlEnabled = true
			logger.Info("TTL is already enabled for Sessions table with attribute 'ttl'")
		}
	}

	// 如果 TTL 尚未啟用，則啟用它
	if !ttlEnabled {
		logger.Info("Enabling TTL for Sessions table...")
		_, err = svc.UpdateTimeToLive(context.TODO(), &dynamodb.UpdateTimeToLiveInput{
			TableName: aws.String("Sessions"),
			TimeToLiveSpecification: &types.TimeToLiveSpecification{
				AttributeName: aws.String("ttl"),
				Enabled:       aws.Bool(true),
			},
		})
		if err != nil {
			return fmt.Errorf("error enabling TTL: %v", err)
		}
		logger.Info("TTL enabled successfully for Sessions table")
	}
	return nil
}

// GetDynamoDBClient Get DynamodDB Client
func GetDynamoDBClient() *dynamodb.Client {
	once.Do(func() {
		var awsCfg aws.Config
		var err error

		if gin.Mode() == gin.ReleaseMode {
			awsCfg, err = config.LoadDefaultConfig(context.TODO(),
				config.WithRegion(cfg.DynamodbConfig.Region),
			)
		} else {
			customCreds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
				cfg.DynamodbConfig.DynamodbLocal.AccessKey,
				cfg.DynamodbConfig.DynamodbLocal.AccessKeyId,
				"",
			))

			awsCfg, err = config.LoadDefaultConfig(
				context.TODO(),
				config.WithRegion(cfg.DynamodbConfig.Region),
				config.WithCredentialsProvider(customCreds),
			)
		}

		if err != nil {
			logger.Error(fmt.Sprintf("unable to load SDK config, %v\n", err), zap.Error(err))
			return
		}

		if gin.Mode() == gin.ReleaseMode {
			dynamoClient = dynamodb.NewFromConfig(awsCfg)
		} else {
			dynamoClient = dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
				o.BaseEndpoint = aws.String(cfg.DynamodbConfig.DynamodbLocal.Endpoint)
			})
		}
	})
	return dynamoClient
}

// Check table exists
func tableExists(svc *dynamodb.Client, tableName string) (bool, error) {
	_, err := svc.DescribeTable(
		context.TODO(),
		&dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		},
	)

	if err != nil {
		// table doesn't exist, return false
		var notFoundEx *types.ResourceNotFoundException
		if ok := errors.As(err, &notFoundEx); ok {
			return false, nil
		}
		// other error, return error
		return false, err
	}

	return true, nil
}
