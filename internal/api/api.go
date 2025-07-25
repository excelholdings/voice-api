package api

import (
	"github.com/flyflow-devs/flyflow/internal/classifier"
	"github.com/flyflow-devs/flyflow/internal/config"
	"gorm.io/gorm"
)

type API struct {
	Cfg *config.Config
	DB *gorm.DB
	Classifier *classifier.Classifier
}

func NewAPI(cfg *config.Config, db *gorm.DB) *API {
	return &API{
		Cfg: cfg,
		DB: db,
		Classifier: classifier.NewClassifier(),
	}
}