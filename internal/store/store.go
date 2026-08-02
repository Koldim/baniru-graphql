package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"baniru-test/internal/models"
)

type Store struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

// GetMain возвращает запись main вместе с сателлитом, если она не удалена.
func (s *Store) GetMain(ctx context.Context, id int) (*models.Main, error) {
	const q = `
		SELECT id, title, sub_id, sub_obj::text, created_at, update_at, deleted_at
		FROM main
		WHERE id = $1 AND deleted_at IS NULL
	`
	var m models.Main
	var subID *int
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&m.ID, &m.Title, &subID, &m.SubObj, &m.CreatedAt, &m.UpdateAt, &m.DeletedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if subID != nil {
		m.SubID = *subID
	}
	if err := s.fillSatellite(ctx, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Store) fillSatellite(ctx context.Context, m *models.Main) error {
	switch m.SubObj {
	case models.SubObjTools:
		const q = `
			SELECT description1 FROM tools
			WHERE id = $1 AND main_id = $2 AND deleted_at IS NULL
		`
		var d string
		err := s.pool.QueryRow(ctx, q, m.SubID, m.ID).Scan(&d)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		m.Description1 = &d
	case models.SubObjTables:
		const q = `
			SELECT description2 FROM "tables"
			WHERE id = $1 AND main_id = $2 AND deleted_at IS NULL
		`
		var d string
		err := s.pool.QueryRow(ctx, q, m.SubID, m.ID).Scan(&d)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		m.Description2 = &d
	case models.SubObjChairs:
		const q = `
			SELECT description3, type::text FROM chairs
			WHERE id = $1 AND main_id = $2 AND deleted_at IS NULL
		`
		var d string
		var t models.ChairType
		err := s.pool.QueryRow(ctx, q, m.SubID, m.ID).Scan(&d, &t)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		m.Description3 = &d
		m.Type = &t
	default:
		return fmt.Errorf("неизвесный sub_obj: %s", m.SubObj)
	}
	return nil
}

// CreateMain создаёт main и связанный сателлит в одной транзакции.
func (s *Store) CreateMain(ctx context.Context, in models.MainInput) (*models.Main, error) {
	if err := validateInput(in); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var mainID int
	err = tx.QueryRow(ctx, `
		INSERT INTO main (title, sub_obj)
		VALUES ($1, $2)
		RETURNING id
	`, in.Title, string(in.SubObj)).Scan(&mainID)
	if err != nil {
		return nil, err
	}

	subID, err := insertSatellite(ctx, tx, mainID, in)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE main SET sub_id = $1, update_at = NOW() WHERE id = $2
	`, subID, mainID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetMain(ctx, mainID)
}

// UpdateMain обновляет main и его сателлит. Смена sub_obj не поддерживаеться.
func (s *Store) UpdateMain(ctx context.Context, id int, in models.MainInput) (*models.Main, error) {
	if err := validateInput(in); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var subID int
	var subObj models.SubObj
	err = tx.QueryRow(ctx, `
		SELECT sub_id, sub_obj::text FROM main
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&subID, &subObj)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("запись не найдена")
	}
	if err != nil {
		return nil, err
	}
	if in.SubObj != subObj {
		return nil, fmt.Errorf("сменить sub_obj нельзя, текущий: %s", subObj)
	}

	_, err = tx.Exec(ctx, `
		UPDATE main SET title = $1, update_at = NOW() WHERE id = $2
	`, in.Title, id)
	if err != nil {
		return nil, err
	}

	if err := updateSatellite(ctx, tx, subID, id, in); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetMain(ctx, id)
}

