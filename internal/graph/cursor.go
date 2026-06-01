package graph

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

type cursorPayload struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func encodeCursor(createdAt time.Time, id string) string {
	payload := cursorPayload{
		CreatedAt: createdAt,
		ID:        id,
	}

	data, _ := json.Marshal(payload)

	return base64.RawURLEncoding.EncodeToString(data)
}

func decodePostCursor(value *string) (*domain.PostCursor, error) {
	payload, err := decodeCursor(value)
	if err != nil {
		return nil, err
	}

	if payload == nil {
		return nil, nil
	}

	return &domain.PostCursor{
		CreatedAt: payload.CreatedAt,
		ID:        payload.ID,
	}, nil
}

func decodeCommentCursor(value *string) (*domain.CommentCursor, error) {
	payload, err := decodeCursor(value)
	if err != nil {
		return nil, err
	}

	if payload == nil {
		return nil, nil
	}

	return &domain.CommentCursor{
		CreatedAt: payload.CreatedAt,
		ID:        payload.ID,
	}, nil
}

func decodeCursor(value *string) (*cursorPayload, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, domain.ErrInvalidCursor
	}

	var payload cursorPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, domain.ErrInvalidCursor
	}

	if payload.ID == "" || payload.CreatedAt.IsZero() {
		return nil, domain.ErrInvalidCursor
	}

	return &payload, nil
}
