package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
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
			return azureToObjectError(err, az.Bucket, object)
		}
	}
	_, err := azblob.UploadStreamToBlockBlob(ctx, r, blobURL, azblob.UploadStreamToBlockBlobOptions{})
	return azureToObjectError(err, az.Bucket, object)
}

func (az *warmBackendAzure) Get(ctx context.Context, object string, opts warmBackendGetOpts) (r io.ReadCloser, err error) {
	if opts.startOffset < 0 {
		return nil, InvalidRange{}
	}
	blobURL := az.serviceURL.NewContainerURL(az.Bucket).NewBlobURL(az.getDest(object))
	blob, err := blobURL.Download(ctx, opts.startOffset, opts.length, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return nil, azureToObjectError(err, az.Bucket, object)
	}

	rc := blob.Body(azblob.RetryReaderOptions{})
	return rc, nil
}

func (az *warmBackendAzure) Remove(ctx context.Context, object string) error {
	blob := az.serviceURL.NewContainerURL(az.Bucket).NewBlobURL(az.getDest(object))
	_, err := blob.Delete(ctx, azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	return azureToObjectError(err, az.Bucket, object)
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
		Prefix:       strings.TrimSuffix(conf.Prefix, slashSeparator),
		StorageClass: conf.StorageClass,
	}, nil
}

// Convert azure errors to minio object layer errors.
func azureToObjectError(err error, params ...string) error {
	if err == nil {
		return nil
	}

	bucket := ""
	object := ""
	if len(params) >= 1 {
		bucket = params[0]
	}
	if len(params) == 2 {
		object = params[1]
	}

	azureErr, ok := err.(azblob.StorageError)
	if !ok {
		// We don't interpret non Azure errors. As azure errors will
		// have StatusCode to help to convert to object errors.
		return err
	}

	serviceCode := string(azureErr.ServiceCode())
	statusCode := azureErr.Response().StatusCode

	return azureCodesToObjectError(err, serviceCode, statusCode, bucket, object)
}

func azureCodesToObjectError(err error, serviceCode string, statusCode int, bucket string, object string) error {
	switch serviceCode {
	case "ContainerNotFound", "ContainerBeingDeleted":
		err = BucketNotFound{Bucket: bucket}
	case "ContainerAlreadyExists":
		err = BucketExists{Bucket: bucket}
	case "InvalidResourceName":
		err = BucketNameInvalid{Bucket: bucket}
	case "RequestBodyTooLarge":
		err = PartTooBig{}
	case "InvalidMetadata":
		err = UnsupportedMetadata{}
	case "BlobAccessTierNotSupportedForAccountType":
		err = NotImplemented{}
	case "OutOfRangeInput":
		err = ObjectNameInvalid{
			Bucket: bucket,
			Object: object,
		}
	default:
		switch statusCode {
		case http.StatusNotFound:
			if object != "" {
				err = ObjectNotFound{
					Bucket: bucket,
					Object: object,
				}
			} else {
				err = BucketNotFound{Bucket: bucket}
			}
		case http.StatusBadRequest:
			err = BucketNameInvalid{Bucket: bucket}
		}
	}
	return err
}
