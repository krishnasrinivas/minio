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

type TransitionStorageClassAzure struct {
	Name      string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Prefix    string
}

type AzureOptions func(*TransitionStorageClassAzure) error

func AzurePrefix(prefix string) func(az *TransitionStorageClassAzure) error {
	return func(az *TransitionStorageClassAzure) error {
		az.Prefix = prefix
		return nil
	}
}

func AzureEndpoint(endpoint string) func(az *TransitionStorageClassAzure) error {
	return func(az *TransitionStorageClassAzure) error {
		az.Endpoint = endpoint
		return nil
	}
}

func NewTransitionStorageClassAzure(name, accessKey, secretKey, bucket string, options ...AzureOptions) (*TransitionStorageClassAzure, error) {
	az := &TransitionStorageClassAzure{
		Name:      name,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,
		// Defaults
		Endpoint: "http://blob.core.windows.net",
		Prefix:   "",
	}

	for _, option := range options {
		err := option(az)
		if err != nil {
			return nil, err
		}
	}

	return az, nil
}
