package repository

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/model"
)

const tableName = "ToiletsTable"

// ToiletRepository は DynamoDB を使ったトイレ情報の永続化を担当する。
type ToiletRepository struct {
	client *dynamodb.Client
}

func NewToiletRepository(client *dynamodb.Client) *ToiletRepository {
	return &ToiletRepository{client: client}
}

// Save はトイレ情報を DynamoDB に保存する。
func (r *ToiletRepository) Save(ctx context.Context, t model.Toilet) error {
	av, err := attributevalue.MarshalMap(t)
	if err != nil {
		return fmt.Errorf("failed to marshal toilet: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to put toilet to DynamoDB: %w", err)
	}

	return nil
}
