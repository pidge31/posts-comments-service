package integration_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/ports"
	"github.com/pidge31/posts-comments-service/internal/storage/postgres"
)

func TestPostgresRepositories(t *testing.T) {
	ctx := context.Background()
	pool := newPostgresPool(t, ctx)

	postRepository := postgres.NewPostRepository(pool)
	commentRepository := postgres.NewCommentRepository(pool)

	baseTime := time.Date(9999, 1, 1, 12, 0, 0, 0, time.UTC)
	firstPost := domain.Post{
		ID:              uuid.NewString(),
		AuthorID:        "author-1",
		Title:           "First post",
		Body:            "First body",
		CommentsEnabled: true,
		CreatedAt:       baseTime,
		UpdatedAt:       baseTime,
	}
	secondPost := domain.Post{
		ID:              uuid.NewString(),
		AuthorID:        "author-2",
		Title:           "Second post",
		Body:            "Second body",
		CommentsEnabled: true,
		CreatedAt:       baseTime.Add(time.Minute),
		UpdatedAt:       baseTime.Add(time.Minute),
	}

	deletePostAfterTest(t, pool, secondPost.ID)
	deletePostAfterTest(t, pool, firstPost.ID)

	if err := postRepository.Create(ctx, firstPost); err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if err := postRepository.Create(ctx, secondPost); err != nil {
		t.Fatalf("create second post: %v", err)
	}

	posts, cursor, err := postRepository.List(ctx, 1, nil)
	if err != nil {
		t.Fatalf("list first post page: %v", err)
	}
	if len(posts) != 1 || posts[0].ID != secondPost.ID {
		t.Fatalf("first post page: got IDs %v, want %q", postIDs(posts), secondPost.ID)
	}
	if cursor == nil {
		t.Fatal("expected cursor for next post page")
	}

	posts, _, err = postRepository.List(ctx, 10, cursor)
	if err != nil {
		t.Fatalf("list second post page: %v", err)
	}
	if !containsPostID(posts, firstPost.ID) {
		t.Fatalf("second post page does not contain %q: %v", firstPost.ID, postIDs(posts))
	}

	rootComment := domain.Comment{
		ID:        uuid.NewString(),
		PostID:    firstPost.ID,
		AuthorID:  "comment-author-1",
		Text:      "Root comment",
		CreatedAt: baseTime,
	}
	secondRootComment := domain.Comment{
		ID:        uuid.NewString(),
		PostID:    firstPost.ID,
		AuthorID:  "comment-author-2",
		Text:      "Second root comment",
		CreatedAt: baseTime.Add(time.Minute),
	}
	reply := domain.Comment{
		ID:        uuid.NewString(),
		PostID:    firstPost.ID,
		ParentID:  &rootComment.ID,
		AuthorID:  "comment-author-3",
		Text:      "Reply comment",
		CreatedAt: baseTime.Add(2 * time.Minute),
	}

	for _, comment := range []domain.Comment{rootComment, secondRootComment, reply} {
		if err := commentRepository.Create(ctx, comment); err != nil {
			t.Fatalf("create comment %q: %v", comment.ID, err)
		}
	}

	rootComments, nextCursor, err := commentRepository.ListByPostAndParent(ctx, firstPost.ID, nil, 1, nil)
	if err != nil {
		t.Fatalf("list first root comment page: %v", err)
	}
	if len(rootComments) != 1 || rootComments[0].ID != rootComment.ID {
		t.Fatalf("first root comment page: got IDs %v, want %q", commentIDs(rootComments), rootComment.ID)
	}
	if nextCursor == nil {
		t.Fatal("expected cursor for next root comment page")
	}

	rootComments, nextCursor, err = commentRepository.ListByPostAndParent(ctx, firstPost.ID, nil, 1, nextCursor)
	if err != nil {
		t.Fatalf("list second root comment page: %v", err)
	}
	if len(rootComments) != 1 || rootComments[0].ID != secondRootComment.ID {
		t.Fatalf("second root comment page: got IDs %v, want %q", commentIDs(rootComments), secondRootComment.ID)
	}
	if nextCursor != nil {
		t.Fatalf("expected no cursor after last root comment page, got %#v", nextCursor)
	}

	replies, _, err := commentRepository.ListByPostAndParent(ctx, firstPost.ID, &rootComment.ID, 10, nil)
	if err != nil {
		t.Fatalf("list replies: %v", err)
	}
	if len(replies) != 1 || replies[0].ID != reply.ID {
		t.Fatalf("replies: got IDs %v, want %q", commentIDs(replies), reply.ID)
	}

	pages, err := commentRepository.ListByPostAndParents(ctx, []ports.CommentListRequest{
		{
			PostID: firstPost.ID,
			Limit:  10,
		},
		{
			PostID:   firstPost.ID,
			ParentID: &rootComment.ID,
			Limit:    10,
		},
	})
	if err != nil {
		t.Fatalf("batch list comments: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("batch page count: got %d, want 2", len(pages))
	}
	if len(pages[0].Comments) != 2 {
		t.Fatalf("batch root comments: got IDs %v, want 2 comments", commentIDs(pages[0].Comments))
	}
	if len(pages[1].Comments) != 1 || pages[1].Comments[0].ID != reply.ID {
		t.Fatalf("batch replies: got IDs %v, want %q", commentIDs(pages[1].Comments), reply.ID)
	}
}

func TestPostgresDeletePost(t *testing.T) {
	ctx := context.Background()
	pool := newPostgresPool(t, ctx)

	postRepository := postgres.NewPostRepository(pool)
	commentRepository := postgres.NewCommentRepository(pool)

	now := time.Now().UTC()
	post := domain.Post{
		ID:              uuid.NewString(),
		AuthorID:        "author-1",
		Title:           "Post to delete",
		Body:            "Body",
		CommentsEnabled: true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	deletePostAfterTest(t, pool, post.ID)

	if err := postRepository.Create(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	comment := domain.Comment{
		ID:        uuid.NewString(),
		PostID:    post.ID,
		AuthorID:  "commenter",
		Text:      "Will be cascade-deleted",
		CreatedAt: now,
	}
	if err := commentRepository.Create(ctx, comment); err != nil {
		t.Fatalf("create comment: %v", err)
	}

	if err := postRepository.Delete(ctx, post.ID, post.AuthorID); err != nil {
		t.Fatalf("delete post: %v", err)
	}

	if _, err := postRepository.GetByID(ctx, post.ID); !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound after delete, got %v", err)
	}

	if _, err := commentRepository.GetByID(ctx, comment.ID); !errors.Is(err, domain.ErrCommentNotFound) {
		t.Fatalf("expected comment to be cascade-deleted, got %v", err)
	}
}

func TestPostgresDeletePost_Forbidden(t *testing.T) {
	ctx := context.Background()
	pool := newPostgresPool(t, ctx)

	postRepository := postgres.NewPostRepository(pool)

	now := time.Now().UTC()
	post := domain.Post{
		ID:        uuid.NewString(),
		AuthorID:  "author-1",
		Title:     "Post",
		Body:      "Body",
		CreatedAt: now,
		UpdatedAt: now,
	}
	deletePostAfterTest(t, pool, post.ID)

	if err := postRepository.Create(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	if err := postRepository.Delete(ctx, post.ID, "wrong-author"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	if err := postRepository.Delete(ctx, uuid.NewString(), "author-1"); !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound for missing post, got %v", err)
	}
}

func TestPostgresDeleteComment(t *testing.T) {
	ctx := context.Background()
	pool := newPostgresPool(t, ctx)

	postRepository := postgres.NewPostRepository(pool)
	commentRepository := postgres.NewCommentRepository(pool)

	now := time.Now().UTC()
	post := domain.Post{
		ID:              uuid.NewString(),
		AuthorID:        "author-1",
		Title:           "Post",
		Body:            "Body",
		CommentsEnabled: true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	deletePostAfterTest(t, pool, post.ID)

	if err := postRepository.Create(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	comment := domain.Comment{
		ID:        uuid.NewString(),
		PostID:    post.ID,
		AuthorID:  "commenter",
		Text:      "To be soft-deleted",
		CreatedAt: now,
	}
	if err := commentRepository.Create(ctx, comment); err != nil {
		t.Fatalf("create comment: %v", err)
	}

	deletedAt := time.Now().UTC()
	if err := commentRepository.Delete(ctx, comment.ID, comment.AuthorID, deletedAt); err != nil {
		t.Fatalf("delete comment: %v", err)
	}

	got, err := commentRepository.GetByID(ctx, comment.ID)
	if err != nil {
		t.Fatalf("get deleted comment: %v", err)
	}
	if got.DeletedAt == nil {
		t.Fatal("expected DeletedAt to be set")
	}

	comments, _, err := commentRepository.ListByPostAndParent(ctx, post.ID, nil, 10, nil)
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected soft-deleted comment to remain as placeholder, got %d", len(comments))
	}
	if comments[0].DeletedAt == nil {
		t.Fatal("expected listed comment to have DeletedAt set")
	}
}

func TestPostgresDeleteComment_Forbidden(t *testing.T) {
	ctx := context.Background()
	pool := newPostgresPool(t, ctx)

	postRepository := postgres.NewPostRepository(pool)
	commentRepository := postgres.NewCommentRepository(pool)

	now := time.Now().UTC()
	post := domain.Post{
		ID:              uuid.NewString(),
		AuthorID:        "author-1",
		Title:           "Post",
		Body:            "Body",
		CommentsEnabled: true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	deletePostAfterTest(t, pool, post.ID)

	if err := postRepository.Create(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	comment := domain.Comment{
		ID:        uuid.NewString(),
		PostID:    post.ID,
		AuthorID:  "commenter",
		Text:      "Comment",
		CreatedAt: now,
	}
	if err := commentRepository.Create(ctx, comment); err != nil {
		t.Fatalf("create comment: %v", err)
	}

	if err := commentRepository.Delete(ctx, comment.ID, "wrong-author", now); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("wrong author: expected ErrForbidden, got %v", err)
	}

	if err := commentRepository.Delete(ctx, uuid.NewString(), "commenter", now); !errors.Is(err, domain.ErrCommentNotFound) {
		t.Fatalf("missing comment: expected ErrCommentNotFound, got %v", err)
	}
}

func TestPostgresDeleteComment_AlreadyDeleted(t *testing.T) {
	ctx := context.Background()
	pool := newPostgresPool(t, ctx)

	postRepository := postgres.NewPostRepository(pool)
	commentRepository := postgres.NewCommentRepository(pool)

	now := time.Now().UTC()
	post := domain.Post{
		ID:              uuid.NewString(),
		AuthorID:        "author-1",
		Title:           "Post",
		Body:            "Body",
		CommentsEnabled: true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	deletePostAfterTest(t, pool, post.ID)

	if err := postRepository.Create(ctx, post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	comment := domain.Comment{
		ID:        uuid.NewString(),
		PostID:    post.ID,
		AuthorID:  "commenter",
		Text:      "Comment",
		CreatedAt: now,
	}
	if err := commentRepository.Create(ctx, comment); err != nil {
		t.Fatalf("create comment: %v", err)
	}

	if err := commentRepository.Delete(ctx, comment.ID, comment.AuthorID, now); err != nil {
		t.Fatalf("first delete: %v", err)
	}

	if err := commentRepository.Delete(ctx, comment.ID, comment.AuthorID, now); !errors.Is(err, domain.ErrCommentNotFound) {
		t.Fatalf("second delete: expected ErrCommentNotFound, got %v", err)
	}
}

func TestPostgresRepositoryErrors(t *testing.T) {
	ctx := context.Background()
	pool := newPostgresPool(t, ctx)

	postRepository := postgres.NewPostRepository(pool)
	commentRepository := postgres.NewCommentRepository(pool)

	missingID := uuid.NewString()

	if _, err := postRepository.GetByID(ctx, missingID); !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("missing post: got %v, want %v", err, domain.ErrPostNotFound)
	}

	if _, err := commentRepository.GetByID(ctx, missingID); !errors.Is(err, domain.ErrCommentNotFound) {
		t.Fatalf("missing comment: got %v, want %v", err, domain.ErrCommentNotFound)
	}

	if err := commentRepository.Create(ctx, domain.Comment{
		ID:        uuid.NewString(),
		PostID:    missingID,
		AuthorID:  "comment-author",
		Text:      "Comment for missing post",
		CreatedAt: time.Now().UTC(),
	}); !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("comment with missing post: got %v, want %v", err, domain.ErrPostNotFound)
	}
}

func newPostgresPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL or DATABASE_URL to run Postgres integration tests")
	}

	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	applyMigrations(t, ctx, pool)

	return pool
}

func applyMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	for _, migration := range []string{
		"001_create_posts.up.sql",
		"002_create_comments.up.sql",
		"003_add_comment_soft_delete.up.sql",
	} {
		path := filepath.Join("..", "..", "migrations", migration)
		query, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", migration, err)
		}

		if _, err := pool.Exec(ctx, string(query)); err != nil {
			t.Fatalf("apply migration %s: %v", migration, err)
		}
	}
}

func deletePostAfterTest(t *testing.T, pool *pgxpool.Pool, postID string) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, _ = pool.Exec(ctx, "DELETE FROM posts WHERE id = $1::uuid", postID)
	})
}

func postIDs(posts []domain.PostPreview) []string {
	ids := make([]string, 0, len(posts))
	for _, post := range posts {
		ids = append(ids, post.ID)
	}
	return ids
}

func containsPostID(posts []domain.PostPreview, id string) bool {
	for _, post := range posts {
		if post.ID == id {
			return true
		}
	}
	return false
}

func commentIDs(comments []domain.Comment) []string {
	ids := make([]string, 0, len(comments))
	for _, comment := range comments {
		ids = append(ids, comment.ID)
	}
	return ids
}
