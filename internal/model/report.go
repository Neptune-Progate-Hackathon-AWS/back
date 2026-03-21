package model

import "time"

// Report はトイレの虚偽報告。
type Report struct {
	ToiletID  string    `dynamodbav:"toiletId"`
	UserID    string    `dynamodbav:"userId"`
	ReportID  string    `dynamodbav:"reportId"`
	Reason    string    `dynamodbav:"reason"`
	Comment   string    `dynamodbav:"comment,omitempty"`
	CreatedAt time.Time `dynamodbav:"createdAt"`
}
