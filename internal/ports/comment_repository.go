package ports

import (
	"context"
	"time"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

type CommentRepository interface {
	Create(ctx context.Context, comment domain.Comment) error

	GetByID(ctx context.Context, id string) (*domain.Comment, error)

	Delete(ctx context.Context, commentID string, authorID string, deletedAt time.Time) error

	ListByPostAndParent(
		ctx context.Context,
		postID string,
		parentID *string,
		limit int,
		cursor *domain.CommentCursor,
	) ([]domain.Comment, *domain.CommentCursor, error)

	ListByPostAndParents(
		ctx context.Context,
		requests []CommentListRequest,
	) ([]CommentListPage, error)
}

type CommentListRequest struct {
	PostID   string
	ParentID *string
	Limit    int
	Cursor   *domain.CommentCursor
}

type CommentListPage struct {
	Comments   []domain.Comment
	NextCursor *domain.CommentCursor
}
