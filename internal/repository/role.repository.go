package repository

import (
	"github.com/allifiz/stockopname-api/internal/domain"
)

// RoleRepository represents the repository for managing users
type RoleRepository struct {
	// Example fields for the repository
	DB interface{}
}

// Example method
func (r *RoleRepository) GetByID(id int) (*domain.Role, error) {
	// Example implementation
	return &domain.Role{}, nil
}
