package model

import "time"

type Vote struct {
	ToiletID           string    `dynamodbav:"toiletId"`
	UserID             string    `dynamodbav:"userId"`
	ToiletType         string    `dynamodbav:"toiletType"`
	RequiresPermission bool      `dynamodbav:"requiresPermission"`
	Note               string    `dynamodbav:"note,omitempty"`
	ImageKey           string    `dynamodbav:"imageKey,omitempty"`
	CreatedAt          time.Time `dynamodbav:"createdAt"`
	UpdatedAt          time.Time `dynamodbav:"updatedAt,omitempty"`
}
