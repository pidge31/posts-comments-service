package graph

import (
	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/graph/model"
)

func postToModel(post domain.Post) *model.Post {
	return &model.Post{
		ID:              post.ID,
		AuthorID:        post.AuthorID,
		Title:           post.Title,
		Body:            post.Body,
		CommentsEnabled: post.CommentsEnabled,
		CreatedAt:       post.CreatedAt,
		UpdatedAt:       post.UpdatedAt,
	}
}

func commentToModel(comment domain.Comment) *model.Comment {
	var parentID *string
	if comment.ParentID != nil {
		value := *comment.ParentID
		parentID = &value
	}

	isDeleted := comment.DeletedAt != nil
	text := comment.Text
	authorID := comment.AuthorID

	if isDeleted {
		text = "[удалено]"
		authorID = ""
	}

	return &model.Comment{
		ID:        comment.ID,
		PostID:    comment.PostID,
		ParentID:  parentID,
		AuthorID:  authorID,
		Text:      text,
		CreatedAt: comment.CreatedAt,
		IsDeleted: isDeleted,
	}
}

func postPreviewToModel(preview domain.PostPreview) *model.PostPreview {
	return &model.PostPreview{
		ID:              preview.ID,
		AuthorID:        preview.AuthorID,
		Title:           preview.Title,
		Excerpt:         preview.Excerpt,
		CommentsEnabled: preview.CommentsEnabled,
		CreatedAt:       preview.CreatedAt,
		UpdatedAt:       preview.UpdatedAt,
	}
}

func postPreviewsToConnection(previews []domain.PostPreview, hasNextPage bool) *model.PostPreviewConnection {
	edges := make([]*model.PostPreviewEdge, 0, len(previews))

	for _, preview := range previews {
		edges = append(edges, &model.PostPreviewEdge{
			Cursor: encodeCursor(preview.CreatedAt, preview.ID),
			Node:   postPreviewToModel(preview),
		})
	}

	return &model.PostPreviewConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			EndCursor:   endCursorFromPreviewEdges(edges),
			HasNextPage: hasNextPage,
		},
	}
}

func commentsToConnection(comments []domain.Comment, hasNextPage bool) *model.CommentConnection {
	edges := make([]*model.CommentEdge, 0, len(comments))

	for _, comment := range comments {
		modelComment := commentToModel(comment)

		edges = append(edges, &model.CommentEdge{
			Cursor: encodeCursor(comment.CreatedAt, comment.ID),
			Node:   modelComment,
		})
	}

	return &model.CommentConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			EndCursor:   endCursorFromCommentEdges(edges),
			HasNextPage: hasNextPage,
		},
	}
}

func endCursorFromPreviewEdges(edges []*model.PostPreviewEdge) *string {
	if len(edges) == 0 {
		return nil
	}

	cursor := edges[len(edges)-1].Cursor

	return &cursor
}

func endCursorFromCommentEdges(edges []*model.CommentEdge) *string {
	if len(edges) == 0 {
		return nil
	}

	cursor := edges[len(edges)-1].Cursor

	return &cursor
}

func limitFromPointer(limit *int) int {
	if limit == nil {
		return domain.DefaultPageSize
	}

	return *limit
}
