package repository

import (
	"github.com/allifiz/go-opname-api/internal/domain"
)

// UserRepository represents the repository for managing users
type UserRepository struct {
	// Example fields for the repository
	DB interface{}
}

// Example method
func (r *UserRepository) GetByID(id int) (*domain.User, error) {
	// Example implementation
	return &domain.User{}, nil
}
