package model

import "time"

// Toilet はアプリ内部で扱うトイレ情報の型。
// API層（自動生成）ともDB層とも独立させることで、変更に強くなる。
type Toilet struct {
	ToiletID           string    `dynamodbav:"toiletId"`
	Name               string    `dynamodbav:"name"`
	Brand              string    `dynamodbav:"brand"`
	Address            string    `dynamodbav:"address,omitempty"`
	Lat                float64   `dynamodbav:"lat"`
	Lng                float64   `dynamodbav:"lng"`
	ImageKey           string    `dynamodbav:"imageKey"`
	MaleCount          int       `dynamodbav:"maleCount"`
	FemaleCount        int       `dynamodbav:"femaleCount"`
	MultipurposeCount  int       `dynamodbav:"multipurposeCount"`
	RequiresPermission bool      `dynamodbav:"requiresPermission"`
	Note               string    `dynamodbav:"note,omitempty"`
	CreatedAt          time.Time `dynamodbav:"createdAt"`
}
