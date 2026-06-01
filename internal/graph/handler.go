package graph

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/pidge31/posts-comments-service/internal/app"
	"github.com/pidge31/posts-comments-service/internal/graph/generated"
)

func NewHandler(
	postService *app.PostService,
	commentService *app.CommentService,
) http.Handler {
	resolver := NewResolver(postService, commentService)

	return handler.NewDefaultServer(
		generated.NewExecutableSchema(
			generated.Config{
				Resolvers: resolver,
			},
		),
	)
}
