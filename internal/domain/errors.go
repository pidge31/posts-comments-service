package domain

import "errors"

var (
	ErrAlreadyExists     = errors.New("already exists")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidPagination = errors.New("invalid pagination")
	ErrPostNotFound      = errors.New("post not found")

	ErrCommentTooLong       = errors.New("comment text is too long")
	ErrCommentsDisabled     = errors.New("comments are disabled")
	ErrCommentNotFound      = errors.New("comment not found")
	ErrInvalidParentComment = errors.New("parent comment does not belong to post")
)
