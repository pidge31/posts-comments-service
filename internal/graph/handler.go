package graph

import (
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gorilla/websocket"

	"github.com/pidge31/posts-comments-service/internal/app"
	"github.com/pidge31/posts-comments-service/internal/graph/generated"
	"github.com/pidge31/posts-comments-service/internal/ports"
)

func NewHandler(
	postService *app.PostService,
	commentService *app.CommentService,
	commentSubscriber ports.CommentEventSubscriber,
) http.Handler {
	resolver := NewResolver(postService, commentService, commentSubscriber)

	server := handler.New(
		generated.NewExecutableSchema(
			generated.Config{
				Resolvers: resolver,
			},
		),
	)

	server.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	})

	server.AddTransport(transport.Options{})
	server.AddTransport(transport.GET{})
	server.AddTransport(transport.POST{})

	return server
}
