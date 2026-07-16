package domain

import (
	"time"

	"github.com/google/uuid"
)

type IngredientUnit string

const (
	UnitKilogram IngredientUnit = "KG"
	UnitLiter    IngredientUnit = "LITER"
)

type Ingredient struct {
	ID        uuid.UUID      `json:"id"`
	Name      string         `json:"name"`
	Unit      IngredientUnit `json:"unit"`
	IsActive  bool           `json:"is_active"`
	CreatedBy uuid.UUID      `json:"created_by"`
	UpdatedBy *uuid.UUID     `json:"updated_by,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}
