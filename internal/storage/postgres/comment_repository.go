package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/ports"
)

var _ ports.CommentRepository = (*CommentRepository)(nil)

type CommentRepository struct {
	pool *pgxpool.Pool
}

func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{pool: pool}
}

func (r *CommentRepository) Create(ctx context.Context, comment domain.Comment) error {
	if comment.ParentID == nil {
		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO comments (id, post_id, parent_id, author_id, text, created_at)
			 VALUES ($1::uuid, $2::uuid, NULL, $3, $4, $5)`,
			comment.ID, comment.PostID, comment.AuthorID, comment.Text, comment.CreatedAt,
		)

		return mapPostgresCommentError(err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return mapPostgresCommentError(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Lock the parent row: blocks DELETE but not concurrent readers/writers.
	// AND deleted_at IS NULL prevents replies to soft-deleted comments.
	var parentPostID string
	err = tx.QueryRow(
		ctx,
		`SELECT post_id::text FROM comments
		 WHERE id = $1::uuid AND deleted_at IS NULL
		 FOR KEY SHARE`,
		*comment.ParentID,
	).Scan(&parentPostID)
	if err != nil {
		return mapPostgresCommentError(err)
	}

	if parentPostID != comment.PostID {
		return domain.ErrInvalidParentComment
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO comments (id, post_id, parent_id, author_id, text, created_at)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6)`,
		comment.ID, comment.PostID, *comment.ParentID, comment.AuthorID, comment.Text, comment.CreatedAt,
	)
	if err != nil {
		return mapPostgresCommentError(err)
	}

	return mapPostgresCommentError(tx.Commit(ctx))
}

func (r *CommentRepository) GetByID(ctx context.Context, id string) (*domain.Comment, error) {
	var comment domain.Comment
	var parentID sql.NullString
	var deletedAt sql.NullTime

	err := r.pool.QueryRow(
		ctx,
		`SELECT id::text, post_id::text, parent_id::text, author_id, text, created_at, deleted_at
		 FROM comments
		 WHERE id = $1::uuid`,
		id,
	).Scan(&comment.ID, &comment.PostID, &parentID, &comment.AuthorID, &comment.Text, &comment.CreatedAt, &deletedAt)
	if err != nil {
		return nil, mapPostgresCommentError(err)
	}

	if parentID.Valid {
		comment.ParentID = &parentID.String
	}

	if deletedAt.Valid {
		comment.DeletedAt = &deletedAt.Time
	}

	return &comment, nil
}

func (r *CommentRepository) Delete(ctx context.Context, commentID string, authorID string, deletedAt time.Time) error {
	tag, err := r.pool.Exec(
		ctx,
		`UPDATE comments
		 SET deleted_at = $1
		 WHERE id = $2::uuid AND author_id = $3 AND deleted_at IS NULL`,
		deletedAt, commentID, authorID,
	)
	if err != nil {
		return mapPostgresCommentError(err)
	}

	if tag.RowsAffected() == 0 {
		existing, err := r.GetByID(ctx, commentID)
		if err != nil {
			return err
		}

		if existing.DeletedAt != nil {
			return domain.ErrCommentNotFound
		}

		return domain.ErrForbidden
	}

	return nil
}

func (r *CommentRepository) ListByPostAndParent(
	ctx context.Context,
	postID string,
	parentID *string,
	limit int,
	cursor *domain.CommentCursor,
) ([]domain.Comment, *domain.CommentCursor, error) {
	pages, err := r.ListByPostAndParents(ctx, []ports.CommentListRequest{
		{PostID: postID, ParentID: parentID, Limit: limit, Cursor: cursor},
	})
	if err != nil {
		return nil, nil, err
	}

	if len(pages) == 0 {
		return []domain.Comment{}, nil, nil
	}

	return pages[0].Comments, pages[0].NextCursor, nil
}

