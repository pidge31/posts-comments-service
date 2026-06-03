package graph_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pidge31/posts-comments-service/internal/app"
	"github.com/pidge31/posts-comments-service/internal/domain"
	"github.com/pidge31/posts-comments-service/internal/graph"
	"github.com/pidge31/posts-comments-service/internal/storage/memory"
	"github.com/pidge31/posts-comments-service/internal/subscriptions"
)

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func TestGraphQL_CreatePostAddCommentAndListComments(t *testing.T) {
	handler := newTestGraphQLHandler()

	createPostResponse := executeGraphQL(t, handler, `
		mutation {
			createPost(input: {
				authorID: "author-1"
				title: "First post"
				body: "Post body"
				commentsEnabled: true
			}) {
				id
				title
				commentsEnabled
			}
		}
	`)

	var createPostData struct {
		CreatePost struct {
			ID              string `json:"id"`
			Title           string `json:"title"`
			CommentsEnabled bool   `json:"commentsEnabled"`
		} `json:"createPost"`
	}

	unmarshalGraphQLData(t, createPostResponse, &createPostData)

	if createPostData.CreatePost.ID == "" {
		t.Fatal("expected created post ID")
	}

	if createPostData.CreatePost.Title != "First post" {
		t.Fatalf("unexpected post title: %q", createPostData.CreatePost.Title)
	}

	if !createPostData.CreatePost.CommentsEnabled {
		t.Fatal("expected comments to be enabled")
	}

	addCommentQuery := fmt.Sprintf(`
		mutation {
			addComment(input: {
				postID: %q
				authorID: "user-2"
				text: "First comment"
			}) {
				id
				postID
				parentID
				text
			}
		}
	`, createPostData.CreatePost.ID)

	addCommentResponse := executeGraphQL(t, handler, addCommentQuery)

	var addCommentData struct {
		AddComment struct {
			ID       string  `json:"id"`
			PostID   string  `json:"postID"`
			ParentID *string `json:"parentID"`
			Text     string  `json:"text"`
		} `json:"addComment"`
	}

	unmarshalGraphQLData(t, addCommentResponse, &addCommentData)

	if addCommentData.AddComment.ID == "" {
		t.Fatal("expected created comment ID")
	}

	if addCommentData.AddComment.PostID != createPostData.CreatePost.ID {
		t.Fatalf("expected post ID %q, got %q", createPostData.CreatePost.ID, addCommentData.AddComment.PostID)
	}

	if addCommentData.AddComment.ParentID != nil {
		t.Fatal("expected root comment to have nil parentID")
	}

	postQuery := fmt.Sprintf(`
		query {
			post(id: %q) {
				id
				title
				comments(first: 10) {
					edges {
						node {
							id
							text
							parentID
						}
					}
					pageInfo {
						hasNextPage
						endCursor
					}
				}
			}
		}
	`, createPostData.CreatePost.ID)

	postResponse := executeGraphQL(t, handler, postQuery)

	var postData struct {
		Post struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Comments struct {
				Edges []struct {
					Node struct {
						ID       string  `json:"id"`
						Text     string  `json:"text"`
						ParentID *string `json:"parentID"`
					} `json:"node"`
				} `json:"edges"`
				PageInfo struct {
					HasNextPage bool    `json:"hasNextPage"`
					EndCursor   *string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"comments"`
		} `json:"post"`
	}

	unmarshalGraphQLData(t, postResponse, &postData)

	if len(postData.Post.Comments.Edges) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(postData.Post.Comments.Edges))
	}

	if postData.Post.Comments.Edges[0].Node.Text != "First comment" {
		t.Fatalf("unexpected comment text: %q", postData.Post.Comments.Edges[0].Node.Text)
	}
}

func TestGraphQL_PostsReturnsExcerptNotFullBody(t *testing.T) {
	handler := newTestGraphQLHandler()

	longBody := strings.Repeat("а", 400)

	executeGraphQL(t, handler, fmt.Sprintf(`
		mutation {
			createPost(input: {
				authorID: "author-1"
				title: "Long post"
				body: %q
				commentsEnabled: true
			}) { id }
		}
	`, longBody))

	listResponse := executeGraphQL(t, handler, `
		query {
			posts(first: 1) {
				edges {
					node {
						id
						title
						excerpt
					}
				}
			}
		}
	`)

	var listData struct {
		Posts struct {
			Edges []struct {
				Node struct {
					ID      string `json:"id"`
					Title   string `json:"title"`
					Excerpt string `json:"excerpt"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"posts"`
	}

	unmarshalGraphQLData(t, listResponse, &listData)

	if len(listData.Posts.Edges) == 0 {
		t.Fatal("expected at least one post in list")
	}

	excerpt := listData.Posts.Edges[0].Node.Excerpt

	excerptRunes := []rune(excerpt)
	if len(excerptRunes) > domain.PostExcerptLength+3 {
		t.Fatalf("excerpt too long: %d runes", len(excerptRunes))
	}

	if !strings.HasSuffix(excerpt, "...") {
		t.Fatalf("expected excerpt to end with '...', got %q", excerpt)
	}
}

func TestGraphQL_AddCommentToDisabledPostReturnsError(t *testing.T) {
	handler := newTestGraphQLHandler()

	createPostResponse := executeGraphQL(t, handler, `
		mutation {
			createPost(input: {
				authorID: "author-1"
				title: "Closed post"
				body: "Post body"
				commentsEnabled: false
			}) {
				id
			}
		}
	`)

	var createPostData struct {
		CreatePost struct {
			ID string `json:"id"`
		} `json:"createPost"`
	}

	unmarshalGraphQLData(t, createPostResponse, &createPostData)

	addCommentQuery := fmt.Sprintf(`
		mutation {
			addComment(input: {
				postID: %q
				authorID: "user-2"
				text: "Should fail"
			}) {
				id
			}
		}
	`, createPostData.CreatePost.ID)

	response := executeGraphQLAllowErrors(t, handler, addCommentQuery)

	if len(response.Errors) == 0 {
		t.Fatal("expected GraphQL error")
	}

	if response.Errors[0].Message == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestGraphQL_DeletePost(t *testing.T) {
	handler := newTestGraphQLHandler()

	createResp := executeGraphQL(t, handler, `
		mutation {
			createPost(input: {
				authorID: "author-1"
				title: "Post to delete"
				body: "Body"
				commentsEnabled: true
			}) { id }
		}
	`)

	var createData struct {
		CreatePost struct{ ID string `json:"id"` } `json:"createPost"`
	}
	unmarshalGraphQLData(t, createResp, &createData)
	postID := createData.CreatePost.ID

	deleteResp := executeGraphQL(t, handler, fmt.Sprintf(`
		mutation { deletePost(postID: %q, authorID: "author-1") }
	`, postID))

	var deleteData struct {
		DeletePost bool `json:"deletePost"`
	}
	unmarshalGraphQLData(t, deleteResp, &deleteData)

	if !deleteData.DeletePost {
		t.Fatal("expected deletePost to return true")
	}

	getResp := executeGraphQLAllowErrors(t, handler, fmt.Sprintf(`
		query { post(id: %q) { id } }
	`, postID))

	if len(getResp.Errors) == 0 {
		t.Fatal("expected error when fetching deleted post")
	}
}

func TestGraphQL_DeletePost_Forbidden(t *testing.T) {
	handler := newTestGraphQLHandler()

	createResp := executeGraphQL(t, handler, `
		mutation {
			createPost(input: {
				authorID: "author-1"
				title: "Post"
				body: "Body"
				commentsEnabled: true
			}) { id }
		}
	`)

	var createData struct {
		CreatePost struct{ ID string `json:"id"` } `json:"createPost"`
	}
	unmarshalGraphQLData(t, createResp, &createData)

	resp := executeGraphQLAllowErrors(t, handler, fmt.Sprintf(`
		mutation { deletePost(postID: %q, authorID: "wrong-author") }
	`, createData.CreatePost.ID))

	if len(resp.Errors) == 0 {
		t.Fatal("expected error when deleting with wrong author")
	}
}

func TestGraphQL_DeleteComment(t *testing.T) {
	handler := newTestGraphQLHandler()

	createPostResp := executeGraphQL(t, handler, `
		mutation {
			createPost(input: {
				authorID: "author-1"
				title: "Post"
				body: "Body"
				commentsEnabled: true
			}) { id }
		}
	`)

	var createPostData struct {
		CreatePost struct{ ID string `json:"id"` } `json:"createPost"`
	}
	unmarshalGraphQLData(t, createPostResp, &createPostData)
	postID := createPostData.CreatePost.ID

	addResp := executeGraphQL(t, handler, fmt.Sprintf(`
		mutation {
			addComment(input: { postID: %q, authorID: "user-1", text: "To be deleted" }) { id }
		}
	`, postID))

	var addData struct {
		AddComment struct{ ID string `json:"id"` } `json:"addComment"`
	}
	unmarshalGraphQLData(t, addResp, &addData)
	commentID := addData.AddComment.ID

	deleteResp := executeGraphQL(t, handler, fmt.Sprintf(`
		mutation { deleteComment(commentID: %q, authorID: "user-1") }
	`, commentID))

	var deleteData struct {
		DeleteComment bool `json:"deleteComment"`
	}
	unmarshalGraphQLData(t, deleteResp, &deleteData)

	if !deleteData.DeleteComment {
		t.Fatal("expected deleteComment to return true")
	}

	postResp := executeGraphQL(t, handler, fmt.Sprintf(`
		query {
			post(id: %q) {
				comments(first: 10) {
					edges { node { id isDeleted text authorID } }
				}
			}
		}
	`, postID))

	var postData struct {
		Post struct {
			Comments struct {
				Edges []struct {
					Node struct {
						ID        string `json:"id"`
						IsDeleted bool   `json:"isDeleted"`
						Text      string `json:"text"`
						AuthorID  string `json:"authorID"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"comments"`
		} `json:"post"`
	}
	unmarshalGraphQLData(t, postResp, &postData)

	edges := postData.Post.Comments.Edges
	if len(edges) != 1 {
		t.Fatalf("expected deleted comment to remain as placeholder, got %d comments", len(edges))
	}
	node := edges[0].Node
	if !node.IsDeleted {
		t.Fatal("expected isDeleted to be true")
	}
	if node.Text != "[удалено]" {
		t.Fatalf("expected text to be masked, got %q", node.Text)
	}
	if node.AuthorID != "" {
		t.Fatalf("expected authorID to be empty, got %q", node.AuthorID)
	}
}

func TestGraphQL_DeleteComment_Forbidden(t *testing.T) {
	handler := newTestGraphQLHandler()

	createPostResp := executeGraphQL(t, handler, `
		mutation {
			createPost(input: {
				authorID: "author-1"
				title: "Post"
				body: "Body"
				commentsEnabled: true
			}) { id }
		}
	`)

	var createPostData struct {
		CreatePost struct{ ID string `json:"id"` } `json:"createPost"`
	}
	unmarshalGraphQLData(t, createPostResp, &createPostData)

	addResp := executeGraphQL(t, handler, fmt.Sprintf(`
		mutation {
			addComment(input: { postID: %q, authorID: "user-1", text: "Comment" }) { id }
		}
	`, createPostData.CreatePost.ID))

	var addData struct {
		AddComment struct{ ID string `json:"id"` } `json:"addComment"`
	}
	unmarshalGraphQLData(t, addResp, &addData)

	resp := executeGraphQLAllowErrors(t, handler, fmt.Sprintf(`
		mutation { deleteComment(commentID: %q, authorID: "wrong-user") }
	`, addData.AddComment.ID))

	if len(resp.Errors) == 0 {
		t.Fatal("expected error when deleting with wrong author")
	}
}

func newTestGraphQLHandler() http.Handler {
	store := memory.NewStore()

	postRepository := memory.NewPostRepository(store)
	commentRepository := memory.NewCommentRepository(store)
	commentBroker := subscriptions.NewBroker()

	postService := app.NewPostService(postRepository)
	commentService := app.NewCommentService(postRepository, commentRepository, commentBroker)

	return graph.NewHandler(postService, commentService, commentBroker)
}

func executeGraphQL(t *testing.T, handler http.Handler, query string) graphQLResponse {
	t.Helper()

	response := executeGraphQLAllowErrors(t, handler, query)

	if len(response.Errors) > 0 {
		t.Fatalf("unexpected GraphQL errors: %+v", response.Errors)
	}

	return response
}

func executeGraphQLAllowErrors(t *testing.T, handler http.Handler, query string) graphQLResponse {
	t.Helper()

	requestBody, err := json.Marshal(map[string]string{
		"query": query,
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response graphQLResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v\nbody: %s", err, recorder.Body.String())
	}

	return response
}

func unmarshalGraphQLData(t *testing.T, response graphQLResponse, target any) {
	t.Helper()

	if len(response.Data) == 0 || string(response.Data) == "null" {
		t.Fatalf("expected GraphQL data, got %s", string(response.Data))
	}

	if err := json.Unmarshal(response.Data, target); err != nil {
		t.Fatalf("failed to unmarshal GraphQL data: %v\nbody: %s", err, string(response.Data))
	}
}
