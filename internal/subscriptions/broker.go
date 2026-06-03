package subscriptions

import (
	"context"
	"sync"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

const subscriberBufferSize = 16

type Broker struct {
	mu sync.RWMutex

	// inner map: channel → once-safe closer (removes from map + closes channel)
	subscribers map[string]map[chan domain.Comment]func()
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string]map[chan domain.Comment]func()),
	}
}

func (b *Broker) PublishCommentCreated(ctx context.Context, comment domain.Comment) error {
	b.mu.RLock()

	var slowClosers []func()

	for ch, closer := range b.subscribers[comment.PostID] {
		select {
		case ch <- comment:
		case <-ctx.Done():
			b.mu.RUnlock()
			return ctx.Err()
		default:
			// subscriber is too slow — collect its closer, do not block
			slowClosers = append(slowClosers, closer)
		}
	}

	b.mu.RUnlock()

	// close slow subscribers outside the read lock to avoid lock inversion
	for _, closer := range slowClosers {
		closer()
	}

	return nil
}

func (b *Broker) SubscribeToPostComments(
	ctx context.Context,
	postID string,
) (<-chan domain.Comment, func(), error) {
	ch := make(chan domain.Comment, subscriberBufferSize)

	var once sync.Once

	closer := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()

			if subs, ok := b.subscribers[postID]; ok {
				delete(subs, ch)

				if len(subs) == 0 {
					delete(b.subscribers, postID)
				}
			}

			close(ch)
		})
	}

	b.mu.Lock()

	if b.subscribers[postID] == nil {
		b.subscribers[postID] = make(map[chan domain.Comment]func())
	}

	b.subscribers[postID][ch] = closer

	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		closer()
	}()

	return ch, closer, nil
}
