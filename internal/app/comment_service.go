package app

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/ports"
)

type CommentService struct {
	postRepository    ports.PostRepository
	commentRepository ports.CommentRepository
}

type AddCommentInput struct {
	PostID   string
	ParentID *string
	AuthorID string
	Text     string
}

func NewCommentService(
	postRepository ports.PostRepository,
	commentRepository ports.CommentRepository,
) *CommentService {
	return &CommentService{
		postRepository:    postRepository,
		commentRepository: commentRepository,
	}
}

func (s *CommentService) AddComment(ctx context.Context, input AddCommentInput) (*domain.Comment, error) {
	postID := strings.TrimSpace(input.PostID)
	authorID := strings.TrimSpace(input.AuthorID)
	text := strings.TrimSpace(input.Text)

	if postID == "" || authorID == "" || text == "" {
		return nil, domain.ErrInvalidInput
	}

	if utf8.RuneCountInString(text) > domain.MaxCommentTextLength {
		return nil, domain.ErrCommentTooLong
	}

	post, err := s.postRepository.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	if !post.CommentsEnabled {
		return nil, domain.ErrCommentsDisabled
	}

	parentID := normalizeOptionalID(input.ParentID)

	if parentID != nil {
		parentComment, err := s.commentRepository.GetByID(ctx, *parentID)
		if err != nil {
			return nil, err
		}

		if parentComment.PostID != postID {
			return nil, domain.ErrInvalidParentComment
		}
	}

	comment := domain.Comment{
		ID:        uuid.NewString(),
		PostID:    postID,
		ParentID:  parentID,
		AuthorID:  authorID,
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.commentRepository.Create(ctx, comment); err != nil {
		return nil, err
	}

	return &comment, nil
}

func (s *CommentService) GetComment(ctx context.Context, id string) (*domain.Comment, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, domain.ErrInvalidInput
	}

	return s.commentRepository.GetByID(ctx, id)
}

func (s *CommentService) ListComments(
	ctx context.Context,
	postID string,
	parentID *string,
	limit int,
	cursor *domain.CommentCursor,
) ([]domain.Comment, *domain.CommentCursor, error) {
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return nil, nil, domain.ErrInvalidInput
	}

	limit = normalizePageLimit(limit)
	parentID = normalizeOptionalID(parentID)

	return s.commentRepository.ListByPostAndParent(ctx, postID, parentID, limit, cursor)
}

func normalizeOptionalID(id *string) *string {
	if id == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*id)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
