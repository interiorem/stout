package porto

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sync"

	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/noxiouz/stout/pkg/log"
	"github.com/uber-go/zap"
)

type asyncSpoolResult struct {
	path string
	err  error
}

type BlobRepository interface {
	Get(ctx context.Context, repository distribution.Repository, dgst digest.Digest) (string, error)
}

type BlobRepositoryConfig struct {
	SpoolPath string `json:"spool"`
}

type blobRepo struct {
	mu sync.Mutex
	BlobRepositoryConfig
	inProgress map[digest.Digest][]chan asyncSpoolResult
}

func NewBlobRepository(ctx context.Context, cfg BlobRepositoryConfig) (BlobRepository, error) {
	if cfg.SpoolPath == "" {
		return nil, fmt.Errorf("spool is not configuried")
	}

	if err := os.MkdirAll(cfg.SpoolPath, 0777); err != nil {
		return nil, err
	}

	s := &blobRepo{
		BlobRepositoryConfig: cfg,
		inProgress:           make(map[digest.Digest][]chan asyncSpoolResult),
	}

	return s, nil
}

func (r *blobRepo) Get(ctx context.Context, repository distribution.Repository, dgst digest.Digest) (string, error) {
	log.G(ctx).Info("get a blob from Repository", zap.Stringer("digest", dgst))
	path := filepath.Join(r.BlobRepositoryConfig.SpoolPath, dgst.String())
	_, err := os.Lstat(path)
	if err == nil {
		log.G(ctx).Info("the blob has already downloaded", zap.Stringer("digest", dgst))
		return path, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	return r.download(ctx, repository, dgst)
}

func (r *blobRepo) download(ctx context.Context, repository distribution.Repository, dgst digest.Digest) (string, error) {
	ch := make(chan asyncSpoolResult, 1)
	r.mu.Lock()
	downloading, ok := r.inProgress[dgst]
	r.inProgress[dgst] = append(downloading, ch)
	if !ok {
		go func() {
			path, err := r.fetch(ctx, repository, dgst)
			res := asyncSpoolResult{path: path, err: err}
			r.mu.Lock()
			for _, ch := range r.inProgress[dgst] {
				ch <- res
			}
			r.mu.Unlock()
		}()
	}
	r.mu.Unlock()

	log.G(ctx).Info("the blob downloading is in progress. Waiting", zap.Stringer("digest", dgst))
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-ch:
		return res.path, res.err
	}
}

// fetch downloads the blob to a tempfile, renames it to the expected name
func (r *blobRepo) fetch(ctx context.Context, repository distribution.Repository, dgst digest.Digest) (path string, err error) {
	log.G(ctx).Info("fetch the blob", zap.Stringer("digest", dgst))
	tempFilePath := filepath.Join(r.SpoolPath, fmt.Sprintf("%s-%d", dgst.String(), rand.Int63()))
	f, err := os.Create(tempFilePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	defer os.Remove(tempFilePath)

	blob, err := repository.Blobs(ctx).Open(ctx, dgst)
	if err != nil {
		return "", err
	}
	defer blob.Close()

	if _, err = io.Copy(f, blob); err != nil {
		return "", err
	}
	f.Close()
	blob.Close()

	resultFilePath := filepath.Join(r.SpoolPath, dgst.String())
	if err = os.Rename(tempFilePath, resultFilePath); err != nil {
		return "", err
	}

	return resultFilePath, nil
}
