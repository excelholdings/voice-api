package models

type APIKey struct {
	BaseModel
	UserId uint   `json:"user_id" gorm:"index"`
	Name   string `json:"name"`
	Key    string `json:"key" gorm:"uniqueIndex"`
}