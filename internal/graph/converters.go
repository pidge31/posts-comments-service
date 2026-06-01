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

	return &model.Comment{
		ID:        comment.ID,
		PostID:    comment.PostID,
		ParentID:  parentID,
		AuthorID:  comment.AuthorID,
		Text:      comment.Text,
		CreatedAt: comment.CreatedAt,
	}
}

func postsToConnection(posts []domain.Post, hasNextPage bool) *model.PostConnection {
	edges := make([]*model.PostEdge, 0, len(posts))

	for _, post := range posts {
		modelPost := postToModel(post)

		edges = append(edges, &model.PostEdge{
			Cursor: encodeCursor(post.CreatedAt, post.ID),
			Node:   modelPost,
		})
	}

	return &model.PostConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			EndCursor:   endCursorFromPostEdges(edges),
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

func endCursorFromPostEdges(edges []*model.PostEdge) *string {
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