func (r *CommentRepository) ListByPostAndParents(
	ctx context.Context,
	requests []ports.CommentListRequest,
) ([]ports.CommentListPage, error) {
	pages := make([]ports.CommentListPage, len(requests))

	if len(requests) == 0 {
		return pages, nil
	}

	activeRequests := make([]ports.CommentListRequest, 0, len(requests))
	activeIndexes := make([]int, 0, len(requests))

	for index, request := range requests {
		if request.Limit <= 0 {
			continue
		}

		activeRequests = append(activeRequests, request)
		activeIndexes = append(activeIndexes, index)
	}

	if len(activeRequests) == 0 {
		return pages, nil
	}

	values := make([]string, 0, len(activeRequests))
	args := make([]any, 0, len(activeRequests)*6)

	for index, request := range activeRequests {
		var parentIDValue any
		if request.ParentID != nil {
			parentIDValue = *request.ParentID
		}

		var cursorCreatedAt any
		var cursorID any

		if request.Cursor != nil {
			cursorCreatedAt = request.Cursor.CreatedAt
			cursorID = request.Cursor.ID
		}

		offset := len(args) + 1
		values = append(values, fmt.Sprintf(
			"($%d::int, $%d::uuid, $%d::uuid, $%d::timestamptz, $%d::uuid, $%d::int)",
			offset, offset+1, offset+2, offset+3, offset+4, offset+5,
		))
		args = append(args,
			activeIndexes[index],
			request.PostID,
			parentIDValue,
			cursorCreatedAt,
			cursorID,
			request.Limit,
		)
	}

	rows, err := r.pool.Query(
		ctx,
		`
		WITH requests (
			request_index,
			post_id,
			parent_id,
			cursor_created_at,
			cursor_id,
			page_limit
		) AS (
			VALUES `+strings.Join(values, ",")+`
		)
		SELECT
			requests.request_index,
			listed_comments.id::text,
			listed_comments.post_id::text,
			listed_comments.parent_id::text,
			listed_comments.author_id,
			listed_comments.text,
			listed_comments.created_at,
			listed_comments.deleted_at
		FROM requests
		JOIN LATERAL (
			SELECT id, post_id, parent_id, author_id, text, created_at, deleted_at
			FROM comments
			WHERE post_id = requests.post_id
			  AND parent_id IS NOT DISTINCT FROM requests.parent_id
			  AND (
			  	requests.cursor_created_at IS NULL
			  	OR (created_at, id) > (requests.cursor_created_at, requests.cursor_id)
			  )
			ORDER BY created_at ASC, id ASC
			LIMIT requests.page_limit + 1
		) AS listed_comments ON TRUE
		ORDER BY requests.request_index ASC, listed_comments.created_at ASC, listed_comments.id ASC
		`,
		args...,
	)
	if err != nil {
		return nil, mapPostgresCommentError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var requestIndex int
		var comment domain.Comment
		var scannedParentID sql.NullString
		var scannedDeletedAt sql.NullTime

		if err := rows.Scan(
			&requestIndex,
			&comment.ID,
			&comment.PostID,
			&scannedParentID,
			&comment.AuthorID,
			&comment.Text,
			&comment.CreatedAt,
			&scannedDeletedAt,
		); err != nil {
			return nil, mapPostgresCommentError(err)
		}

		if requestIndex < 0 || requestIndex >= len(pages) {
			return nil, domain.ErrInvalidInput
		}

		if scannedParentID.Valid {
			comment.ParentID = &scannedParentID.String
		}

		if scannedDeletedAt.Valid {
			comment.DeletedAt = &scannedDeletedAt.Time
		}

		pages[requestIndex].Comments = append(pages[requestIndex].Comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, mapPostgresCommentError(err)
	}

	for index, request := range requests {
		if len(pages[index].Comments) <= request.Limit {
			continue
		}

		last := pages[index].Comments[request.Limit-1]
		pages[index].NextCursor = &domain.CommentCursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		}
		pages[index].Comments = pages[index].Comments[:request.Limit]
	}

	return pages, nil
}
