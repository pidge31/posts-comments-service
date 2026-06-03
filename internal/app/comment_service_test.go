package app_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pidge31/posts-comments-service/internal/app"
	"github.com/pidge31/posts-comments-service/internal/domain"
)

func TestCommentService_AddRootComment(t *testing.T) {
	postService, commentService := newTestServices()
	post := createTestPost(t, postService)

	comment, err := commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		AuthorID: "comment-author-1",
		Text:     " First comment ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if comment.ID == "" {
		t.Fatal("expected comment ID to be set")
	}

	if comment.PostID != post.ID {
		t.Fatalf("expected post ID %q, got %q", post.ID, comment.PostID)
	}

	if comment.ParentID != nil {
		t.Fatal("expected root comment to have nil ParentID")
	}

	if comment.Text != "First comment" {
		t.Fatalf("expected text to be trimmed, got %q", comment.Text)
	}

	if comment.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
}

func TestCommentService_AddReply(t *testing.T) {
	postService, commentService := newTestServices()
	post := createTestPost(t, postService)

	parentComment, err := commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		AuthorID: "comment-author-1",
		Text:     "Parent comment",
	})
	if err != nil {
		t.Fatalf("failed to create parent comment: %v", err)
	}

	reply, err := commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		ParentID: &parentComment.ID,
		AuthorID: "comment-author-2",
		Text:     "Reply comment",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if reply.ParentID == nil {
		t.Fatal("expected reply to have ParentID")
	}

	if *reply.ParentID != parentComment.ID {
		t.Fatalf("expected ParentID %q, got %q", parentComment.ID, *reply.ParentID)
	}
}

func TestCommentService_AddComment_CommentsDisabled(t *testing.T) {
	postService, commentService := newTestServices()

	post, err := postService.CreatePost(context.Background(), app.CreatePostInput{
		AuthorID:        "author-1",
		Title:           "Post with disabled comments",
		Body:            "Body",
		CommentsEnabled: false,
	})
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	_, err = commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		AuthorID: "comment-author-1",
		Text:     "Comment",
	})
	if !errors.Is(err, domain.ErrCommentsDisabled) {
		t.Fatalf("expected ErrCommentsDisabled, got %v", err)
	}
}

func TestCommentService_AddComment_TextTooLong(t *testing.T) {
	postService, commentService := newTestServices()
	post := createTestPost(t, postService)

	_, err := commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		AuthorID: "comment-author-1",
		Text:     strings.Repeat("a", domain.MaxCommentTextLength+1),
	})
	if !errors.Is(err, domain.ErrCommentTooLong) {
		t.Fatalf("expected ErrCommentTooLong, got %v", err)
	}
}

func TestCommentService_AddComment_InvalidInput(t *testing.T) {
	_, commentService := newTestServices()

	_, err := commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   "",
		AuthorID: "comment-author-1",
		Text:     "Comment",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCommentService_AddReply_ParentFromAnotherPost(t *testing.T) {
	postService, commentService := newTestServices()

	firstPost := createTestPost(t, postService)

	secondPost, err := postService.CreatePost(context.Background(), app.CreatePostInput{
		AuthorID:        "author-2",
		Title:           "Second post",
		Body:            "Second body",
		CommentsEnabled: true,
	})
	if err != nil {
		t.Fatalf("failed to create second post: %v", err)
	}

	parentComment, err := commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   firstPost.ID,
		AuthorID: "comment-author-1",
		Text:     "Parent comment",
	})
	if err != nil {
		t.Fatalf("failed to create parent comment: %v", err)
	}

	_, err = commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   secondPost.ID,
		ParentID: &parentComment.ID,
		AuthorID: "comment-author-2",
		Text:     "Invalid reply",
	})
	if !errors.Is(err, domain.ErrInvalidParentComment) {
		t.Fatalf("expected ErrParentCommentInvalid, got %v", err)
	}
}

func TestCommentService_ListComments(t *testing.T) {
	postService, commentService := newTestServices()
	post := createTestPost(t, postService)

	firstComment, err := commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		AuthorID: "comment-author-1",
		Text:     "First comment",
	})
	if err != nil {
		t.Fatalf("failed to create first comment: %v", err)
	}

	_, err = commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		AuthorID: "comment-author-2",
		Text:     "Second comment",
	})
	if err != nil {
		t.Fatalf("failed to create second comment: %v", err)
	}

	_, err = commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		ParentID: &firstComment.ID,
		AuthorID: "comment-author-3",
		Text:     "Reply to first comment",
	})
	if err != nil {
		t.Fatalf("failed to create reply: %v", err)
	}

	rootComments, _, err := commentService.ListComments(
		context.Background(),
		post.ID,
		nil,
		10,
		nil,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(rootComments) != 2 {
		t.Fatalf("expected 2 root comments, got %d", len(rootComments))
	}

	replies, _, err := commentService.ListComments(
		context.Background(),
		post.ID,
		&firstComment.ID,
		10,
		nil,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(replies))
	}

	if replies[0].Text != "Reply to first comment" {
		t.Fatalf("unexpected reply text: %q", replies[0].Text)
	}
}

