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
	postRepository        ports.PostRepository
	commentRepository     ports.CommentRepository
	commentEventPublisher ports.CommentEventPublisher
}

type AddCommentInput struct {
	PostID   string
	ParentID *string
	AuthorID string
	Text     string
}

type CommentPageRequest struct {
	PostID   string
	ParentID *string
	Limit    int
	Cursor   *domain.CommentCursor
}

type CommentPage struct {
	Comments   []domain.Comment
	NextCursor *domain.CommentCursor
}

func NewCommentService(
	postRepository ports.PostRepository,
	commentRepository ports.CommentRepository,
	commentEventPublisher ...ports.CommentEventPublisher,
) *CommentService {
	publisher := ports.CommentEventPublisher(noopCommentEventPublisher{})

	if len(commentEventPublisher) > 0 && commentEventPublisher[0] != nil {
		publisher = commentEventPublisher[0]
	}

	return &CommentService{
		postRepository:        postRepository,
		commentRepository:     commentRepository,
		commentEventPublisher: publisher,
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

	if err := s.commentEventPublisher.PublishCommentCreated(ctx, comment); err != nil {
		return nil, err
	}

	return &comment, nil
}

func (s *CommentService) DeleteComment(ctx context.Context, commentID string, authorID string) error {
	commentID = strings.TrimSpace(commentID)
	authorID = strings.TrimSpace(authorID)

	if commentID == "" || authorID == "" {
		return domain.ErrInvalidInput
	}

	return s.commentRepository.Delete(ctx, commentID, authorID, time.Now().UTC())
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

	if _, err := s.postRepository.GetByID(ctx, postID); err != nil {
		return nil, nil, err
	}

	return s.ListCommentsForExistingPost(ctx, postID, parentID, limit, cursor)
}

func (s *CommentService) ListCommentsForExistingPost(
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

func (s *CommentService) ListCommentPagesForExistingPosts(
	ctx context.Context,
	requests []CommentPageRequest,
) ([]CommentPage, error) {
	repositoryRequests := make([]ports.CommentListRequest, 0, len(requests))

	for _, request := range requests {
		postID := strings.TrimSpace(request.PostID)
		if postID == "" {
			return nil, domain.ErrInvalidInput
		}

		repositoryRequests = append(repositoryRequests, ports.CommentListRequest{
			PostID:   postID,
			ParentID: normalizeOptionalID(request.ParentID),
			Limit:    normalizePageLimit(request.Limit),
			Cursor:   request.Cursor,
		})
	}

	repositoryPages, err := s.commentRepository.ListByPostAndParents(ctx, repositoryRequests)
	if err != nil {
		return nil, err
	}

	if len(repositoryPages) != len(requests) {
		return nil, domain.ErrInvalidInput
	}

	pages := make([]CommentPage, 0, len(repositoryPages))
	for _, repositoryPage := range repositoryPages {
		pages = append(pages, CommentPage{
			Comments:   repositoryPage.Comments,
			NextCursor: repositoryPage.NextCursor,
		})
	}

	return pages, nil
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

type noopCommentEventPublisher struct{}

func (noopCommentEventPublisher) PublishCommentCreated(ctx context.Context, comment domain.Comment) error {
	return nil
}
