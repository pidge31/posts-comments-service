package memory

import (
	"context"
	"sort"
	"time"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/ports"
)

var _ ports.CommentRepository = (*CommentRepository)(nil)

type CommentRepository struct {
	store *Store
}

func NewCommentRepository(store *Store) *CommentRepository {
	return &CommentRepository{store: store}
}

func (r *CommentRepository) Create(ctx context.Context, comment domain.Comment) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if comment.ParentID != nil {
		parent, ok := r.store.comments[*comment.ParentID]
		if !ok || parent.DeletedAt != nil {
			return domain.ErrCommentNotFound
		}

		if parent.PostID != comment.PostID {
			return domain.ErrInvalidParentComment
		}
	}

	if _, ok := r.store.comments[comment.ID]; ok {
		return domain.ErrAlreadyExists
	}

	r.store.comments[comment.ID] = comment

	key := makePostParentKey(comment.PostID, comment.ParentID)
	entry := commentIndexEntry{CreatedAt: comment.CreatedAt, ID: comment.ID}
	r.store.commentsByParent[key] = insertSortedEntry(r.store.commentsByParent[key], entry)

	return nil
}

func (r *CommentRepository) GetByID(ctx context.Context, id string) (*domain.Comment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	comment, ok := r.store.comments[id]
	if !ok {
		return nil, domain.ErrCommentNotFound
	}

	return &comment, nil
}

func (r *CommentRepository) Delete(ctx context.Context, commentID string, authorID string, deletedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	comment, ok := r.store.comments[commentID]
	if !ok {
		return domain.ErrCommentNotFound
	}

	if comment.AuthorID != authorID {
		return domain.ErrForbidden
	}

	if comment.DeletedAt != nil {
		return domain.ErrCommentNotFound
	}

	comment.DeletedAt = &deletedAt
	r.store.comments[commentID] = comment

	return nil
}

func (r *CommentRepository) ListByPostAndParent(
	ctx context.Context,
	postID string,
	parentID *string,
	limit int,
	cursor *domain.CommentCursor,
) ([]domain.Comment, *domain.CommentCursor, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	if limit <= 0 {
		return []domain.Comment{}, nil, nil
	}

	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	key := makePostParentKey(postID, parentID)
	entries := r.store.commentsByParent[key]

	// binary search: first entry strictly after cursor
	start := 0
	if cursor != nil {
		start = sort.Search(len(entries), func(i int) bool {
			e := entries[i]
			if e.CreatedAt.Equal(cursor.CreatedAt) {
				return e.ID > cursor.ID
			}
			return e.CreatedAt.After(cursor.CreatedAt)
		})
	}

	// load only limit+1 comments from the found position
	end := start + limit + 1
	if end > len(entries) {
		end = len(entries)
	}

	page := make([]domain.Comment, 0, end-start)
	for _, entry := range entries[start:end] {
		if comment, ok := r.store.comments[entry.ID]; ok {
			page = append(page, comment)
		}
	}

	var nextCursor *domain.CommentCursor
	if len(page) > limit {
		last := page[limit-1]
		nextCursor = &domain.CommentCursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		}
		page = page[:limit]
	}

	return page, nextCursor, nil
}

func (r *CommentRepository) ListByPostAndParents(
	ctx context.Context,
	requests []ports.CommentListRequest,
) ([]ports.CommentListPage, error) {
	pages := make([]ports.CommentListPage, 0, len(requests))

	for _, request := range requests {
		comments, nextCursor, err := r.ListByPostAndParent(
			ctx,
			request.PostID,
			request.ParentID,
			request.Limit,
			request.Cursor,
		)
		if err != nil {
			return nil, err
		}

		pages = append(pages, ports.CommentListPage{
			Comments:   comments,
			NextCursor: nextCursor,
		})
	}

	return pages, nil
}

// insertSortedEntry inserts entry into a slice sorted ascending by
// (CreatedAt, ID), preserving the order. O(log n) search + O(n) shift.
func insertSortedEntry(entries []commentIndexEntry, entry commentIndexEntry) []commentIndexEntry {
	pos := sort.Search(len(entries), func(i int) bool {
		e := entries[i]
		if e.CreatedAt.Equal(entry.CreatedAt) {
			return e.ID >= entry.ID
		}
		return e.CreatedAt.After(entry.CreatedAt)
	})

	entries = append(entries, commentIndexEntry{})
	copy(entries[pos+1:], entries[pos:])
	entries[pos] = entry

	return entries
}
