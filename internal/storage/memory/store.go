package memory

import (
	"sync"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

type Store struct {
	mu sync.RWMutex

	posts    map[string]domain.Post
	comments map[string]domain.Comment

	commentIDsByParent map[postParentKey][]string
}

type postParentKey struct {
	postID    string
	parentID  string
	hasParent bool
}

func NewStore() *Store {
	return &Store{
		posts:              make(map[string]domain.Post),
		comments:           make(map[string]domain.Comment),
		commentIDsByParent: make(map[postParentKey][]string),
	}
}

func makePostParentKey(postID string, parentID *string) postParentKey {
	if parentID == nil {
		return postParentKey{
			postID: postID,
		}
	}

	return postParentKey{
		postID:    postID,
		parentID:  *parentID,
		hasParent: true,
	}
}
