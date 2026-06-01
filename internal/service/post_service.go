package service

import (
	"context"

	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/ports"
)

type PostService struct {
	posts ports.PostRepository
}

func NewPostService(posts ports.PostRepository) *PostService {
	return &PostService{
		posts: posts,
	}
}

func (s *PostService) SetCommentsEnabled(
	ctx context.Context,
	postID string,
	authorID string,
	enabled bool,
) error {
	post, err := s.posts.GetByID(ctx, postID)
	if err != nil {
		return err
	}

	if post.AuthorID != authorID {
		return domain.ErrForbidden
	}

	return s.posts.UpdateCommentsEnabled(ctx, postID, enabled)
}
