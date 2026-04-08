package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLocalRepositoryCache_FetchExactFullURL(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}

	cache := &LocalRepositoryCache{
		Parent: parent,
		LocalRepositories: map[string]string{
			"https://github.com/org/repo@v1": "/local/path",
		},
		CacheDirCache: map[string]string{},
	}

	sha, err := cache.Fetch(ctx, "org/repo", "https://github.com/org/repo", "v1", "token")
	assert.NoError(t, err)
	assert.Equal(t, "v1", sha)
	assert.Equal(t, "/local/path", cache.CacheDirCache["org/repo@v1"])
	parent.AssertExpectations(t)
}

func TestLocalRepositoryCache_FetchExactPathOnly(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}

	cache := &LocalRepositoryCache{
		Parent: parent,
		LocalRepositories: map[string]string{
			"org/repo@v1": "/local/path",
		},
		CacheDirCache: map[string]string{},
	}

	sha, err := cache.Fetch(ctx, "org/repo", "https://github.com/org/repo", "v1", "token")
	assert.NoError(t, err)
	assert.Equal(t, "v1", sha)
	assert.Equal(t, "/local/path", cache.CacheDirCache["org/repo@v1"])
	parent.AssertExpectations(t)
}

func TestLocalRepositoryCache_FetchWildcardFullURL(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}

	cache := &LocalRepositoryCache{
		Parent: parent,
		LocalRepositories: map[string]string{
			// No @ref → matches any ref
			"https://github.com/org/repo": "/local/path",
		},
		CacheDirCache: map[string]string{},
	}

	sha, err := cache.Fetch(ctx, "org/repo", "https://github.com/org/repo", "v2", "token")
	assert.NoError(t, err)
	assert.Equal(t, "v2", sha)
	assert.Equal(t, "/local/path", cache.CacheDirCache["org/repo@v2"])
	parent.AssertExpectations(t)
}

func TestLocalRepositoryCache_FetchWildcardPathOnly(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}

	cache := &LocalRepositoryCache{
		Parent: parent,
		LocalRepositories: map[string]string{
			// Path-only, no @ref → matches any host and any ref
			"org/repo": "/local/path",
		},
		CacheDirCache: map[string]string{},
	}

	sha, err := cache.Fetch(ctx, "org/repo", "https://github.com/org/repo", "main", "token")
	assert.NoError(t, err)
	assert.Equal(t, "main", sha)
	assert.Equal(t, "/local/path", cache.CacheDirCache["org/repo@main"])
	parent.AssertExpectations(t)
}

func TestLocalRepositoryCache_FetchExactTakesPriorityOverWildcard(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}

	cache := &LocalRepositoryCache{
		Parent: parent,
		LocalRepositories: map[string]string{
			"org/repo@v1": "/exact/path",
			"org/repo":    "/wildcard/path",
		},
		CacheDirCache: map[string]string{},
	}

	sha, err := cache.Fetch(ctx, "org/repo", "https://github.com/org/repo", "v1", "token")
	assert.NoError(t, err)
	assert.Equal(t, "v1", sha)
	assert.Equal(t, "/exact/path", cache.CacheDirCache["org/repo@v1"])
	parent.AssertExpectations(t)
}

func TestLocalRepositoryCache_FetchNoMatch(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}
	parent.On("Fetch", mock.Anything, "org/repo", "https://github.com/org/repo", "v1", "token").
		Return("abc123", nil)

	cache := &LocalRepositoryCache{
		Parent: parent,
		LocalRepositories: map[string]string{
			"other/repo@v1": "/local/path",
		},
		CacheDirCache: map[string]string{},
	}

	sha, err := cache.Fetch(ctx, "org/repo", "https://github.com/org/repo", "v1", "token")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", sha)
	parent.AssertExpectations(t)
}
