package dao

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"glt-calendar-service/api/database"
	"glt-calendar-service/api/model"
	"glt-calendar-service/settings/log"
	"go.uber.org/zap"
)

var logger = log.GetLogger()

// SessionDaoInterface defines the interface for session data access
type SessionDaoInterface interface {
	GetSessionsBySessionID(sessionID string) (*model.Session, error)
	InsertSession(session model.Session) error
	UpdateSession(session model.Session) error
	DeleteSession(sessionID string) error
}

type SessionDao struct {
	dynamoClient *dynamodb.Client
}

func NewSessionDao() *SessionDao {
	return &SessionDao{
		dynamoClient: database.GetDynamoDBClient(),
	}
}

func (s *SessionDao) GetSessionsBySessionID(sessionID string) (*model.Session, error) {
	client := s.dynamoClient

	getInput := &dynamodb.GetItemInput{
		TableName: aws.String("Sessions"),
		Key: map[string]types.AttributeValue{
			"session_id": &types.AttributeValueMemberS{Value: sessionID},
		},
		ConsistentRead: aws.Bool(true),
	}

	result, err := client.GetItem(context.TODO(), getInput)
	if err != nil {
		return nil, fmt.Errorf("get item error: %w", err)
	}

	if result.Item == nil || len(result.Item) == 0 {
		return nil, fmt.Errorf("no session found")
	}

	var session model.Session
	err = attributevalue.UnmarshalMap(result.Item, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (s *SessionDao) InsertSession(session model.Session) error {

	// session data convert to DynamoDB Attribute Value
	av, err := attributevalue.MarshalMap(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session : %w", err)
	}

	// Create PutItem Request
	input := &dynamodb.PutItemInput{
		TableName: aws.String("Sessions"),
		Item:      av,
	}

	// save to DynamoDB
	_, err = s.dynamoClient.PutItem(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to save session to DynamoDB : %w", err)
	}
	return nil
}

func (s *SessionDao) UpdateSession(updatedSession model.Session) error {
	// update session date transfer DynamoDB Attribute
	av, err := attributevalue.MarshalMap(updatedSession)
	if err != nil {
		return fmt.Errorf("failed to marshal updated session : %w", err)
	}

	// create PutItem request
	input := &dynamodb.PutItemInput{
		TableName: aws.String("Sessions"),
		Item:      av,
	}

	// save DynamoDB
	_, err = s.dynamoClient.PutItem(context.TODO(), input)
	if err != nil {
		logger.Error("Failed to update session in DynamoDB", zap.Error(err))
		return err
	}
	return nil
}

func (s *SessionDao) DeleteSession(sessionID string) error {

	// create DeleteItem request
	deleteInput := &dynamodb.DeleteItemInput{
		TableName: aws.String("Sessions"),
		Key: map[string]types.AttributeValue{
			"session_id": &types.AttributeValueMemberS{Value: sessionID},
		},
	}

	// From DynamoDB delete data
	_, err := s.dynamoClient.DeleteItem(context.TODO(), deleteInput)
	if err != nil {
		return fmt.Errorf("failed to delete session from DynamoDB : %w", err)
	}
	return nil
}
