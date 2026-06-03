package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pidge31/posts-comments-service/internal/app"
	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/storage/memory"
)

func TestPostService_CreatePost(t *testing.T) {
	postService, _ := newTestServices()

	post, err := postService.CreatePost(context.Background(), app.CreatePostInput{
		AuthorID:        "author-1",
		Title:           " First post ",
		Body:            " Post body ",
		CommentsEnabled: true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if post.ID == "" {
		t.Fatal("expected post ID to be set")
	}

	if post.AuthorID != "author-1" {
		t.Fatalf("expected author ID to be trimmed and saved, got %q", post.AuthorID)
	}

	if post.Title != "First post" {
		t.Fatalf("expected title to be trimmed, got %q", post.Title)
	}

	if post.Body != "Post body" {
		t.Fatalf("expected body to be trimmed, got %q", post.Body)
	}

	if !post.CommentsEnabled {
		t.Fatal("expected comments to be enabled")
	}

	if post.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	if post.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}
}

func TestPostService_CreatePost_InvalidInput(t *testing.T) {
	postService, _ := newTestServices()

	_, err := postService.CreatePost(context.Background(), app.CreatePostInput{
		AuthorID:        "author-1",
		Title:           "   ",
		Body:            "Post body",
		CommentsEnabled: true,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestPostService_CreatePost_TitleTooLong(t *testing.T) {
	postService, _ := newTestServices()

	_, err := postService.CreatePost(context.Background(), app.CreatePostInput{
		AuthorID:        "author-1",
		Title:           strings.Repeat("a", domain.MaxPostTitleLength+1),
		Body:            "Post body",
		CommentsEnabled: true,
	})
	if !errors.Is(err, domain.ErrPostTitleTooLong) {
		t.Fatalf("expected ErrPostTitleTooLong, got %v", err)
	}
}

func TestPostService_CreatePost_BodyTooLong(t *testing.T) {
	postService, _ := newTestServices()

	_, err := postService.CreatePost(context.Background(), app.CreatePostInput{
		AuthorID:        "author-1",
		Title:           "Post title",
		Body:            strings.Repeat("a", domain.MaxPostBodyLength+1),
		CommentsEnabled: true,
	})
	if !errors.Is(err, domain.ErrPostBodyTooLong) {
		t.Fatalf("expected ErrPostBodyTooLong, got %v", err)
	}
}

func TestPostService_SetCommentsEnabled(t *testing.T) {
	postService, _ := newTestServices()

	post := createTestPost(t, postService)

	updatedPost, err := postService.SetCommentsEnabled(
		context.Background(),
		post.ID,
		post.AuthorID,
		false,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updatedPost.CommentsEnabled {
		t.Fatal("expected comments to be disabled")
	}

	if !updatedPost.UpdatedAt.After(post.UpdatedAt) && !updatedPost.UpdatedAt.Equal(post.UpdatedAt) {
		t.Fatal("expected UpdatedAt not to move backwards")
	}
}

func TestPostService_SetCommentsEnabled_Forbidden(t *testing.T) {
	postService, _ := newTestServices()

	post := createTestPost(t, postService)

	_, err := postService.SetCommentsEnabled(
		context.Background(),
		post.ID,
		"another-author",
		false,
	)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestPostService_DeletePost(t *testing.T) {
	postService, _ := newTestServices()
	post := createTestPost(t, postService)

	if err := postService.DeletePost(context.Background(), post.ID, post.AuthorID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err := postService.GetPost(context.Background(), post.ID)
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound after delete, got %v", err)
	}
}

func TestPostService_DeletePost_Forbidden(t *testing.T) {
	postService, _ := newTestServices()
	post := createTestPost(t, postService)

	err := postService.DeletePost(context.Background(), post.ID, "another-author")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func newTestServices() (*app.PostService, *app.CommentService) {
	store := memory.NewStore()

	postRepository := memory.NewPostRepository(store)
	commentRepository := memory.NewCommentRepository(store)

	postService := app.NewPostService(postRepository)
	commentService := app.NewCommentService(postRepository, commentRepository)

	return postService, commentService
}

func createTestPost(t *testing.T, postService *app.PostService) *domain.Post {
	t.Helper()

	post, err := postService.CreatePost(context.Background(), app.CreatePostInput{
		AuthorID:        "author-1",
		Title:           "Test post",
		Body:            "Test body",
		CommentsEnabled: true,
	})
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	return post
}
