package logger

import (
	"go.uber.org/zap"
)

var S *zap.SugaredLogger

func InitLogger(env string) {
	var logger *zap.Logger
	var err error

	// Adjust the logging level based on the environment
	if env == "production" {
		// Production logger config: JSON format and error level
		logger, err = zap.NewProduction()
	} else {
		// Development logger config: console format and debug level
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		panic("cannot initialize zap logger: " + err.Error())
	}

	S = logger.Sugar()
}
