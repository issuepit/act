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
