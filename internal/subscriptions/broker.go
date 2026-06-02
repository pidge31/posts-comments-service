package subscriptions

import (
	"context"
	"sync"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

const subscriberBufferSize = 16

type Broker struct {
	mu sync.RWMutex

	subscribers map[string]map[chan domain.Comment]struct{}
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string]map[chan domain.Comment]struct{}),
	}
}

func (b *Broker) PublishCommentCreated(ctx context.Context, comment domain.Comment) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for subscriber := range b.subscribers[comment.PostID] {
		select {
		case subscriber <- comment:
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

func (b *Broker) SubscribeToPostComments(
	ctx context.Context,
	postID string,
) (<-chan domain.Comment, func(), error) {
	ch := make(chan domain.Comment, subscriberBufferSize)

	b.mu.Lock()

	if b.subscribers[postID] == nil {
		b.subscribers[postID] = make(map[chan domain.Comment]struct{})
	}

	b.subscribers[postID][ch] = struct{}{}

	b.mu.Unlock()

	var once sync.Once

	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()

			if subscribers, ok := b.subscribers[postID]; ok {
				delete(subscribers, ch)

				if len(subscribers) == 0 {
					delete(b.subscribers, postID)
				}
			}

			close(ch)
		})
	}

	go func() {
		<-ctx.Done()
		unsubscribe()
	}()

	return ch, unsubscribe, nil
}
