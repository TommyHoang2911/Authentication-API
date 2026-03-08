package service

import "auth-service/internal/model"

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (s *AuthService) Register(email string, password string) (*model.User, error) {

	user := &model.User{
		Email: email,
	}

	return user, nil
}
