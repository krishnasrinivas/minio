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
	"encoding/base64"
	"fmt"
	"log"
	"testing"
)

func ExampleTransitionStorageClassS3() {
	simpleS3SC, err := NewTransitionStorageClassS3("simple-s3", "accessKey", "secretKey", "testbucket")
	if err != nil {
		log.Fatalln(err, "Failed to create s3 backed storage-class")
	}
	fmt.Println(simpleS3SC)

	fullyCustomS3SC, err := NewTransitionStorageClassS3("custom-s3", "accessKey", "secretKey", "testbucket",
		S3Endpoint("https://s3.amazonaws.com"), S3Prefix("testprefix"), S3Region("us-west-1"), S3StorageClass("S3_IA"))
	if err != nil {
		log.Fatalln(err, "Failed to create s3 storage-class")
	}
	fmt.Println(fullyCustomS3SC)
}

func ExampleTransitionStorageClassAzure() {
	simpleAzSC, err := NewTransitionStorageClassAzure("simple-az", "accessKey", "secretKey", "testbucket")
	if err != nil {
		log.Fatalln(err, "Failed to create azure backed storage-class")
	}
	fmt.Println(simpleAzSC)

	fullyCustomAzSC, err := NewTransitionStorageClassAzure("custom-az", "accessKey", "secretKey", "testbucket", AzureEndpoint("http://blob.core.windows.net"), AzurePrefix("testprefix"))
	if err != nil {
		log.Fatalln(err, "Failed to create azure backed storage-class")
	}
	fmt.Println(fullyCustomAzSC)
}

func ExampleTransitionStorageClassGCS() {
	credsJSON := []byte("credentials json content goes here")
	simpleGCSSC, err := NewTransitionStorageClassGCS("simple-gcs", credsJSON, "testbucket")
	if err != nil {
		log.Fatalln(err, "Failed to create GCS backed storage-class")
	}
	fmt.Println(simpleGCSSC)

	fullyCustomGCSSC, err := NewTransitionStorageClassGCS("custom-gcs", credsJSON, "testbucket", GCSEndpoint("https://storage.googleapis.com/storage/v1/"), GCSPrefix("testprefix"))
	if err != nil {
		log.Fatalln(err, "Failed to create GCS backed storage-class")
	}
	fmt.Println(fullyCustomGCSSC)
}

// TestS3StorageClass tests S3Options helpers
func TestS3StorageClass(t *testing.T) {
	scName := "test-s3"
	endpoint := "https://mys3.com"
	accessKey, secretKey := "accessKey", "secretKey"
	bucket, prefix := "testbucket", "testprefix"
	region := "us-west-1"
	storageClass := "S3_IA"
	want := &TransitionStorageClassS3{
		Name:      scName,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,

		// custom values
		Endpoint:     endpoint,
		Prefix:       prefix,
		Region:       region,
		StorageClass: storageClass,
	}
	options := []S3Options{
		S3Endpoint(endpoint),
		S3Prefix(prefix),
		S3Region(region),
		S3StorageClass(storageClass),
	}
	got, err := NewTransitionStorageClassS3(scName, accessKey, secretKey, bucket, options...)
	if err != nil {
		t.Fatalf("Failed to create a custom s3 transition storage class %s", err)
	}

	if *got != *want {
		t.Fatalf("got != want, got = %v want = %v", got, want)
	}
}

// TestAzStorageClass tests AzureOptions helpers
func TestAzStorageClass(t *testing.T) {
	scName := "test-az"
	endpoint := "https://myazure.com"
	accessKey, secretKey := "accessKey", "secretKey"
	bucket, prefix := "testbucket", "testprefix"
	region := "us-east-1"
	want := &TransitionStorageClassAzure{
		Name:      scName,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucket,

		// custom values
		Endpoint: endpoint,
		Prefix:   prefix,
		Region:   region,
	}
	options := []AzureOptions{
		AzureEndpoint(endpoint),
		AzurePrefix(prefix),
		AzureRegion(region),
	}
	got, err := NewTransitionStorageClassAzure(scName, accessKey, secretKey, bucket, options...)
	if err != nil {
		t.Fatalf("Failed to create a custom s3 transition storage class %s", err)
	}

	if *got != *want {
		t.Fatalf("got != want, got = %v want = %v", got, want)
	}
}

// TestGCSStorageClass tests GCSOptions helpers
func TestGCSStorageClass(t *testing.T) {
	scName := "test-gcs"
	endpoint := "https://mygcs.com"
	credsJSON := []byte("test-creds-json")
	encodedCreds := base64.URLEncoding.EncodeToString(credsJSON)
	bucket, prefix := "testbucket", "testprefix"
	region := "us-west-2"
	want := &TransitionStorageClassGCS{
		Name:   scName,
		Bucket: bucket,
		Creds:  encodedCreds,

		// custom values
		Endpoint: endpoint,
		Prefix:   prefix,
		Region:   region,
	}
	options := []GCSOptions{
		GCSEndpoint(endpoint),
		GCSRegion(region),
		GCSPrefix(prefix),
	}
	got, err := NewTransitionStorageClassGCS(scName, credsJSON, bucket, options...)
	if err != nil {
		t.Fatalf("Failed to create a custom s3 transition storage class %s", err)
	}

	if *got != *want {
		t.Fatalf("got != want, got = %v want = %v", got, want)
	}
}
