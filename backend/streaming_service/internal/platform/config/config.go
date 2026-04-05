package config

import (
	"net/url"
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	Port        string
	GinMode     string
	DatabaseDSN string
	Logger      *zap.Logger
}

func LoadConfig() *Config {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(logger)
	}

	if os.Getenv("ENV") != "prod" {
		if err := godotenv.Load(); err != nil {
			logger.Warn("failed to load .env", zap.Error(err))
		}
	}

	return &Config{
		Port:        os.Getenv("PORT"),
		Logger:      logger,
		DatabaseDSN: makeDatabaseDSN(),
	}
}

//==========================================//
//             PRIVATE FUNCTIONS            //
//==========================================//

func makeDatabaseDSN() string {
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD")),
		Host:     os.Getenv("POSTGRES_HOST") + ":" + os.Getenv("POSTGRES_PORT"),
		Path:     os.Getenv("POSTGRES_DB"),
		RawQuery: "sslmode=disable",
	}

	return u.String()

}
