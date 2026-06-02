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
) ([]domain.Post, *domain.PostCursor, error) {
	if limit <= 0 {
		return []domain.Post{}, nil, nil
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
			body,
			comments_enabled,
			created_at,
			updated_at
		FROM posts
		WHERE (
			$1::timestamptz IS NULL
			OR (created_at, id) < ($1::timestamptz, $2::uuid)
		)
		ORDER BY created_at DESC, id DESC
		LIMIT $3
		`,
		cursorCreatedAt,
		cursorID,
		limit+1,
	)
	if err != nil {
		return nil, nil, mapPostgresError(err)
	}
	defer rows.Close()

	posts := make([]domain.Post, 0, limit+1)

	for rows.Next() {
		var post domain.Post

		if err := rows.Scan(
			&post.ID,
			&post.AuthorID,
			&post.Title,
			&post.Body,
			&post.CommentsEnabled,
			&post.CreatedAt,
			&post.UpdatedAt,
		); err != nil {
			return nil, nil, mapPostgresError(err)
		}

		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, mapPostgresError(err)
	}

	var nextCursor *domain.PostCursor
	if len(posts) > limit {
		lastPost := posts[limit-1]
		nextCursor = &domain.PostCursor{
			CreatedAt: lastPost.CreatedAt,
			ID:        lastPost.ID,
		}

		posts = posts[:limit]
	}

	return posts, nextCursor, nil
}

func (r *PostRepository) SetCommentsEnabled(
	ctx context.Context,
	postID string,
	enabled bool,
	updatedAt time.Time,
) error {
	commandTag, err := r.pool.Exec(
		ctx,
		`
		UPDATE posts
		SET
			comments_enabled = $1,
			updated_at = $2
		WHERE id = $3::uuid
		`,
		enabled,
		updatedAt,
		postID,
	)
	if err != nil {
		return mapPostgresError(err)
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrPostNotFound
	}

	return nil
}
