package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/storage/memory"
)

func TestCommentRepository_ListByPostAndParent(t *testing.T) {
	store := memory.NewStore()
	repository := memory.NewCommentRepository(store)

	baseTime := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)

	rootComment := domain.Comment{
		ID:        "comment-1",
		PostID:    "post-1",
		ParentID:  nil,
		AuthorID:  "user-1",
		Text:      "Root comment",
		CreatedAt: baseTime,
	}

	secondRootComment := domain.Comment{
		ID:        "comment-2",
		PostID:    "post-1",
		ParentID:  nil,
		AuthorID:  "user-2",
		Text:      "Second root comment",
		CreatedAt: baseTime.Add(time.Minute),
	}

	reply := domain.Comment{
		ID:        "comment-3",
		PostID:    "post-1",
		ParentID:  &rootComment.ID,
		AuthorID:  "user-3",
		Text:      "Reply",
		CreatedAt: baseTime.Add(2 * time.Minute),
	}

	for _, comment := range []domain.Comment{rootComment, secondRootComment, reply} {
		if err := repository.Create(context.Background(), comment); err != nil {
			t.Fatalf("failed to create comment: %v", err)
		}
	}

	rootComments, nextCursor, err := repository.ListByPostAndParent(
		context.Background(),
		"post-1",
		nil,
		10,
		nil,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if nextCursor != nil {
		t.Fatal("expected no next cursor")
	}

	if len(rootComments) != 2 {
		t.Fatalf("expected 2 root comments, got %d", len(rootComments))
	}

	if rootComments[0].ID != "comment-1" {
		t.Fatalf("expected first root comment first, got %q", rootComments[0].ID)
	}

	replies, nextCursor, err := repository.ListByPostAndParent(
		context.Background(),
		"post-1",
		&rootComment.ID,
		10,
		nil,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if nextCursor != nil {
		t.Fatal("expected no next cursor")
	}

	if len(replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(replies))
	}

	if replies[0].ID != "comment-3" {
		t.Fatalf("expected reply comment, got %q", replies[0].ID)
	}
}

func TestCommentRepository_ListByPostAndParent_WithPagination(t *testing.T) {
	store := memory.NewStore()
	repository := memory.NewCommentRepository(store)

	baseTime := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)

	for i, comment := range []domain.Comment{
		{
			ID:        "comment-1",
			PostID:    "post-1",
			AuthorID:  "user-1",
			Text:      "First",
			CreatedAt: baseTime,
		},
		{
			ID:        "comment-2",
			PostID:    "post-1",
			AuthorID:  "user-2",
			Text:      "Second",
			CreatedAt: baseTime.Add(time.Minute),
		},
		{
			ID:        "comment-3",
			PostID:    "post-1",
			AuthorID:  "user-3",
			Text:      "Third",
			CreatedAt: baseTime.Add(2 * time.Minute),
		},
	} {
		if err := repository.Create(context.Background(), comment); err != nil {
			t.Fatalf("failed to create comment %d: %v", i, err)
		}
	}

	firstPage, nextCursor, err := repository.ListByPostAndParent(
		context.Background(),
		"post-1",
		nil,
		2,
		nil,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(firstPage) != 2 {
		t.Fatalf("expected 2 comments on first page, got %d", len(firstPage))
	}

	if nextCursor == nil {
		t.Fatal("expected next cursor")
	}

	secondPage, nextCursor, err := repository.ListByPostAndParent(
		context.Background(),
		"post-1",
		nil,
		2,
		nextCursor,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(secondPage) != 1 {
		t.Fatalf("expected 1 comment on second page, got %d", len(secondPage))
	}

	if secondPage[0].ID != "comment-3" {
		t.Fatalf("expected third comment on second page, got %q", secondPage[0].ID)
	}

	if nextCursor != nil {
		t.Fatal("expected no next cursor on last page")
	}
}
