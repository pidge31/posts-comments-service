package graph

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gorilla/websocket"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/pidge31/posts-comments-service/internal/app"
	"github.com/pidge31/posts-comments-service/internal/domain"
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

	server.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
		gqlErr := graphql.DefaultErrorPresenter(ctx, err)

		switch {
		case errors.Is(err, domain.ErrForbidden):
			gqlErr.Message = "Доступ запрещён: вы не являетесь автором"
		case errors.Is(err, domain.ErrPostNotFound):
			gqlErr.Message = "Пост не найден"
		case errors.Is(err, domain.ErrCommentNotFound):
			gqlErr.Message = "Комментарий не найден"
		case errors.Is(err, domain.ErrCommentsDisabled):
			gqlErr.Message = "Комментарии к этому посту отключены автором"
		case errors.Is(err, domain.ErrCommentTooLong):
			gqlErr.Message = "Текст комментария превышает 2000 символов"
		case errors.Is(err, domain.ErrPostTitleTooLong):
			gqlErr.Message = "Заголовок поста превышает 200 символов"
		case errors.Is(err, domain.ErrPostBodyTooLong):
			gqlErr.Message = "Текст поста превышает 10 000 символов"
		case errors.Is(err, domain.ErrInvalidParentComment):
			gqlErr.Message = "Родительский комментарий не принадлежит этому посту"
		case errors.Is(err, domain.ErrInvalidInput):
			gqlErr.Message = "Некорректные данные запроса"
		case errors.Is(err, domain.ErrInvalidCursor):
			gqlErr.Message = "Некорректный курсор пагинации"
		}

		return gqlErr
	})

	server.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		return next(withCommentPageLoader(ctx, newCommentPageLoader(commentService)))
	})

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