func TestCommentService_ListComments_UnknownPost(t *testing.T) {
	_, commentService := newTestServices()

	_, _, err := commentService.ListComments(
		context.Background(),
		"unknown-post",
		nil,
		10,
		nil,
	)
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestCommentService_ListComments_UsesCursorPagination(t *testing.T) {
	postService, commentService := newTestServices()
	post := createTestPost(t, postService)

	createRootComments(t, commentService, post.ID, 3)

	firstPage, cursor, err := commentService.ListComments(
		context.Background(),
		post.ID,
		nil,
		1,
		nil,
	)
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(firstPage) != 1 {
		t.Fatalf("expected 1 comment on first page, got %d", len(firstPage))
	}
	if cursor == nil {
		t.Fatal("expected first page cursor")
	}

	secondPage, cursor, err := commentService.ListComments(
		context.Background(),
		post.ID,
		nil,
		1,
		cursor,
	)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(secondPage) != 1 {
		t.Fatalf("expected 1 comment on second page, got %d", len(secondPage))
	}
	if cursor == nil {
		t.Fatal("expected second page cursor")
	}
	if secondPage[0].ID == firstPage[0].ID {
		t.Fatal("expected cursor to move to another comment")
	}

	thirdPage, cursor, err := commentService.ListComments(
		context.Background(),
		post.ID,
		nil,
		1,
		cursor,
	)
	if err != nil {
		t.Fatalf("list third page: %v", err)
	}
	if len(thirdPage) != 1 {
		t.Fatalf("expected 1 comment on third page, got %d", len(thirdPage))
	}
	if cursor != nil {
		t.Fatalf("expected no cursor after last page, got %#v", cursor)
	}
}

func TestCommentService_ListComments_UsesDefaultAndMaxLimit(t *testing.T) {
	postService, commentService := newTestServices()
	post := createTestPost(t, postService)

	createRootComments(t, commentService, post.ID, domain.MaxPageSize+1)

	defaultPage, defaultCursor, err := commentService.ListComments(
		context.Background(),
		post.ID,
		nil,
		0,
		nil,
	)
	if err != nil {
		t.Fatalf("list with default limit: %v", err)
	}
	if len(defaultPage) != domain.DefaultPageSize {
		t.Fatalf("expected default page size %d, got %d", domain.DefaultPageSize, len(defaultPage))
	}
	if defaultCursor == nil {
		t.Fatal("expected cursor for default page")
	}

	maxPage, maxCursor, err := commentService.ListComments(
		context.Background(),
		post.ID,
		nil,
		domain.MaxPageSize+100,
		nil,
	)
	if err != nil {
		t.Fatalf("list with max limit: %v", err)
	}
	if len(maxPage) != domain.MaxPageSize {
		t.Fatalf("expected max page size %d, got %d", domain.MaxPageSize, len(maxPage))
	}
	if maxCursor == nil {
		t.Fatal("expected cursor for max page")
	}
}

func TestCommentService_DeleteComment(t *testing.T) {
	postService, commentService := newTestServices()
	post := createTestPost(t, postService)

	comment, err := commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		AuthorID: "user-1",
		Text:     "To be deleted",
	})
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	if err := commentService.DeleteComment(context.Background(), comment.ID, "user-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// comment stays in list but is masked
	comments, _, err := commentService.ListComments(context.Background(), post.ID, nil, 10, nil)
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected comment to remain as placeholder, got %d comments", len(comments))
	}
	if comments[0].DeletedAt == nil {
		t.Fatal("expected DeletedAt to be set")
	}
}

func TestCommentService_DeleteComment_Forbidden(t *testing.T) {
	postService, commentService := newTestServices()
	post := createTestPost(t, postService)

	comment, err := commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		AuthorID: "user-1",
		Text:     "Comment",
	})
	if err != nil {
		t.Fatalf("add comment: %v", err)
	}

	err = commentService.DeleteComment(context.Background(), comment.ID, "other-user")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCommentService_CannotReplyToDeletedComment(t *testing.T) {
	postService, commentService := newTestServices()
	post := createTestPost(t, postService)

	parent, err := commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		AuthorID: "user-1",
		Text:     "Parent",
	})
	if err != nil {
		t.Fatalf("add parent: %v", err)
	}

	if err := commentService.DeleteComment(context.Background(), parent.ID, "user-1"); err != nil {
		t.Fatalf("delete parent: %v", err)
	}

	_, err = commentService.AddComment(context.Background(), app.AddCommentInput{
		PostID:   post.ID,
		ParentID: &parent.ID,
		AuthorID: "user-2",
		Text:     "Reply to deleted",
	})
	if !errors.Is(err, domain.ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
}

func createRootComments(t *testing.T, commentService *app.CommentService, postID string, count int) {
	t.Helper()

	for i := 0; i < count; i++ {
		_, err := commentService.AddComment(context.Background(), app.AddCommentInput{
			PostID:   postID,
			AuthorID: "comment-author",
			Text:     fmt.Sprintf("Comment %d", i+1),
		})
		if err != nil {
			t.Fatalf("create comment %d: %v", i, err)
		}
	}
}
