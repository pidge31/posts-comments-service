package domain

import "time"

const MaxCommentTextLength = 2000

type Comment struct {
	ID        string
	PostID    string
	ParentID  *string
	AuthorID  string
	Text      string
	CreatedAt time.Time
}
