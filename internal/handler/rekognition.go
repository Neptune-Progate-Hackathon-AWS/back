package handler

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
)

// isToiletImage は、S3に保存された画像をRekognitionで解析し、
// 「Toilet（トイレ）」というラベルが信頼度(Confidence) 70%以上で含まれているか判定します。
func (s *Server) isToiletImage(ctx context.Context, imageKey string) (bool, error) {
	// 1. Rekognitionに渡す「どの画像を判定するか」のお願いを作成
	input := &rekognition.DetectLabelsInput{
		Image: &types.Image{
			S3Object: &types.S3Object{
				Bucket: aws.String(s.bucketName), // 対象のS3バケット
				Name:   aws.String(imageKey),     // 対象の画像のキー
			},
		},
		MaxLabels:     aws.Int32(10),     // 上位10個のラベルをチェック
		MinConfidence: aws.Float32(70.0), // 信頼度70%以上を指定
	}

	// 2. AIに画像を送信して解析
	output, err := s.rekognitionClient.DetectLabels(ctx, input)
	if err != nil {
		return false, fmt.Errorf("failed to detect labels: %w", err)
	}

	// 3. ラベルの中に「Toilet」があるか探す
	for _, label := range output.Labels {
		if label.Name != nil && *label.Name == "Toilet" {
			return true, nil
		}
	}

	return false, nil
}
