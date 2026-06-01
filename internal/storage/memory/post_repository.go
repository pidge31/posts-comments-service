package memory

import (
	"context"
	"sort"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/ports"
)

var _ ports.PostRepository = (*PostRepository)(nil)

type PostRepository struct {
	store *Store
}

func NewPostRepository(store *Store) *PostRepository {
	return &PostRepository{
		store: store,
	}
}

func (r *PostRepository) Create(ctx context.Context, post domain.Post) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, ok := r.store.posts[post.ID]; ok {
		return domain.ErrAlreadyExists
	}

	r.store.posts[post.ID] = post

	return nil
}

func (r *PostRepository) GetByID(ctx context.Context, id string) (*domain.Post, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	post, ok := r.store.posts[id]
	if !ok {
		return nil, domain.ErrNotFound
	}

	return &post, nil
}

func (r *PostRepository) List(
	ctx context.Context,
	limit int,
	cursor *domain.PostCursor,
) ([]domain.Post, *domain.PostCursor, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	if limit <= 0 {
		return []domain.Post{}, nil, nil
	}

	posts := r.listPosts()

	sort.Slice(posts, func(i, j int) bool {
		if posts[i].CreatedAt.Equal(posts[j].CreatedAt) {
			return posts[i].ID > posts[j].ID
		}

		return posts[i].CreatedAt.After(posts[j].CreatedAt)
	})

	page := make([]domain.Post, 0, limit+1)
	for _, post := range posts {
		if cursor != nil && !isPostAfterCursor(post, cursor) {
			continue
		}

		page = append(page, post)
		if len(page) == limit+1 {
			break
		}
	}

	var nextCursor *domain.PostCursor
	if len(page) > limit {
		lastPost := page[limit-1]
		nextCursor = &domain.PostCursor{
			CreatedAt: lastPost.CreatedAt,
			ID:        lastPost.ID,
		}

		page = page[:limit]
	}

	return page, nextCursor, nil
}

func (r *PostRepository) UpdateCommentsEnabled(
	ctx context.Context,
	postID string,
	enabled bool,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	post, ok := r.store.posts[postID]
	if !ok {
		return domain.ErrNotFound
	}

	post.CommentsEnabled = enabled
	r.store.posts[postID] = post

	return nil
}

func (r *PostRepository) listPosts() []domain.Post {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	posts := make([]domain.Post, 0, len(r.store.posts))
	for _, post := range r.store.posts {
		posts = append(posts, post)
	}

	return posts
}

func isPostAfterCursor(post domain.Post, cursor *domain.PostCursor) bool {
	if post.CreatedAt.Before(cursor.CreatedAt) {
		return true
	}

	if post.CreatedAt.Equal(cursor.CreatedAt) && post.ID < cursor.ID {
		return true
	}

	return false
}
