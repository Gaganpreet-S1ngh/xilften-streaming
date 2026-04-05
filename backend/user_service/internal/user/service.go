package user

import (
	"go.uber.org/zap"
)

type Service interface {
}

type service struct {
	repository Repository
	logger     *zap.Logger
}

func NewService(repository Repository, logger *zap.Logger) Service {
	return &service{
		repository: repository,
	}
}
