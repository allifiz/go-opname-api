package service

import (
	"github.com/allifiz/go-opname-api/internal/domain"
	"github.com/allifiz/go-opname-api/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
}

// NewUserService creates a new instance of UserService.
func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// GetUserByID retrieves a user by their ID.
func (s *UserService) GetUserByID(id int) (*domain.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return user, nil
}
