package graph

import (
	"context"
	"sync"
	"time"

	"github.com/pidge31/posts-comments-service/internal/app"
)

const commentPageBatchDelay = time.Millisecond

type commentPageLoaderContextKey struct{}

type commentPageLoader struct {
	commentService *app.CommentService

	mu        sync.Mutex
	pending   []commentPageLoad
	scheduled bool
}

type commentPageLoad struct {
	request  app.CommentPageRequest
	response chan commentPageLoadResult
}

type commentPageLoadResult struct {
	page app.CommentPage
	err  error
}

func newCommentPageLoader(commentService *app.CommentService) *commentPageLoader {
	return &commentPageLoader{
		commentService: commentService,
	}
}

func withCommentPageLoader(ctx context.Context, loader *commentPageLoader) context.Context {
	return context.WithValue(ctx, commentPageLoaderContextKey{}, loader)
}

func commentPageLoaderFromContext(ctx context.Context) *commentPageLoader {
	loader, _ := ctx.Value(commentPageLoaderContextKey{}).(*commentPageLoader)

	return loader
}

func loadCommentPage(
	ctx context.Context,
	commentService *app.CommentService,
	request app.CommentPageRequest,
) (app.CommentPage, error) {
	loader := commentPageLoaderFromContext(ctx)
	if loader == nil {
		comments, nextCursor, err := commentService.ListCommentsForExistingPost(
			ctx,
			request.PostID,
			request.ParentID,
			request.Limit,
			request.Cursor,
		)
		if err != nil {
			return app.CommentPage{}, err
		}

		return app.CommentPage{
			Comments:   comments,
			NextCursor: nextCursor,
		}, nil
	}

	return loader.Load(ctx, request)
}

func (l *commentPageLoader) Load(
	ctx context.Context,
	request app.CommentPageRequest,
) (app.CommentPage, error) {
	response := make(chan commentPageLoadResult, 1)

	l.mu.Lock()
	l.pending = append(l.pending, commentPageLoad{
		request:  request,
		response: response,
	})

	if !l.scheduled {
		l.scheduled = true
		go l.dispatch(ctx)
	}
	l.mu.Unlock()

	select {
	case result := <-response:
		return result.page, result.err

	case <-ctx.Done():
		return app.CommentPage{}, ctx.Err()
	}
}

func (l *commentPageLoader) dispatch(ctx context.Context) {
	timer := time.NewTimer(commentPageBatchDelay)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-ctx.Done():
	}

	l.mu.Lock()
	batch := l.pending
	l.pending = nil
	l.scheduled = false
	l.mu.Unlock()

	if len(batch) == 0 {
		return
	}

	requests := make([]app.CommentPageRequest, 0, len(batch))
	for _, load := range batch {
		requests = append(requests, load.request)
	}

	pages, err := l.commentService.ListCommentPagesForExistingPosts(ctx, requests)

	for index, load := range batch {
		result := commentPageLoadResult{err: err}
		if err == nil {
			result.page = pages[index]
		}

		select {
		case load.response <- result:
		default:
		}
	}
}
