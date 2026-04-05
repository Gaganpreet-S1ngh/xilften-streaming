package config

import (
	"net/url"
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	Port          string
	DatabaseDSN   string
	RedisDSN      string
	GinMode       string
	Logger        *zap.Logger
	AccessSecret  string
	RefreshSecret string
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
		Port:          os.Getenv("PORT"),
		AccessSecret:  os.Getenv("JWT_ACCESS_SECRET"),
		RefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
		DatabaseDSN:   makeDatabaseDSN(),
		RedisDSN:      makeRedisDSN(),
		Logger:        logger,
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

func makeRedisDSN() string {
	scheme := "redis"
	if os.Getenv("REDIS_TLS") == "true" {
		scheme = "rediss"
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Path:   "/" + os.Getenv("REDIS_DB"),
	}

	if pass := os.Getenv("REDIS_PASSWORD"); pass != "" {
		u.User = url.UserPassword("", pass)
	}

	return u.String()
}
