package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/ports"
)

var _ ports.PostRepository = (*PostRepository)(nil)

type PostRepository struct {
	pool *pgxpool.Pool
}

func NewPostRepository(pool *pgxpool.Pool) *PostRepository {
	return &PostRepository{
		pool: pool,
	}
}

func (r *PostRepository) Create(ctx context.Context, post domain.Post) error {
	_, err := r.pool.Exec(
		ctx,
		`
		INSERT INTO posts (
			id,
			author_id,
			title,
			body,
			comments_enabled,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
		`,
		post.ID,
		post.AuthorID,
		post.Title,
		post.Body,
		post.CommentsEnabled,
		post.CreatedAt,
		post.UpdatedAt,
	)

	return mapPostgresError(err)
}

func (r *PostRepository) GetByID(ctx context.Context, id string) (*domain.Post, error) {
	var post domain.Post

	err := r.pool.QueryRow(
		ctx,
		`
		SELECT
			id::text,
			author_id,
			title,
			body,
			comments_enabled,
			created_at,
			updated_at
		FROM posts
		WHERE id = $1::uuid
		`,
		id,
	).Scan(
		&post.ID,
		&post.AuthorID,
		&post.Title,
		&post.Body,
		&post.CommentsEnabled,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}

	return &post, nil
}

func (r *PostRepository) List(
	ctx context.Context,
	limit int,
	cursor *domain.PostCursor,
) ([]domain.PostPreview, *domain.PostCursor, error) {
	if limit <= 0 {
		return []domain.PostPreview{}, nil, nil
	}

	var cursorCreatedAt any
	var cursorID any

	if cursor != nil {
		cursorCreatedAt = cursor.CreatedAt
		cursorID = cursor.ID
	}

	rows, err := r.pool.Query(
		ctx,
		`
		SELECT
			id::text,
			author_id,
			title,
			SUBSTRING(body, 1, $3),
			comments_enabled,
			created_at,
			updated_at
		FROM posts
		WHERE (
			$1::timestamptz IS NULL
			OR (created_at, id) < ($1::timestamptz, $2::uuid)
		)
		ORDER BY created_at DESC, id DESC
		LIMIT $4
		`,
		cursorCreatedAt,
		cursorID,
		domain.PostExcerptLength+1,
		limit+1,
	)
	if err != nil {
		return nil, nil, mapPostgresError(err)
	}
	defer rows.Close()

	previews := make([]domain.PostPreview, 0, limit+1)

	for rows.Next() {
		var preview domain.PostPreview
		var rawExcerpt string

		if err := rows.Scan(
			&preview.ID,
			&preview.AuthorID,
			&preview.Title,
			&rawExcerpt,
			&preview.CommentsEnabled,
			&preview.CreatedAt,
			&preview.UpdatedAt,
		); err != nil {
			return nil, nil, mapPostgresError(err)
		}

		preview.Excerpt = domain.MakePostExcerpt(rawExcerpt)
		previews = append(previews, preview)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, mapPostgresError(err)
	}

	var nextCursor *domain.PostCursor
	if len(previews) > limit {
		last := previews[limit-1]
		nextCursor = &domain.PostCursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		}

		previews = previews[:limit]
	}

	return previews, nextCursor, nil
}

func (r *PostRepository) Delete(ctx context.Context, postID string, authorID string) error {
	tag, err := r.pool.Exec(
		ctx,
		`DELETE FROM posts WHERE id = $1::uuid AND author_id = $2`,
		postID,
		authorID,
	)
	if err != nil {
		return mapPostgresError(err)
	}

	if tag.RowsAffected() == 0 {
		if _, err := r.GetByID(ctx, postID); err != nil {
			return err
		}

		return domain.ErrForbidden
	}

	return nil
}

func (r *PostRepository) SetCommentsEnabled(
	ctx context.Context,
	postID string,
	authorID string,
	enabled bool,
	updatedAt time.Time,
) error {
	tag, err := r.pool.Exec(
		ctx,
		`
		UPDATE posts
		SET
			comments_enabled = $1,
			updated_at = $2
		WHERE id = $3::uuid
		  AND author_id = $4
		`,
		enabled,
		updatedAt,
		postID,
		authorID,
	)
	if err != nil {
		return mapPostgresError(err)
	}

	if tag.RowsAffected() == 0 {
		if _, err := r.GetByID(ctx, postID); err != nil {
			return err
		}

		return domain.ErrForbidden
	}

	return nil
}
