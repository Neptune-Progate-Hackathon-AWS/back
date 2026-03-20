package repository

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

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

// FindAll は全トイレ情報を DynamoDB から取得する。
func (r *ToiletRepository) FindAll(ctx context.Context) ([]model.Toilet, error) {
	output, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan toilets: %w", err)
	}

	var toilets []model.Toilet
	if err := attributevalue.UnmarshalListOfMaps(output.Items, &toilets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal toilets: %w", err)
	}

	return toilets, nil
}

// FindByID は指定IDのトイレ情報を DynamoDB から取得する。
// 見つからない場合は nil を返す。
func (r *ToiletRepository) FindByID(ctx context.Context, id string) (*model.Toilet, error) {
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"toiletId": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get toilet from DynamoDB: %w", err)
	}

	if output.Item == nil {
		return nil, nil
	}

	var t model.Toilet
	if err := attributevalue.UnmarshalMap(output.Item, &t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal toilet: %w", err)
	}

	return &t, nil
}

// Delete は指定IDのトイレ情報を DynamoDB から削除する。
func (r *ToiletRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"toiletId": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete toilet from DynamoDB: %w", err)
	}
	return nil
}
