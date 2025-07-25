package models

type User struct {
	BaseModel
	Email          string `json:"email" gorm:"uniqueIndex"`
	HashedPassword string `json:"-"`
	Password       string `json:"-" gorm:"-"`

	StripeCustomerID string `json"-"`

	Plan string `json:"plan"`
	Details PlanDetails `json:"details" gorm:"serializer:json"`

}

type PlanDetails struct {
	Minutes uint `json:"minutes"`
	PricePerMinute float64 `json:"price_per_minute"`
}