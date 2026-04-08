package runner

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockActionCache struct {
	mock.Mock
}

func (m *mockActionCache) Fetch(ctx context.Context, cacheDir, url, ref, token string) (string, error) {
	args := m.Called(ctx, cacheDir, url, ref, token)
	return args.String(0), args.Error(1)
}

func (m *mockActionCache) GetTarArchive(ctx context.Context, cacheDir, sha, includePrefix string) (io.ReadCloser, error) {
	args := m.Called(ctx, cacheDir, sha, includePrefix)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func TestRemoteRepositoryCache_FetchFullURL(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}
	parent.On("Fetch", ctx, "old/action", "https://newgit.com/new/action", "newref", "token").
		Return("abc123", nil)

	cache := &RemoteRepositoryCache{
		Parent: parent,
		RemoteRepositories: map[string]string{
			"https://github.com/old/action@v1": "https://newgit.com/new/action@newref",
		},
	}

	sha, err := cache.Fetch(ctx, "old/action", "https://github.com/old/action", "v1", "token")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", sha)
	parent.AssertExpectations(t)
}

func TestRemoteRepositoryCache_FetchPathOnly(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}
	parent.On("Fetch", ctx, "old/action", "https://newgit.com/new/action", "newref", "token").
		Return("abc123", nil)

	cache := &RemoteRepositoryCache{
		Parent: parent,
		RemoteRepositories: map[string]string{
			"old/action@v1": "https://newgit.com/new/action@newref",
		},
	}

	sha, err := cache.Fetch(ctx, "old/action", "https://github.com/old/action", "v1", "token")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", sha)
	parent.AssertExpectations(t)
}

func TestRemoteRepositoryCache_FetchWildcardFullURL(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}
	// Wildcard: no ref in key, original ref "v2" preserved in Fetch call
	parent.On("Fetch", ctx, "old/action", "https://newgit.com/new/action", "v2", "token").
		Return("abc123", nil)

	cache := &RemoteRepositoryCache{
		Parent: parent,
		RemoteRepositories: map[string]string{
			// No @ref in key → matches any ref
			"https://github.com/old/action": "https://newgit.com/new/action",
		},
	}

	sha, err := cache.Fetch(ctx, "old/action", "https://github.com/old/action", "v2", "token")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", sha)
	parent.AssertExpectations(t)
}

func TestRemoteRepositoryCache_FetchWildcardPathOnly(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}
	// Wildcard: no ref in key, original ref "main" preserved
	parent.On("Fetch", ctx, "old/action", "https://newgit.com/new/action", "main", "token").
		Return("abc123", nil)

	cache := &RemoteRepositoryCache{
		Parent: parent,
		RemoteRepositories: map[string]string{
			// Path-only, no @ref → matches any host and any ref
			"old/action": "https://newgit.com/new/action",
		},
	}

	sha, err := cache.Fetch(ctx, "old/action", "https://github.com/old/action", "main", "token")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", sha)
	parent.AssertExpectations(t)
}

func TestRemoteRepositoryCache_FetchWildcardWithTargetRef(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}
	// Wildcard source but explicit target ref
	parent.On("Fetch", ctx, "old/action", "https://newgit.com/new/action", "stable", "token").
		Return("abc123", nil)

	cache := &RemoteRepositoryCache{
		Parent: parent,
		RemoteRepositories: map[string]string{
			// No @ref in source key, but explicit @ref in target
			"old/action": "https://newgit.com/new/action@stable",
		},
	}

	sha, err := cache.Fetch(ctx, "old/action", "https://github.com/old/action", "any-ref", "token")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", sha)
	parent.AssertExpectations(t)
}

func TestRemoteRepositoryCache_ExactMatchTakesPriorityOverWildcard(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}
	// Exact match should win over wildcard
	parent.On("Fetch", ctx, "old/action", "https://exact.com/action", "v1", "token").
		Return("abc123", nil)

	cache := &RemoteRepositoryCache{
		Parent: parent,
		RemoteRepositories: map[string]string{
			"old/action@v1": "https://exact.com/action@v1",
			"old/action":    "https://wildcard.com/action",
		},
	}

	sha, err := cache.Fetch(ctx, "old/action", "https://github.com/old/action", "v1", "token")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", sha)
	parent.AssertExpectations(t)
}

func TestRemoteRepositoryCache_FetchNoMatch(t *testing.T) {
	ctx := context.Background()
	parent := &mockActionCache{}
	parent.On("Fetch", ctx, "old/action", "https://github.com/old/action", "v1", "token").
		Return("abc123", nil)

	cache := &RemoteRepositoryCache{
		Parent: parent,
		RemoteRepositories: map[string]string{
			"other/action@v1": "https://newgit.com/new/action@newref",
		},
	}

	sha, err := cache.Fetch(ctx, "old/action", "https://github.com/old/action", "v1", "token")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", sha)
	parent.AssertExpectations(t)
}

func TestRemoteRepositoryCache_GetTarArchiveDelegatesToParent(t *testing.T) {
	ctx := context.Background()
	expected := io.NopCloser(strings.NewReader("tar data"))
	parent := &mockActionCache{}
	parent.On("GetTarArchive", ctx, "old/action", "abc123", "path/to").
		Return(expected, nil)

	cache := &RemoteRepositoryCache{
		Parent:             parent,
		RemoteRepositories: map[string]string{},
	}

	rc, err := cache.GetTarArchive(ctx, "old/action", "abc123", "path/to")
	assert.NoError(t, err)
	assert.Equal(t, expected, rc)
	parent.AssertExpectations(t)
}

func TestSplitRemoteURL(t *testing.T) {
	tests := []struct {
		input   string
		wantURL string
		wantRef string
	}{
		{"https://github.com/org/repo@v1", "https://github.com/org/repo", "v1"},
		{"https://github.com/org/repo@main", "https://github.com/org/repo", "main"},
		{"https://github.com/org/repo@abc123def456", "https://github.com/org/repo", "abc123def456"},
		{"https://github.com/org/repo", "https://github.com/org/repo", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotURL, gotRef := splitRemoteURL(tt.input)
			assert.Equal(t, tt.wantURL, gotURL)
			assert.Equal(t, tt.wantRef, gotRef)
		})
	}
}

func TestResolveRemoteRef(t *testing.T) {
	tests := []struct {
		dest        string
		originalRef string
		wantURL     string
		wantRef     string
	}{
		// Target has explicit ref → use it
		{"https://github.com/org/repo@v2", "v1", "https://github.com/org/repo", "v2"},
		// Target has no ref → use original ref
		{"https://github.com/org/repo", "v1", "https://github.com/org/repo", "v1"},
		{"https://github.com/org/repo", "main", "https://github.com/org/repo", "main"},
	}
	for _, tt := range tests {
		t.Run(tt.dest+"_from_"+tt.originalRef, func(t *testing.T) {
			gotURL, gotRef := resolveRemoteRef(tt.dest, tt.originalRef)
			assert.Equal(t, tt.wantURL, gotURL)
			assert.Equal(t, tt.wantRef, gotRef)
		})
	}
}
