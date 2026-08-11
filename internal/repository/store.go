package repository

import (
	"context"

	db "github.com/allifiz/go-opname-api/internal/database/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"
)

type Store struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: db.New(pool)}
}

func (s *Store) ListUnits(ctx context.Context) ([]db.ListUnitsRow, error) {
	return s.queries.ListUnits(ctx)
}

func (s *Store) ListMaterials(ctx context.Context) ([]db.ListMaterialsRow, error) {
	return s.queries.ListMaterials(ctx)
}

func (s *Store) GetMaterial(ctx context.Context, id pgtype.UUID) (db.GetMaterialByIDRow, error) {
	return s.queries.GetMaterialByID(ctx, id)
}

func (s *Store) CreateMaterial(ctx context.Context, arg db.CreateMaterialParams) (db.Material, error) {
	return s.queries.CreateMaterial(ctx, arg)
}

func (s *Store) UpdateMaterial(ctx context.Context, arg db.UpdateMaterialParams) (db.Material, error) {
	return s.queries.UpdateMaterial(ctx, arg)
}

func (s *Store) ListPeriods(ctx context.Context) ([]db.Period, error) {
	return s.queries.ListPeriods(ctx)
}

func (s *Store) GetPeriod(ctx context.Context, id pgtype.UUID) (db.Period, error) {
	return s.queries.GetPeriodByID(ctx, id)
}

func (s *Store) CreatePeriod(ctx context.Context, arg db.CreatePeriodParams) (db.Period, error) {
	return s.queries.CreatePeriod(ctx, arg)
}

func (s *Store) ListMenuTemplates(ctx context.Context) ([]db.MenuTemplate, error) {
	return s.queries.ListMenuTemplates(ctx)
}

func (s *Store) GetMenuTemplate(ctx context.Context, id pgtype.UUID) (db.MenuTemplate, error) {
	return s.queries.GetMenuTemplateByID(ctx, id)
}

func (s *Store) ListMenuTemplateComponents(ctx context.Context, id pgtype.UUID) ([]db.MenuTemplateComponent, error) {
	return s.queries.ListMenuTemplateComponents(ctx, id)
}

func (s *Store) ListMenuTemplateComponentMaterials(ctx context.Context, id pgtype.UUID) ([]db.ListMenuTemplateComponentMaterialsRow, error) {
	return s.queries.ListMenuTemplateComponentMaterials(ctx, id)
}

func (s *Store) GetScheduledMenu(ctx context.Context, id pgtype.UUID) (db.ScheduledMenu, error) {
	return s.queries.GetScheduledMenuByID(ctx, id)
}

func (s *Store) ListScheduledMenuComponents(ctx context.Context, id pgtype.UUID) ([]db.ScheduledMenuComponent, error) {
	return s.queries.ListScheduledMenuComponents(ctx, id)
}

func (s *Store) ListScheduledMenuComponentMaterials(ctx context.Context, id pgtype.UUID) ([]db.ListScheduledMenuComponentMaterialsRow, error) {
	return s.queries.ListScheduledMenuComponentMaterials(ctx, id)
}

func (s *Store) WithTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(s.queries.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
