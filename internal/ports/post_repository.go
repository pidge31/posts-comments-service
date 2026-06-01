package ports

import (
	"context"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

type PostRepository interface {
	Create(ctx context.Context, post domain.Post) error

	GetByID(ctx context.Context, id string) (*domain.Post, error)

	List(
		ctx context.Context,
		limit int,
		cursor *domain.PostCursor,
	) ([]domain.Post, *domain.PostCursor, error)

	UpdateCommentsEnabled(
		ctx context.Context,
		postID string,
		enabled bool,
	) error
}
