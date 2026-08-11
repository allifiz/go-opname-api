package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/allifiz/go-opname-api/internal/database/generated"
	"github.com/allifiz/go-opname-api/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrInvalidInput = errors.New("invalid input")

type CoreService struct {
	store *repository.Store
}

func NewCoreService(store *repository.Store) *CoreService {
	return &CoreService{store: store}
}

type CreateMaterialInput struct {
	Name   string `json:"name"`
	UnitID string `json:"unit_id"`
}

type UpdateMaterialInput struct {
	Name     string `json:"name"`
	UnitID   string `json:"unit_id"`
	IsActive bool   `json:"is_active"`
}

type CreatePeriodInput struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
}

type MenuMaterialInput struct {
	MaterialID   string `json:"material_id"`
	QtyPerPortion string `json:"qty_per_portion"`
	UnitID       string `json:"unit_id"`
}

type MenuComponentInput struct {
	Name      string              `json:"name"`
	SortOrder int32               `json:"sort_order"`
	Materials []MenuMaterialInput `json:"materials"`
}

type CreateMenuTemplateInput struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Components  []MenuComponentInput `json:"components"`
}

type CreateScheduledMenuInput struct {
	PeriodID        string `json:"period_id"`
	MenuTemplateID  string `json:"menu_template_id"`
	MenuDate        string `json:"menu_date"`
	PlannedPortions int32  `json:"planned_portions"`
}

func parseUUID(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if strings.TrimSpace(value) == "" {
		return id, fmt.Errorf("%w: uuid is required", ErrInvalidInput)
	}
	if err := id.Scan(value); err != nil {
		return pgtype.UUID{}, fmt.Errorf("%w: invalid uuid", ErrInvalidInput)
	}
	return id, nil
}

func parseDate(value string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("%w: date must use YYYY-MM-DD", ErrInvalidInput)
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func parseNumeric(value string) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(value); err != nil || !n.Valid {
		return pgtype.Numeric{}, fmt.Errorf("%w: invalid numeric value", ErrInvalidInput)
	}
	return n, nil
}

func (s *CoreService) ListUnits(ctx context.Context) ([]db.ListUnitsRow, error) {
	return s.store.ListUnits(ctx)
}

func (s *CoreService) ListMaterials(ctx context.Context) ([]db.ListMaterialsRow, error) {
	return s.store.ListMaterials(ctx)
}

func (s *CoreService) GetMaterial(ctx context.Context, id string) (db.GetMaterialByIDRow, error) {
	materialID, err := parseUUID(id)
	if err != nil {
		return db.GetMaterialByIDRow{}, err
	}
	return s.store.GetMaterial(ctx, materialID)
}

