package postgres

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

func mapPostgresError(err error) error {
	return mapPostgresPostError(err)
}

func mapPostgresPostError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrPostNotFound
	}

	return mapCommonPostgresError(err, domain.ErrPostNotFound)
}

func mapPostgresCommentError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrCommentNotFound
	}

	return mapCommonPostgresError(err, domain.ErrCommentNotFound)
}

func mapCommonPostgresError(err error, fallbackForeignKeyError error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.InvalidTextRepresentation:
			return domain.ErrInvalidInput

		case pgerrcode.UniqueViolation:
			return domain.ErrAlreadyExists

		case pgerrcode.ForeignKeyViolation:
			switch pgErr.ConstraintName {
			case "comments_post_id_fkey":
				return domain.ErrPostNotFound
			case "comments_parent_id_fkey":
				return domain.ErrCommentNotFound
			}

			if fallbackForeignKeyError != nil {
				return fallbackForeignKeyError
			}

			return domain.ErrPostNotFound

		case pgerrcode.CheckViolation:
			if pgErr.ConstraintName == "comments_text_length" {
				return domain.ErrCommentTooLong
			}

			return domain.ErrInvalidInput
		}
	}

	return err
}
