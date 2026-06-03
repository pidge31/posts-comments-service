package ports

import (
	"context"
	"time"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

type PostRepository interface {
	Create(ctx context.Context, post domain.Post) error

	GetByID(ctx context.Context, id string) (*domain.Post, error)

	List(
		ctx context.Context,
		limit int,
		cursor *domain.PostCursor,
	) ([]domain.PostPreview, *domain.PostCursor, error)

	SetCommentsEnabled(
		ctx context.Context,
		postID string,
		authorID string,
		enabled bool,
		updatedAt time.Time,
	) error

	Delete(ctx context.Context, postID string, authorID string) error
}