func (s *CoreService) CreateMaterial(ctx context.Context, input CreateMaterialInput) (db.Material, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return db.Material{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	unitID, err := parseUUID(input.UnitID)
	if err != nil {
		return db.Material{}, err
	}
	return s.store.CreateMaterial(ctx, db.CreateMaterialParams{Name: input.Name, UnitID: unitID})
}

func (s *CoreService) UpdateMaterial(ctx context.Context, id string, input UpdateMaterialInput) (db.Material, error) {
	materialID, err := parseUUID(id)
	if err != nil {
		return db.Material{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return db.Material{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	unitID, err := parseUUID(input.UnitID)
	if err != nil {
		return db.Material{}, err
	}
	return s.store.UpdateMaterial(ctx, db.UpdateMaterialParams{ID: materialID, Name: input.Name, UnitID: unitID, IsActive: input.IsActive})
}

func (s *CoreService) ListPeriods(ctx context.Context) ([]db.Period, error) {
	return s.store.ListPeriods(ctx)
}

func (s *CoreService) CreatePeriod(ctx context.Context, input CreatePeriodInput) (db.Period, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return db.Period{}, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	startDate, err := parseDate(input.StartDate)
	if err != nil {
		return db.Period{}, err
	}
	end := startDate.Time.AddDate(0, 0, 13)
	return s.store.CreatePeriod(ctx, db.CreatePeriodParams{
		Name: input.Name,
		StartDate: startDate,
		EndDate: pgtype.Date{Time: end, Valid: true},
	})
}

func (s *CoreService) ListMenuTemplates(ctx context.Context) ([]db.MenuTemplate, error) {
	return s.store.ListMenuTemplates(ctx)
}

func (s *CoreService) GetMenuTemplate(ctx context.Context, id string) (map[string]any, error) {
	templateID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}
	tpl, err := s.store.GetMenuTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	components, err := s.store.ListMenuTemplateComponents(ctx, templateID)
	if err != nil {
		return nil, err
	}
	componentData := make([]map[string]any, 0, len(components))
	for _, component := range components {
		materials, err := s.store.ListMenuTemplateComponentMaterials(ctx, component.ID)
		if err != nil {
			return nil, err
		}
		componentData = append(componentData, map[string]any{"component": component, "materials": materials})
	}
	return map[string]any{"template": tpl, "components": componentData}, nil
}

func (s *CoreService) CreateMenuTemplate(ctx context.Context, input CreateMenuTemplateInput) (db.MenuTemplate, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Components) == 0 {
		return db.MenuTemplate{}, fmt.Errorf("%w: name and components are required", ErrInvalidInput)
	}
	var created db.MenuTemplate
	err := s.store.WithTx(ctx, func(q *db.Queries) error {
		var description pgtype.Text
		if strings.TrimSpace(input.Description) != "" {
			description = pgtype.Text{String: strings.TrimSpace(input.Description), Valid: true}
		}
		var err error
		created, err = q.CreateMenuTemplate(ctx, db.CreateMenuTemplateParams{Name: input.Name, Description: description})
		if err != nil {
			return err
		}
		for _, componentInput := range input.Components {
			if strings.TrimSpace(componentInput.Name) == "" || len(componentInput.Materials) == 0 {
				return fmt.Errorf("%w: each component needs name and materials", ErrInvalidInput)
			}
			component, err := q.CreateMenuTemplateComponent(ctx, db.CreateMenuTemplateComponentParams{
				MenuTemplateID: created.ID,
				Name: strings.TrimSpace(componentInput.Name),
				SortOrder: componentInput.SortOrder,
			})
			if err != nil {
				return err
			}
			for _, materialInput := range componentInput.Materials {
				materialID, err := parseUUID(materialInput.MaterialID)
				if err != nil { return err }
				unitID, err := parseUUID(materialInput.UnitID)
				if err != nil { return err }
				qty, err := parseNumeric(materialInput.QtyPerPortion)
				if err != nil { return err }
				if _, err := q.CreateMenuTemplateComponentMaterial(ctx, db.CreateMenuTemplateComponentMaterialParams{
					MenuTemplateComponentID: component.ID,
					MaterialID: materialID,
					QtyPerPortion: qty,
					UnitID: unitID,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return created, err
}

func (s *CoreService) CreateScheduledMenu(ctx context.Context, input CreateScheduledMenuInput) (db.ScheduledMenu, error) {
	periodID, err := parseUUID(input.PeriodID)
	if err != nil { return db.ScheduledMenu{}, err }
	templateID, err := parseUUID(input.MenuTemplateID)
	if err != nil { return db.ScheduledMenu{}, err }
	menuDate, err := parseDate(input.MenuDate)
	if err != nil { return db.ScheduledMenu{}, err }
	if input.PlannedPortions <= 0 {
		return db.ScheduledMenu{}, fmt.Errorf("%w: planned_portions must be greater than zero", ErrInvalidInput)
	}
	period, err := s.store.GetPeriod(ctx, periodID)
	if err != nil { return db.ScheduledMenu{}, err }
	if menuDate.Time.Before(period.StartDate.Time) || menuDate.Time.After(period.EndDate.Time) {
		return db.ScheduledMenu{}, fmt.Errorf("%w: menu_date must be inside period", ErrInvalidInput)
	}

	var scheduled db.ScheduledMenu
	err = s.store.WithTx(ctx, func(q *db.Queries) error {
		var err error
		scheduled, err = q.CreateScheduledMenu(ctx, db.CreateScheduledMenuParams{
			PeriodID: periodID,
			MenuTemplateID: templateID,
			MenuDate: menuDate,
			PlannedPortions: input.PlannedPortions,
		})
		if err != nil { return err }

		components, err := q.ListMenuTemplateComponents(ctx, templateID)
		if err != nil { return err }
		if len(components) == 0 {
			return fmt.Errorf("%w: template has no components", ErrInvalidInput)
		}
		for _, sourceComponent := range components {
			component, err := q.CreateScheduledMenuComponent(ctx, db.CreateScheduledMenuComponentParams{
				ScheduledMenuID: scheduled.ID,
				SourceTemplateComponentID: sourceComponent.ID,
				Name: sourceComponent.Name,
				SortOrder: sourceComponent.SortOrder,
			})
			if err != nil { return err }
			materials, err := q.ListMenuTemplateComponentMaterials(ctx, sourceComponent.ID)
			if err != nil { return err }
			for _, sourceMaterial := range materials {
				if _, err := q.CreateScheduledMenuComponentMaterial(ctx, db.CreateScheduledMenuComponentMaterialParams{
					ScheduledMenuComponentID: component.ID,
					SourceTemplateMaterialID: sourceMaterial.ID,
					MaterialID: sourceMaterial.MaterialID,
					QtyPerPortion: sourceMaterial.QtyPerPortion,
					UnitID: sourceMaterial.UnitID,
				}); err != nil { return err }
			}
		}
		return nil
	})
	return scheduled, err
}

func (s *CoreService) GetScheduledMenu(ctx context.Context, id string) (map[string]any, error) {
	scheduledID, err := parseUUID(id)
	if err != nil { return nil, err }
	menu, err := s.store.GetScheduledMenu(ctx, scheduledID)
	if err != nil { return nil, err }
	components, err := s.store.ListScheduledMenuComponents(ctx, scheduledID)
	if err != nil { return nil, err }
	componentData := make([]map[string]any, 0, len(components))
	for _, component := range components {
		materials, err := s.store.ListScheduledMenuComponentMaterials(ctx, component.ID)
		if err != nil { return nil, err }
		componentData = append(componentData, map[string]any{"component": component, "materials": materials})
	}
	return map[string]any{"scheduled_menu": menu, "components": componentData}, nil
}