// SoftDeleteMain помечает deleted_at у main и связанного сателлита.
func (s *Store) SoftDeleteMain(ctx context.Context, id int) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var subID *int
	var subObj models.SubObj
	err = tx.QueryRow(ctx, `
		SELECT sub_id, sub_obj::text FROM main
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&subID, &subObj)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	now := time.Now().UTC()
	tag, err := tx.Exec(ctx, `
		UPDATE main SET deleted_at = $1, update_at = $1 WHERE id = $2 AND deleted_at IS NULL
	`, now, id)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}

	if subID != nil {
		switch subObj {
		case models.SubObjTools:
			_, err = tx.Exec(ctx, `UPDATE tools SET deleted_at = $1, update_at = $1 WHERE id = $2 AND main_id = $3`, now, *subID, id)
		case models.SubObjTables:
			_, err = tx.Exec(ctx, `UPDATE "tables" SET deleted_at = $1, update_at = $1 WHERE id = $2 AND main_id = $3`, now, *subID, id)
		case models.SubObjChairs:
			_, err = tx.Exec(ctx, `UPDATE chairs SET deleted_at = $1, update_at = $1 WHERE id = $2 AND main_id = $3`, now, *subID, id)
		}
		if err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func insertSatellite(ctx context.Context, tx pgx.Tx, mainID int, in models.MainInput) (int, error) {
	var subID int
	switch in.SubObj {
	case models.SubObjTools:
		err := tx.QueryRow(ctx, `
			INSERT INTO tools (main_id, description1) VALUES ($1, $2) RETURNING id
		`, mainID, deref(in.Description1)).Scan(&subID)
		return subID, err
	case models.SubObjTables:
		err := tx.QueryRow(ctx, `
			INSERT INTO "tables" (main_id, description2) VALUES ($1, $2) RETURNING id
		`, mainID, deref(in.Description2)).Scan(&subID)
		return subID, err
	case models.SubObjChairs:
		if in.Type == nil {
			return 0, fmt.Errorf("для chairs нужно поле type")
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO chairs (main_id, description3, type) VALUES ($1, $2, $3) RETURNING id
		`, mainID, deref(in.Description3), string(*in.Type)).Scan(&subID)
		return subID, err
	default:
		return 0, fmt.Errorf("неизвестный sub_obj: %s", in.SubObj)
	}
}

func updateSatellite(ctx context.Context, tx pgx.Tx, subID, mainID int, in models.MainInput) error {
	switch in.SubObj {
	case models.SubObjTools:
		_, err := tx.Exec(ctx, `
			UPDATE tools SET description1 = $1, update_at = NOW()
			WHERE id = $2 AND main_id = $3 AND deleted_at IS NULL
		`, deref(in.Description1), subID, mainID)
		return err
	case models.SubObjTables:
		_, err := tx.Exec(ctx, `
			UPDATE "tables" SET description2 = $1, update_at = NOW()
			WHERE id = $2 AND main_id = $3 AND deleted_at IS NULL
		`, deref(in.Description2), subID, mainID)
		return err
	case models.SubObjChairs:
		if in.Type == nil {
			return fmt.Errorf("для chairs нужно поле type")
		}
		_, err := tx.Exec(ctx, `
			UPDATE chairs SET description3 = $1, type = $2, update_at = NOW()
			WHERE id = $3 AND main_id = $4 AND deleted_at IS NULL
		`, deref(in.Description3), string(*in.Type), subID, mainID)
		return err
	default:
		return fmt.Errorf("неизвестный sub_obj: %s", in.SubObj)
	}
}

func validateInput(in models.MainInput) error {
	if in.Title == "" {
		return fmt.Errorf("title обязателен")
	}
	switch in.SubObj {
	case models.SubObjTools, models.SubObjTables, models.SubObjChairs:
	default:
		return fmt.Errorf("sub_obj должен быть tools|tables|chairs")
	}
	if in.SubObj == models.SubObjChairs {
		if in.Type == nil {
			return fmt.Errorf("type обязателен для chairs")
		}
		if *in.Type != models.ChairTypeABC && *in.Type != models.ChairTypeCDE {
			return fmt.Errorf("type должен быть abc или cde")
		}
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
