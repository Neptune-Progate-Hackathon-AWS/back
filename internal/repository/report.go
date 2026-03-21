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

const reportsTableName = "ReportsTable"

// ReportRepository は DynamoDB を使った虚偽報告の永続化を担当する。
type ReportRepository struct {
	client *dynamodb.Client
}

func NewReportRepository(client *dynamodb.Client) *ReportRepository {
	return &ReportRepository{client: client}
}

// Save は報告を DynamoDB に保存する。
// PK=toiletId, SK=userId で同一ユーザーの重複を防止する。
func (r *ReportRepository) Save(ctx context.Context, report model.Report) error {
	av, err := attributevalue.MarshalMap(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(reportsTableName),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(toiletId) AND attribute_not_exists(userId)"),
	})
	if err != nil {
		return fmt.Errorf("failed to put report to DynamoDB: %w", err)
	}

	return nil
}

// CountByToiletID は指定トイレIDの報告件数を取得する。
func (r *ReportRepository) CountByToiletID(ctx context.Context, toiletID string) (int, error) {
	output, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(reportsTableName),
		KeyConditionExpression: aws.String("toiletId = :tid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":tid": &types.AttributeValueMemberS{Value: toiletID},
		},
		Select:         types.SelectCount,
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count reports: %w", err)
	}

	return int(output.Count), nil
}

// ExistsByToiletAndUser は指定トイレ・ユーザーの組み合わせで報告が存在するか確認する。
func (r *ReportRepository) ExistsByToiletAndUser(ctx context.Context, toiletID, userID string) (bool, error) {
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(reportsTableName),
		Key: map[string]types.AttributeValue{
			"toiletId": &types.AttributeValueMemberS{Value: toiletID},
			"userId":   &types.AttributeValueMemberS{Value: userID},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return false, fmt.Errorf("failed to check report existence: %w", err)
	}

	return output.Item != nil, nil
}
