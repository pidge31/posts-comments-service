package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

func TestCommentRepositoryCreateRejectsDuplicateID(t *testing.T) {
	t.Parallel()

	repo := NewCommentRepository(NewStore())
	comment := domain.Comment{
		ID:        "comment-1",
		PostID:    "post-1",
		AuthorID:  "author-1",
		Text:      "First comment",
		CreatedAt: time.Now(),
	}

	if err := repo.Create(context.Background(), comment); err != nil {
		t.Fatalf("create comment: %v", err)
	}

	if err := repo.Create(context.Background(), comment); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("create duplicate comment: got %v, want %v", err, domain.ErrAlreadyExists)
	}
}

func TestCommentRepositoryKeepsRootAndChildCommentsSeparate(t *testing.T) {
	t.Parallel()

	repo := NewCommentRepository(NewStore())
	parentID := "comment-1"
	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	comments := []domain.Comment{
		{ID: parentID, PostID: "post-1", Text: "root", CreatedAt: baseTime},
		{ID: "comment-2", PostID: "post-1", ParentID: &parentID, Text: "child", CreatedAt: baseTime.Add(time.Minute)},
	}

	for _, comment := range comments {
		if err := repo.Create(context.Background(), comment); err != nil {
			t.Fatalf("create comment %q: %v", comment.ID, err)
		}
	}

	rootComments, _, err := repo.ListByPostAndParent(context.Background(), "post-1", nil, 10, nil)
	if err != nil {
		t.Fatalf("list root comments: %v", err)
	}
	assertCommentIDs(t, rootComments, []string{"comment-1"})

	childComments, _, err := repo.ListByPostAndParent(context.Background(), "post-1", &parentID, 10, nil)
	if err != nil {
		t.Fatalf("list child comments: %v", err)
	}
	assertCommentIDs(t, childComments, []string{"comment-2"})
}

func TestCommentRepositoryListUsesStableCursorOrder(t *testing.T) {
	t.Parallel()

	repo := NewCommentRepository(NewStore())
	baseTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	comments := []domain.Comment{
		{ID: "comment-3", PostID: "post-1", CreatedAt: baseTime.Add(time.Minute)},
		{ID: "comment-1", PostID: "post-1", CreatedAt: baseTime},
		{ID: "comment-2", PostID: "post-1", CreatedAt: baseTime},
	}

	for _, comment := range comments {
		if err := repo.Create(context.Background(), comment); err != nil {
			t.Fatalf("create comment %q: %v", comment.ID, err)
		}
	}

	firstPage, cursor, err := repo.ListByPostAndParent(context.Background(), "post-1", nil, 2, nil)
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}

	assertCommentIDs(t, firstPage, []string{"comment-1", "comment-2"})
	if cursor == nil {
		t.Fatal("first page cursor is nil")
	}

	secondPage, nextCursor, err := repo.ListByPostAndParent(context.Background(), "post-1", nil, 2, cursor)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}

	assertCommentIDs(t, secondPage, []string{"comment-3"})
	if nextCursor != nil {
		t.Fatalf("second page cursor: got %#v, want nil", nextCursor)
	}
}

func TestCommentRepositoryRespectsCanceledContext(t *testing.T) {
	t.Parallel()

	repo := NewCommentRepository(NewStore())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.Create(ctx, domain.Comment{ID: "comment-1"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("create with canceled context: got %v, want %v", err, context.Canceled)
	}
}

func assertCommentIDs(t *testing.T, comments []domain.Comment, want []string) {
	t.Helper()

	if len(comments) != len(want) {
		t.Fatalf("comment count: got %d, want %d", len(comments), len(want))
	}

	for i, comment := range comments {
		if comment.ID != want[i] {
			t.Fatalf("comment[%d]: got %q, want %q", i, comment.ID, want[i])
		}
	}
}
