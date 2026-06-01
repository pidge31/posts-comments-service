package app

import "github.com/pidge31/posts-comments-service/internal/domain"

func normalizePageLimit(limit int) int {
	if limit <= 0 {
		return domain.DefaultPageSize
	}

	if limit > domain.MaxPageSize {
		return domain.MaxPageSize
	}

	return limit
}
