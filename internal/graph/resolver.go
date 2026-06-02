package graph

import (
	"github.com/pidge31/posts-comments-service/internal/app"
	"github.com/pidge31/posts-comments-service/internal/ports"
)

type Resolver struct {
	postService            *app.PostService
	commentService         *app.CommentService
	commentEventSubscriber ports.CommentEventSubscriber
}

func NewResolver(
	postService *app.PostService,
	commentService *app.CommentService,
	commentEventSubscriber ports.CommentEventSubscriber,
) *Resolver {
	return &Resolver{
		postService:            postService,
		commentService:         commentService,
		commentEventSubscriber: commentEventSubscriber,
	}
}
