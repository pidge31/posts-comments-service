package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/storage/memory"
)

func TestPostRepositoryCreateRejectsDuplicateID(t *testing.T) {
	t.Parallel()

	repo := memory.NewPostRepository(memory.NewStore())
	post := domain.Post{
		ID:              "post-1",
		AuthorID:        "author-1",
		Title:           "First post",
		Body:            "Body",
		CommentsEnabled: true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := repo.Create(context.Background(), post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	if err := repo.Create(context.Background(), post); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("create duplicate post: got %v, want %v", err, domain.ErrAlreadyExists)
	}
}

func TestPostRepositoryListUsesStableCursorOrder(t *testing.T) {
	t.Parallel()

	repo := memory.NewPostRepository(memory.NewStore())
	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	posts := []domain.Post{
		{ID: "post-1", CreatedAt: baseTime.Add(-2 * time.Hour)},
		{ID: "post-2", CreatedAt: baseTime},
		{ID: "post-3", CreatedAt: baseTime.Add(-1 * time.Hour)},
		{ID: "post-4", CreatedAt: baseTime},
	}

	for _, post := range posts {
		if err := repo.Create(context.Background(), post); err != nil {
			t.Fatalf("create post %q: %v", post.ID, err)
		}
	}

	firstPage, cursor, err := repo.List(context.Background(), 2, nil)
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}

	assertPostIDs(t, firstPage, []string{"post-4", "post-2"})
	if cursor == nil {
		t.Fatal("first page cursor is nil")
	}

	secondPage, nextCursor, err := repo.List(context.Background(), 2, cursor)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}

	assertPostIDs(t, secondPage, []string{"post-3", "post-1"})
	if nextCursor != nil {
		t.Fatalf("second page cursor: got %#v, want nil", nextCursor)
	}
}

func TestPostRepositoryRespectsCanceledContext(t *testing.T) {
	t.Parallel()

	repo := memory.NewPostRepository(memory.NewStore())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.Create(ctx, domain.Post{ID: "post-1"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("create with canceled context: got %v, want %v", err, context.Canceled)
	}
}

func TestPostRepositorySetCommentsEnabled(t *testing.T) {
	t.Parallel()

	repo := memory.NewPostRepository(memory.NewStore())
	post := domain.Post{
		ID:              "post-1",
		AuthorID:        "author-1",
		CommentsEnabled: true,
	}

	if err := repo.Create(context.Background(), post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	if err := repo.SetCommentsEnabled(context.Background(), "post-1", "author-1", false, time.Now()); err != nil {
		t.Fatalf("update comments enabled: %v", err)
	}

	updatedPost, err := repo.GetByID(context.Background(), "post-1")
	if err != nil {
		t.Fatalf("get post: %v", err)
	}

	if updatedPost.CommentsEnabled {
		t.Fatal("comments are still enabled")
	}
}

func TestPostRepositorySetCommentsEnabledForbidden(t *testing.T) {
	t.Parallel()

	repo := memory.NewPostRepository(memory.NewStore())
	post := domain.Post{
		ID:              "post-1",
		AuthorID:        "author-1",
		CommentsEnabled: true,
	}

	if err := repo.Create(context.Background(), post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	err := repo.SetCommentsEnabled(context.Background(), "post-1", "other-author", false, time.Now())
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("wrong author: got %v, want %v", err, domain.ErrForbidden)
	}
}

func assertPostIDs(t *testing.T, posts []domain.PostPreview, want []string) {
	t.Helper()

	if len(posts) != len(want) {
		t.Fatalf("post count: got %d, want %d", len(posts), len(want))
	}

	for i, post := range posts {
		if post.ID != want[i] {
			t.Fatalf("post[%d]: got %q, want %q", i, post.ID, want[i])
		}
	}
}
