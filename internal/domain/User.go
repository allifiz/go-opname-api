package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleChef        UserRole = "CHEF"
	RoleProcurement UserRole = "PROCUREMENT"
	RoleWarehouse   UserRole = "WAREHOUSE"
	RoleAccountant  UserRole = "ACCOUNTANT"
	RoleOwner       UserRole = "OWNER"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         UserRole  `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
