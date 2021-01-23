package cmd

import (
	"context"
	"io"
	"net/url"
)

type warmBackendGetOpts struct{}

// TargetConfig
type TargetConfig struct {
	Bucket      string
	Prefix      string
	EndpointURL *url.URL
}
type warmBackend interface {
	Put(ctx context.Context, object string, r io.Reader, length int64) error
	Get(ctx context.Context, object string, opts warmBackendGetOpts) (io.ReadCloser, error)
	Remove(ctx context.Context, object string) error
	// GetTarget() (string, string)
}
