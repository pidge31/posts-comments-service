package subscriptions

import (
	"context"
	"testing"
	"time"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

func TestBrokerDeliversCommentsToPostSubscribers(t *testing.T) {
	t.Parallel()

	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	postComments, unsubscribePost, err := broker.SubscribeToPostComments(ctx, "post-1")
	if err != nil {
		t.Fatalf("subscribe to post comments: %v", err)
	}
	defer unsubscribePost()

	otherPostComments, unsubscribeOtherPost, err := broker.SubscribeToPostComments(ctx, "post-2")
	if err != nil {
		t.Fatalf("subscribe to other post comments: %v", err)
	}
	defer unsubscribeOtherPost()

	comment := domain.Comment{
		ID:     "comment-1",
		PostID: "post-1",
		Text:   "First comment",
	}

	if err := broker.PublishCommentCreated(context.Background(), comment); err != nil {
		t.Fatalf("publish comment: %v", err)
	}

	select {
	case got := <-postComments:
		if got.ID != comment.ID {
			t.Fatalf("comment ID: got %q, want %q", got.ID, comment.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for post comment")
	}

	select {
	case got := <-otherPostComments:
		t.Fatalf("got comment for another post: %#v", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestBrokerStopsDeliveringAfterUnsubscribe(t *testing.T) {
	t.Parallel()

	broker := NewBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	comments, unsubscribe, err := broker.SubscribeToPostComments(ctx, "post-1")
	if err != nil {
		t.Fatalf("subscribe to post comments: %v", err)
	}

	unsubscribe()

	if err := broker.PublishCommentCreated(context.Background(), domain.Comment{
		ID:     "comment-1",
		PostID: "post-1",
	}); err != nil {
		t.Fatalf("publish comment: %v", err)
	}

	select {
	case _, ok := <-comments:
		if ok {
			t.Fatal("got comment after unsubscribe")
		}
	case <-time.After(time.Second):
		t.Fatal("subscription channel was not closed")
	}
}
