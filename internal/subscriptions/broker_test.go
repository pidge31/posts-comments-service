package subscriptions_test

import (
	"context"
	"testing"
	"time"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/subscriptions"
)

func TestBroker_PublishCommentCreated(t *testing.T) {
	broker := subscriptions.NewBroker()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	comments, unsubscribe, err := broker.SubscribeToPostComments(ctx, "post-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer unsubscribe()

	comment := domain.Comment{
		ID:        "comment-1",
		PostID:    "post-1",
		AuthorID:  "user-1",
		Text:      "New comment",
		CreatedAt: time.Now().UTC(),
	}

	if err := broker.PublishCommentCreated(context.Background(), comment); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case receivedComment := <-comments:
		if receivedComment.ID != comment.ID {
			t.Fatalf("expected comment ID %q, got %q", comment.ID, receivedComment.ID)
		}

	case <-time.After(time.Second):
		t.Fatal("expected comment to be published")
	}
}

func TestBroker_DoesNotPublishToDifferentPost(t *testing.T) {
	broker := subscriptions.NewBroker()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	comments, unsubscribe, err := broker.SubscribeToPostComments(ctx, "post-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer unsubscribe()

	comment := domain.Comment{
		ID:        "comment-1",
		PostID:    "post-2",
		AuthorID:  "user-1",
		Text:      "New comment",
		CreatedAt: time.Now().UTC(),
	}

	if err := broker.PublishCommentCreated(context.Background(), comment); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	assertNoComment(t, comments)
}

func TestBroker_SlowSubscriberDoesNotBlockPublisher(t *testing.T) {
	broker := subscriptions.NewBroker()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	comments, unsubscribe, err := broker.SubscribeToPostComments(ctx, "post-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer unsubscribe()

	for i := 0; i < 32; i++ {
		err := broker.PublishCommentCreated(context.Background(), domain.Comment{
			ID:     "comment",
			PostID: "post-1",
		})
		if err != nil {
			t.Fatalf("publish comment %d: %v", i, err)
		}
	}

	select {
	case <-comments:
	case <-time.After(time.Second):
		t.Fatal("expected at least one queued comment")
	}
}

func assertNoComment(t *testing.T, comments <-chan domain.Comment) {
	t.Helper()

	select {
	case receivedComment := <-comments:
		t.Fatalf("did not expect comment, got %q", receivedComment.ID)
	case <-time.After(100 * time.Millisecond):
		return
	}
}
