package domain

import "time"

const (
	DefaultPageSize = 20
	MaxPageSize     = 50
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
