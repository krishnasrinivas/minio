/*
 * MinIO Cloud Storage, (C) 2021 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package madmin

import (
	"errors"
	"log"
)

type StorageClassType int

const (
	unsupported StorageClassType = iota
	S3
	Azure
	GCS
)

func (st StorageClassType) String() string {
	switch st {
	case S3:
		return "s3"
	case Azure:
		return "azure"
	case GCS:
		return "gcs"
	}
	return "unsupported"
}

func NewStorageClassType(scType string) (StorageClassType, error) {
	switch scType {
	case S3.String():
		return S3, nil
	case Azure.String():
		return Azure, nil
	case GCS.String():
		return GCS, nil
	}

	return unsupported, errors.New("Unsupported storage class type")
}

type TransitionStorageClassConfig struct {
	Type  StorageClassType
	S3    *TransitionStorageClassS3
	Azure *TransitionStorageClassAzure
	GCS   *TransitionStorageClassGCS
}

func (cfg *TransitionStorageClassConfig) Endpoint() string {
	switch cfg.Type {
	case S3:
		return cfg.S3.Endpoint
	case Azure:
		return cfg.Azure.Endpoint
	case GCS:
		return cfg.Azure.Endpoint
	}
	log.Printf("unexpected transition storage-class type %s", cfg.Type)
	return ""
}

func (cfg *TransitionStorageClassConfig) Bucket() string {
	switch cfg.Type {
	case S3:
		return cfg.S3.Bucket
	case Azure:
		return cfg.Azure.Bucket
	case GCS:
		return cfg.Azure.Bucket
	}
	log.Printf("unexpected transition storage-class type %s", cfg.Type)
	return ""
}

func (cfg *TransitionStorageClassConfig) Prefix() string {
	switch cfg.Type {
	case S3:
		return cfg.S3.Prefix
	case Azure:
		return cfg.Azure.Prefix
	case GCS:
		return cfg.GCS.Prefix
	}
	log.Printf("unexpected transition storage-class type %s", cfg.Type)
	return ""
}

func (cfg *TransitionStorageClassConfig) Region() string {
	switch cfg.Type {
	case S3:
		return cfg.S3.Region
	case Azure:
		return cfg.Azure.Region
	case GCS:
		return cfg.GCS.Region
	}
	log.Printf("unexpected transition storage-class type %s", cfg.Type)
	return ""
}

func (cfg *TransitionStorageClassConfig) Name() string {
	switch cfg.Type {
	case S3:
		return cfg.S3.Name
	case Azure:
		return cfg.Azure.Name
	case GCS:
		return cfg.GCS.Name
	}
	log.Printf("unexpected transition storage-class type %s", cfg.Type)
	return ""
}
