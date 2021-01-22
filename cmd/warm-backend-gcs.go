package cmd

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/minio/minio/pkg/madmin"
)

type warmBackendGCS struct {
	client       *storage.Client
	Bucket       string
	Prefix       string
	StorageClass string
}

func (gcs *warmBackendGCS) getDest(object string) string {
	destObj := object
	if gcs.Prefix != "" {
		destObj = fmt.Sprintf("%s/%s", gcs.Prefix, object)
	}
	return destObj
}
func (gcs *warmBackendGCS) Put(ctx context.Context, key string, data io.Reader, length int64) error {
	object := gcs.client.Bucket(gcs.Bucket).Object(gcs.getDest(key))
	//TODO: set storage class
	w := object.NewWriter(ctx)
	if gcs.StorageClass != "" {
		w.ObjectAttrs.StorageClass = gcs.StorageClass
	}
	if _, err := io.Copy(w, data); err != nil {
		return err
	}

	return w.Close()
}

func (gcs *warmBackendGCS) Get(ctx context.Context, key string, opts warmBackendGetOpts) (r io.ReadCloser, err error) {
	// GCS storage decompresses a gzipped object by default and returns the data.
	// Refer to https://cloud.google.com/storage/docs/transcoding#decompressive_transcoding
	// Need to set `Accept-Encoding` header to `gzip` when issuing a GetObject call, to be able
	// to download the object in compressed state.
	// Calling ReadCompressed with true accomplishes that.
	object := gcs.client.Bucket(gcs.Bucket).Object(gcs.getDest(key)).ReadCompressed(true)

	r, err = object.NewRangeReader(ctx, 0, 0)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (gcs *warmBackendGCS) Remove(ctx context.Context, object string) error {
	return gcs.client.Bucket(gcs.Bucket).Object(object).Delete(ctx)
}

func newWarmBackendGCS(conf madmin.TransitionStorageClassGCS) (*warmBackendGCS, error) {
	// TODO: trim slash on prefix
	return nil, nil
}
