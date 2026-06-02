package graph_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/pidge31/posts-comments-service/internal/app"
	"github.com/pidge31/posts-comments-service/internal/graph"
	"github.com/pidge31/posts-comments-service/internal/storage/memory"
	"github.com/pidge31/posts-comments-service/internal/subscriptions"
)

func TestCommentAddedSubscriptionReceivesNewComments(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(newTestGraphHandler())
	defer server.Close()

	postID := createPost(t, server.URL)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/query"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Sec-WebSocket-Protocol": []string{"graphql-ws"},
	})
	if err != nil {
		t.Fatalf("connect websocket: %v", err)
	}
	defer conn.Close()

	writeGraphQLWSMessage(t, conn, map[string]any{
		"type": "connection_init",
	})
	readGraphQLWSMessageOfType(t, conn, "connection_ack")

	writeGraphQLWSMessage(t, conn, map[string]any{
		"id":   "comment-subscription",
		"type": "start",
		"payload": map[string]any{
			"query": `subscription ($postID: ID!) {
				commentAdded(postID: $postID) {
					id
					postID
					authorID
					text
				}
			}`,
			"variables": map[string]any{
				"postID": postID,
			},
		},
	})

	addComment(t, server.URL, postID)

	message := readGraphQLWSMessageOfType(t, conn, "data")
	payload := message["payload"].(map[string]any)
	data := payload["data"].(map[string]any)
	comment := data["commentAdded"].(map[string]any)

	if comment["postID"] != postID {
		t.Fatalf("postID: got %q, want %q", comment["postID"], postID)
	}

	if comment["text"] != "First comment" {
		t.Fatalf("text: got %q, want %q", comment["text"], "First comment")
	}
}

func newTestGraphHandler() http.Handler {
	store := memory.NewStore()

	postRepository := memory.NewPostRepository(store)
	commentRepository := memory.NewCommentRepository(store)
	commentBroker := subscriptions.NewBroker()

	postService := app.NewPostService(postRepository)
	commentService := app.NewCommentService(postRepository, commentRepository, commentBroker)

	return graph.NewHandler(postService, commentService, commentBroker)
}

func createPost(t *testing.T, baseURL string) string {
	t.Helper()

	response := sendGraphQLRequest(t, baseURL, `mutation {
		createPost(input: {
			authorID: "author-1"
			title: "First post"
			body: "Post body"
			commentsEnabled: true
		}) {
			id
		}
	}`)

	data := response["data"].(map[string]any)
	post := data["createPost"].(map[string]any)

	return post["id"].(string)
}

func addComment(t *testing.T, baseURL string, postID string) {
	t.Helper()

	sendGraphQLRequest(t, baseURL, `mutation ($postID: ID!) {
		addComment(input: {
			postID: $postID
			authorID: "comment-author"
			text: "First comment"
		}) {
			id
		}
	}`, map[string]any{
		"postID": postID,
	})
}

func sendGraphQLRequest(t *testing.T, baseURL string, query string, variables ...map[string]any) map[string]any {
	t.Helper()

	requestBody := map[string]any{
		"query": query,
	}

	if len(variables) > 0 {
		requestBody["variables"] = variables[0]
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/query", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if errorsValue, ok := response["errors"]; ok {
		t.Fatalf("graphql errors: %v", errorsValue)
	}

	return response
}

func writeGraphQLWSMessage(t *testing.T, conn *websocket.Conn, message map[string]any) {
	t.Helper()

	if err := conn.WriteJSON(message); err != nil {
		t.Fatalf("write websocket message: %v", err)
	}
}

func readGraphQLWSMessageOfType(t *testing.T, conn *websocket.Conn, messageType string) map[string]any {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for {
		if err := conn.SetReadDeadline(deadline); err != nil {
			t.Fatalf("set read deadline: %v", err)
		}

		var message map[string]any
		if err := conn.ReadJSON(&message); err != nil {
			t.Fatalf("read websocket message: %v", err)
		}

		if message["type"] == messageType {
			return message
		}
	}
}
