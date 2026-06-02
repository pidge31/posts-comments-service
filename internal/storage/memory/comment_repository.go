package memory

import (
	"context"
	"sort"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/ports"
)

var _ ports.CommentRepository = (*CommentRepository)(nil)

type CommentRepository struct {
	store *Store
}

func NewCommentRepository(store *Store) *CommentRepository {
	return &CommentRepository{
		store: store,
	}
}

func (r *CommentRepository) Create(ctx context.Context, comment domain.Comment) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.store.mu.Lock()
	defer r.store.mu.Unlock()

	if _, ok := r.store.comments[comment.ID]; ok {
		return domain.ErrAlreadyExists
	}

	r.store.comments[comment.ID] = comment

	key := makePostParentKey(comment.PostID, comment.ParentID)
	r.store.commentIDsByParent[key] = append(r.store.commentIDsByParent[key], comment.ID)

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

	comments := r.listCommentsByParent(postID, parentID)

	sort.Slice(comments, func(i, j int) bool {
		if comments[i].CreatedAt.Equal(comments[j].CreatedAt) {
			return comments[i].ID < comments[j].ID
		}

		return comments[i].CreatedAt.Before(comments[j].CreatedAt)
	})

	page := make([]domain.Comment, 0, limit+1)
	for _, comment := range comments {
		if cursor != nil && !isCommentAfterCursor(comment, cursor) {
			continue
		}

		page = append(page, comment)
		if len(page) == limit+1 {
			break
		}
	}

	var nextCursor *domain.CommentCursor
	if len(page) > limit {
		lastComment := page[limit-1]
		nextCursor = &domain.CommentCursor{
			CreatedAt: lastComment.CreatedAt,
			ID:        lastComment.ID,
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

func (r *CommentRepository) listCommentsByParent(postID string, parentID *string) []domain.Comment {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	key := makePostParentKey(postID, parentID)
	commentIDs := r.store.commentIDsByParent[key]

	comments := make([]domain.Comment, 0, len(commentIDs))
	for _, commentID := range commentIDs {
		comment, ok := r.store.comments[commentID]
		if ok {
			comments = append(comments, comment)
		}
	}

	return comments
}

func isCommentAfterCursor(comment domain.Comment, cursor *domain.CommentCursor) bool {
	if comment.CreatedAt.After(cursor.CreatedAt) {
		return true
	}

	if comment.CreatedAt.Equal(cursor.CreatedAt) && comment.ID > cursor.ID {
		return true
	}

	return false
}
