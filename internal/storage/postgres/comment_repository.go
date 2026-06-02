package postgres

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/ports"
)

var _ ports.CommentRepository = (*CommentRepository)(nil)

type CommentRepository struct {
	pool *pgxpool.Pool
}

func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{
		pool: pool,
	}
}

func (r *CommentRepository) Create(ctx context.Context, comment domain.Comment) error {
	var parentID any
	if comment.ParentID != nil {
		parentID = *comment.ParentID
	}

	_, err := r.pool.Exec(
		ctx,
		`
		INSERT INTO comments (
			id,
			post_id,
			parent_id,
			author_id,
			text,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6)
		`,
		comment.ID,
		comment.PostID,
		parentID,
		comment.AuthorID,
		comment.Text,
		comment.CreatedAt,
	)

	return mapPostgresError(err)
}

func (r *CommentRepository) GetByID(ctx context.Context, id string) (*domain.Comment, error) {
	var comment domain.Comment
	var parentID sql.NullString

	err := r.pool.QueryRow(
		ctx,
		`
		SELECT
			id::text,
			post_id::text,
			parent_id::text,
			author_id,
			text,
			created_at
		FROM comments
		WHERE id = $1::uuid
		`,
		id,
	).Scan(
		&comment.ID,
		&comment.PostID,
		&parentID,
		&comment.AuthorID,
		&comment.Text,
		&comment.CreatedAt,
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}

	if parentID.Valid {
		comment.ParentID = &parentID.String
	}

	return &comment, nil
}

func (r *CommentRepository) ListByPostAndParent(
	ctx context.Context,
	postID string,
	parentID *string,
	limit int,
	cursor *domain.CommentCursor,
) ([]domain.Comment, *domain.CommentCursor, error) {
	if limit <= 0 {
		return []domain.Comment{}, nil, nil
	}

	var parentIDValue any
	if parentID != nil {
		parentIDValue = *parentID
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
			post_id::text,
			parent_id::text,
			author_id,
			text,
			created_at
		FROM comments
		WHERE post_id = $1::uuid
		  AND parent_id IS NOT DISTINCT FROM $2::uuid
		  AND (
		  	$3::timestamptz IS NULL
		  	OR (created_at, id) > ($3::timestamptz, $4::uuid)
		  )
		ORDER BY created_at ASC, id ASC
		LIMIT $5
		`,
		postID,
		parentIDValue,
		cursorCreatedAt,
		cursorID,
		limit+1,
	)
	if err != nil {
		return nil, nil, mapPostgresError(err)
	}
	defer rows.Close()

	comments := make([]domain.Comment, 0, limit+1)

	for rows.Next() {
		var comment domain.Comment
		var scannedParentID sql.NullString

		if err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&scannedParentID,
			&comment.AuthorID,
			&comment.Text,
			&comment.CreatedAt,
		); err != nil {
			return nil, nil, mapPostgresError(err)
		}

		if scannedParentID.Valid {
			comment.ParentID = &scannedParentID.String
		}

		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, mapPostgresError(err)
	}

	var nextCursor *domain.CommentCursor
	if len(comments) > limit {
		lastComment := comments[limit-1]
		nextCursor = &domain.CommentCursor{
			CreatedAt: lastComment.CreatedAt,
			ID:        lastComment.ID,
		}

		comments = comments[:limit]
	}

	return comments, nextCursor, nil
}
