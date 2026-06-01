package domain

import "time"

const (
	MaxPostTitleLength = 200
	MaxPostBodyLength  = 10000
	PostExcerptLength  = 300
)

type Post struct {
	ID              string
	AuthorID        string
	Title           string
	Body            string
	CommentsEnabled bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type PostPreview struct {
	ID              string
	AuthorID        string
	Title           string
	Excerpt         string
	CommentsEnabled bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func MakePostExcerpt(body string) string {
	runes := []rune(body)

	if len(runes) <= PostExcerptLength {
		return body
	}

	return string(runes[:PostExcerptLength]) + "..."
}

func NewPostPreview(post Post) PostPreview {
	return PostPreview{
		ID:              post.ID,
		AuthorID:        post.AuthorID,
		Title:           post.Title,
		Excerpt:         MakePostExcerpt(post.Body),
		CommentsEnabled: post.CommentsEnabled,
		CreatedAt:       post.CreatedAt,
		UpdatedAt:       post.UpdatedAt,
	}
}
