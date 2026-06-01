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

type PostService struct {
	postRepository ports.PostRepository
}

type CreatePostInput struct {
	AuthorID        string
	Title           string
	Body            string
	CommentsEnabled bool
}

func NewPostService(postRepository ports.PostRepository) *PostService {
	return &PostService{
		postRepository: postRepository,
	}
}

func (s *PostService) CreatePost(ctx context.Context, input CreatePostInput) (*domain.Post, error) {
	authorID := strings.TrimSpace(input.AuthorID)
	title := strings.TrimSpace(input.Title)
	body := strings.TrimSpace(input.Body)

	if authorID == "" || title == "" || body == "" {
		return nil, domain.ErrInvalidInput
	}

	if utf8.RuneCountInString(title) > domain.MaxPostTitleLength {
		return nil, domain.ErrPostTitleTooLong
	}

	if utf8.RuneCountInString(body) > domain.MaxPostBodyLength {
		return nil, domain.ErrPostBodyTooLong
	}

	now := time.Now().UTC()

	post := domain.Post{
		ID:              uuid.NewString(),
		AuthorID:        authorID,
		Title:           title,
		Body:            body,
		CommentsEnabled: input.CommentsEnabled,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.postRepository.Create(ctx, post); err != nil {
		return nil, err
	}

	return &post, nil
}

func (s *PostService) GetPost(ctx context.Context, id string) (*domain.Post, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, domain.ErrInvalidInput
	}

	return s.postRepository.GetByID(ctx, id)
}

func (s *PostService) ListPosts(
	ctx context.Context,
	limit int,
	cursor *domain.PostCursor,
) ([]domain.Post, *domain.PostCursor, error) {
	limit = normalizePageLimit(limit)

	return s.postRepository.List(ctx, limit, cursor)
}

func (s *PostService) SetCommentsEnabled(
	ctx context.Context,
	postID string,
	authorID string,
	enabled bool,
) (*domain.Post, error) {
	postID = strings.TrimSpace(postID)
	authorID = strings.TrimSpace(authorID)

	if postID == "" || authorID == "" {
		return nil, domain.ErrInvalidInput
	}

	updatedAt := time.Now().UTC()

	post, err := s.postRepository.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	if post.AuthorID != authorID {
		return nil, domain.ErrForbidden
	}

	if err := s.postRepository.SetCommentsEnabled(ctx, postID, enabled, updatedAt); err != nil {
		return nil, err
	}

	return s.postRepository.GetByID(ctx, postID)
}
