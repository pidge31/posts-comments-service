package memory

import (
	"sync"
	"time"

	"github.com/pidge31/posts-comments-service/internal/domain"
)

// commentIndexEntry is a lightweight record kept in the sorted per-parent
// index. Storing only (CreatedAt, ID) avoids duplicating full Comment data
// and keeps the sort key self-contained.
type commentIndexEntry struct {
	CreatedAt time.Time
	ID        string
}

type Store struct {
	mu sync.RWMutex

	posts    map[string]domain.Post
	comments map[string]domain.Comment

	// sorted ascending by (CreatedAt, ID) — maintained on every insert
	commentsByParent map[postParentKey][]commentIndexEntry
}

type postParentKey struct {
	postID    string
	parentID  string
	hasParent bool
}

func NewStore() *Store {
	return &Store{
		posts:            make(map[string]domain.Post),
		comments:         make(map[string]domain.Comment),
		commentsByParent: make(map[postParentKey][]commentIndexEntry),
	}
}

func makePostParentKey(postID string, parentID *string) postParentKey {
	if parentID == nil {
		return postParentKey{postID: postID}
	}

	return postParentKey{
		postID:    postID,
		parentID:  *parentID,
		hasParent: true,
	}
}
