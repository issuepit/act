package runner

import (
	"context"
	"fmt"
	"io"
	goURL "net/url"
	"strings"

	"github.com/nektos/act/pkg/common"
)

type RemoteRepositoryCache struct {
	Parent             ActionCache
	RemoteRepositories map[string]string
}

func (r *RemoteRepositoryCache) Fetch(ctx context.Context, cacheDir, url, ref, token string) (string, error) {
	logger := common.Logger(ctx)
	logger.Debugf("RemoteRepositoryCache fetch %s with ref %s", url, ref)
	if dest, ok := r.RemoteRepositories[fmt.Sprintf("%s@%s", url, ref)]; ok {
		newURL, newRef := splitRemoteURL(dest)
		logger.Infof("RemoteRepositoryCache matched %s with ref %s to %s@%s", url, ref, newURL, newRef)
		return r.Parent.Fetch(ctx, cacheDir, newURL, newRef, token)
	}
	if purl, err := goURL.Parse(url); err == nil {
		if dest, ok := r.RemoteRepositories[fmt.Sprintf("%s@%s", strings.TrimPrefix(purl.Path, "/"), ref)]; ok {
			newURL, newRef := splitRemoteURL(dest)
			logger.Infof("RemoteRepositoryCache matched %s with ref %s to %s@%s", url, ref, newURL, newRef)
			return r.Parent.Fetch(ctx, cacheDir, newURL, newRef, token)
		}
	}
	logger.Infof("RemoteRepositoryCache not matched %s with Ref %s", url, ref)
	return r.Parent.Fetch(ctx, cacheDir, url, ref, token)
}

func (r *RemoteRepositoryCache) GetTarArchive(ctx context.Context, cacheDir, sha, includePrefix string) (io.ReadCloser, error) {
	return r.Parent.GetTarArchive(ctx, cacheDir, sha, includePrefix)
}

// splitRemoteURL splits a remote URL of the form "https://host/org/repo@ref" into
// the URL part ("https://host/org/repo") and the ref part ("ref").
// If no "@" is found, the entire string is treated as the URL with an empty ref.
func splitRemoteURL(dest string) (string, string) {
	idx := strings.LastIndex(dest, "@")
	if idx < 0 {
		return dest, ""
	}
	return dest[:idx], dest[idx+1:]
}
