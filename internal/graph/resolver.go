package graph

import "github.com/pidge31/posts-comments-service/internal/app"

type Resolver struct {
	postService    *app.PostService
	commentService *app.CommentService
}

func NewResolver(
	postService *app.PostService,
	commentService *app.CommentService,
) *Resolver {
	return &Resolver{
		postService:    postService,
		commentService: commentService,
	}
}
