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

const subscriptionsTableName = "SubscriptionsTable"

// SubscriptionRepository は DynamoDB を使ったプッシュ通知サブスクリプションの永続化を担当する。
type SubscriptionRepository struct {
	client *dynamodb.Client
}

func NewSubscriptionRepository(client *dynamodb.Client) *SubscriptionRepository {
	return &SubscriptionRepository{client: client}
}

// Save はサブスクリプションを DynamoDB に保存する。
func (r *SubscriptionRepository) Save(ctx context.Context, sub model.Subscription) error {
	av, err := attributevalue.MarshalMap(sub)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(subscriptionsTableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to put subscription to DynamoDB: %w", err)
	}

	return nil
}

// FindByID は指定IDのサブスクリプションを取得する。
func (r *SubscriptionRepository) FindByID(ctx context.Context, id string) (*model.Subscription, error) {
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(subscriptionsTableName),
		Key: map[string]types.AttributeValue{
			"subscriptionId": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription from DynamoDB: %w", err)
	}

	if output.Item == nil {
		return nil, nil
	}

	var sub model.Subscription
	if err := attributevalue.UnmarshalMap(output.Item, &sub); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscription: %w", err)
	}

	return &sub, nil
}

// FindByUserID は指定ユーザーIDの全サブスクリプションを取得する。
// GSI "userId-index" を使用する。
func (r *SubscriptionRepository) FindByUserID(ctx context.Context, userID string) ([]model.Subscription, error) {
	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(subscriptionsTableName),
		IndexName:              aws.String("userId-index"),
		KeyConditionExpression: aws.String("userId = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query subscriptions: %w", err)
	}

	var subs []model.Subscription
	if err := attributevalue.UnmarshalListOfMaps(output.Items, &subs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscriptions: %w", err)
	}

	return subs, nil
}

// Delete は指定IDのサブスクリプションを削除する。
func (r *SubscriptionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(subscriptionsTableName),
		Key: map[string]types.AttributeValue{
			"subscriptionId": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete subscription from DynamoDB: %w", err)
	}
	return nil
}
