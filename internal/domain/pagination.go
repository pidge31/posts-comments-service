package domain

import "time"

const (
	DefaultPostPageSize = 10
	MaxPostPageSize     = 30

	DefaultCommentPageSize = 20
	MaxCommentPageSize     = 50
)

type PageInfo struct {
	EndCursor   *string
	HasNextPage bool
}

type PostCursor struct {
	CreatedAt time.Time
	ID        string
}

type CommentCursor struct {
	CreatedAt time.Time
	ID        string
}
