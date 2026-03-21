package repository

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/Neptune-Progate-Hackathon-AWS/back/internal/model"
)

const votesTableName = "VotesTable"

type VoteRepository struct {
	client *dynamodb.Client
}

func NewVoteRepository(client *dynamodb.Client) *VoteRepository {
	return &VoteRepository{client: client}
}

// Save は投票を保存する（upsert）。
func (r *VoteRepository) Save(ctx context.Context, v model.Vote) error {
	av, err := attributevalue.MarshalMap(v)
	if err != nil {
		return fmt.Errorf("failed to marshal vote: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(votesTableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to put vote to DynamoDB: %w", err)
	}

	return nil
}

// FindByToiletID は指定トイレIDの投票一覧を取得する。
func (r *VoteRepository) FindByToiletID(ctx context.Context, toiletID string) ([]model.Vote, error) {
	keyCond := expression.Key("toiletId").Equal(expression.Value(toiletID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(votesTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query votes: %w", err)
	}

	var votes []model.Vote
	if err := attributevalue.UnmarshalListOfMaps(output.Items, &votes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal votes: %w", err)
	}

	return votes, nil
}

// CountByToiletID は指定トイレIDの投票数を返す。
func (r *VoteRepository) CountByToiletID(ctx context.Context, toiletID string) (int, error) {
	keyCond := expression.Key("toiletId").Equal(expression.Value(toiletID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return 0, fmt.Errorf("failed to build expression: %w", err)
	}

	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(votesTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		Select:                    types.SelectCount,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count votes: %w", err)
	}

	return int(output.Count), nil
}

// DeleteByToiletID は指定トイレIDの全投票を削除する。
func (r *VoteRepository) DeleteByToiletID(ctx context.Context, toiletID string) error {
	votes, err := r.FindByToiletID(ctx, toiletID)
	if err != nil {
		return err
	}

	for _, v := range votes {
		_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(votesTableName),
			Key: map[string]types.AttributeValue{
				"toiletId": &types.AttributeValueMemberS{Value: v.ToiletID},
				"userId":   &types.AttributeValueMemberS{Value: v.UserID},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to delete vote: %w", err)
		}
	}

	return nil
}
