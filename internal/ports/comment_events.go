package ports

import (
	"context"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

type CommentEventPublisher interface {
	PublishCommentCreated(ctx context.Context, comment domain.Comment) error
}

type CommentEventSubscriber interface {
	SubscribeToPostComments(
		ctx context.Context,
		postID string,
	) (<-chan domain.Comment, func(), error)
}
