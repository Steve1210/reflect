package internal

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	DB *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{DB: db}
}

func (s *Store) InsertReflection(ctx context.Context, r Reflection) (int64, error) {
	var id int64

	query := `
		INSERT INTO reflections (title, tags, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := s.DB.QueryRow(
		ctx,
		query,
		r.Title,
		r.Tags,
		r.Body,
		r.CreatedAt,
		r.UpdatedAt,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *Store) FetchReflectionByID(ctx context.Context, id int64) (Reflection, error) {
	var r Reflection
	query := `SELECT id, title, tags, body, created_at, updated_at FROM reflections WHERE id = $1`
	err := s.DB.QueryRow(ctx, query, id).Scan(&r.Id, &r.Title, &r.Tags, &r.Body, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return Reflection{}, err
	}
	return r, nil
}

func (s *Store) UpdateReflection(ctx context.Context, id int64, r Reflection) error {
	query := `UPDATE reflections SET title=$1, tags=$2, body=$3, updated_at=$4 WHERE id=$5`
	tag, err := s.DB.Exec(ctx, query, r.Title, r.Tags, r.Body, r.UpdatedAt, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) DeleteReflection(ctx context.Context, id int64) error {
	query := `DELETE FROM reflections WHERE id = $1`
	tag, err := s.DB.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) FetchAllMetadata(ctx context.Context) ([]ReflectionHeader, error) {
	var headers []ReflectionHeader
	query := `
		SELECT id, title, tags, created_at, updated_at
		FROM reflections
		ORDER BY created_at DESC
	`

	rows, err := s.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var header ReflectionHeader
		err := rows.Scan(&header.Id, &header.Title, &header.Tags, &header.CreatedAt, &header.UpdatedAt)
		if err != nil {
			// log and return the error rather than silently skipping
			log.Printf("fetch metadata scan error: %v", err)
			return nil, err
		}
		headers = append(headers, header)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return headers, nil
}

func (s *Store) FetchAllMetadataWithFilters(ctx context.Context, filters FilterOptions) ([]ReflectionHeader, error) {
	var headers []ReflectionHeader

	var where []string
	var args []interface{}
	argID := 1

	if filters.CreatedAfter != 0 {
		where = append(where, fmt.Sprintf("created_at >= $%d", argID))
		args = append(args, filters.CreatedAfter)
		argID++
	}
	if filters.CreatedBefore != 0 {
		where = append(where, fmt.Sprintf("created_at < $%d", argID))
		args = append(args, filters.CreatedBefore)
		argID++
	}
	if filters.UpdatedAfter != 0 {
		where = append(where, fmt.Sprintf("updated_at >= $%d", argID))
		args = append(args, filters.UpdatedAfter)
		argID++
	}
	if filters.UpdatedBefore != 0 {
		where = append(where, fmt.Sprintf("updated_at < $%d", argID))
		args = append(args, filters.UpdatedBefore)
		argID++
	}

	query := `SELECT id, title, tags, created_at, updated_at
			  FROM reflections`

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var header ReflectionHeader
		err := rows.Scan(&header.Id, &header.Title, &header.Tags, &header.CreatedAt, &header.UpdatedAt)
		if err != nil {
			log.Printf("fetch metadata with filters scan error: %v", err)
			return nil, err
		}
		headers = append(headers, header)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return headers, nil
}
