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

import "encoding/base64"

type TransitionStorageClassGCS struct {
	Name     string
	endpoint string // custom endpoint is not supported for GCS
	Creds    string // base64 encoding of credentials.json FIXME: TBD how do we persist gcs creds file
	Bucket   string
	Prefix   string
	Region   string
}

type GCSOptions func(*TransitionStorageClassGCS) error

func GCSPrefix(prefix string) func(*TransitionStorageClassGCS) error {
	return func(gcs *TransitionStorageClassGCS) error {
		gcs.Prefix = prefix
		return nil
	}
}

func GCSRegion(region string) func(*TransitionStorageClassGCS) error {
	return func(gcs *TransitionStorageClassGCS) error {
		gcs.Region = region
		return nil
	}
}

func (gcs *TransitionStorageClassGCS) GetCredentialJSON() ([]byte, error) {
	return base64.URLEncoding.DecodeString(gcs.Creds)
}

func NewTransitionStorageClassGCS(name string, credsJSON []byte, bucket string, options ...GCSOptions) (*TransitionStorageClassGCS, error) {
	creds := base64.URLEncoding.EncodeToString(credsJSON)
	gcs := &TransitionStorageClassGCS{
		Name:     name,
		Creds:    creds,
		Bucket:   bucket,
		endpoint: "https://storage.googleapis.com",
		// Defaults
		Prefix: "",
		Region: "",
	}

	for _, option := range options {
		err := option(gcs)
		if err != nil {
			return nil, err
		}
	}

	return gcs, nil
}
