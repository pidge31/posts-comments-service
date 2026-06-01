package service

import (
	"context"
	"errors"
	"testing"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/storage/memory"
)

func TestPostServiceSetCommentsEnabledAllowsAuthor(t *testing.T) {
	t.Parallel()

	repo := memory.NewPostRepository(memory.NewStore())
	service := NewPostService(repo)

	post := domain.Post{
		ID:              "post-1",
		AuthorID:        "author-1",
		CommentsEnabled: true,
	}

	if err := repo.Create(context.Background(), post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	err := service.SetCommentsEnabled(context.Background(), "post-1", "author-1", false)
	if err != nil {
		t.Fatalf("set comments enabled: %v", err)
	}

	updatedPost, err := repo.GetByID(context.Background(), "post-1")
	if err != nil {
		t.Fatalf("get post: %v", err)
	}

	if updatedPost.CommentsEnabled {
		t.Fatal("comments are still enabled")
	}
}

func TestPostServiceSetCommentsEnabledRejectsNonAuthor(t *testing.T) {
	t.Parallel()

	repo := memory.NewPostRepository(memory.NewStore())
	service := NewPostService(repo)

	post := domain.Post{
		ID:              "post-1",
		AuthorID:        "author-1",
		CommentsEnabled: true,
	}

	if err := repo.Create(context.Background(), post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	err := service.SetCommentsEnabled(context.Background(), "post-1", "author-2", false)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("set comments enabled by another author: got %v, want %v", err, domain.ErrForbidden)
	}
}
