package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/minio/minio/pkg/madmin"
)

type warmBackendAzure struct {
	serviceURL   azblob.ServiceURL
	Bucket       string
	Prefix       string
	StorageClass string
}

func (az *warmBackendAzure) getDest(object string) string {
	destObj := object
	if az.Prefix != "" {
		destObj = fmt.Sprintf("%s/%s", az.Prefix, object)
	}
	return destObj
}
func (az *warmBackendAzure) tier() azblob.AccessTierType {
	for _, t := range azblob.PossibleAccessTierTypeValues() {
		if strings.ToLower(az.StorageClass) == strings.ToLower(string(t)) {
			return t
		}
	}
	return azblob.AccessTierType("")
}
func (az *warmBackendAzure) Put(ctx context.Context, object string, r io.Reader, length int64) error {
	blobURL := az.serviceURL.NewContainerURL(az.Bucket).NewBlockBlobURL(az.getDest(object))
	// set tier if specified -
	if az.StorageClass != "" {
		if _, err := blobURL.SetTier(ctx, az.tier(), azblob.LeaseAccessConditions{}); err != nil {
			return err
		}
	}
	_, err := azblob.UploadStreamToBlockBlob(ctx, r, blobURL, azblob.UploadStreamToBlockBlobOptions{})
	return err
}

func (az *warmBackendAzure) Get(ctx context.Context, object string, opts warmBackendGetOpts) (r io.ReadCloser, err error) {
	if opts.startOffset < 0 {
		return nil, InvalidRange{}
	}
	blobURL := az.serviceURL.NewContainerURL(az.Bucket).NewBlobURL(az.getDest(object))
	blob, err := blobURL.Download(ctx, opts.startOffset, opts.length, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return nil, err
	}

	rc := blob.Body(azblob.RetryReaderOptions{})
	return rc, nil
}

func (az *warmBackendAzure) Remove(ctx context.Context, object string) error {
	blob := az.serviceURL.NewContainerURL(az.Bucket).NewBlobURL(az.getDest(object))
	_, err := blob.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	return err
}

func newWarmBackendAzure(conf madmin.TransitionStorageClassAzure) (*warmBackendAzure, error) {
	credential, err := azblob.NewSharedKeyCredential(conf.AccessKey, conf.SecretKey)
	if err != nil {
		if _, ok := err.(base64.CorruptInputError); ok {
			return nil, errors.New("invalid Azure credentials")
		}
		return &warmBackendAzure{}, err
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", conf.AccessKey))
	serviceURL := azblob.NewServiceURL(*u, p)
	return &warmBackendAzure{
		serviceURL:   serviceURL,
		Bucket:       conf.Bucket,
		Prefix:       conf.Prefix,
		StorageClass: conf.StorageClass,
	}, nil
}
