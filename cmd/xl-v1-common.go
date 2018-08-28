/*
 * Minio Cloud Storage, (C) 2016, 2017 Minio, Inc.
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
 */

package cmd

import (
	"context"
	"path"
)

// getLoadBalancedDisks - fetches load balanced (sufficiently randomized) disk slice.
func (xl xlObjects) getLoadBalancedDisks() (disks []StorageAPI) {
	// Based on the random shuffling return back randomized disks.
	for _, i := range hashOrder(UTCNow().String(), len(xl.getDisks())) {
		disks = append(disks, xl.getDisks()[i-1])
	}
	return disks
}

// This function does the following check, suppose
// object is "a/b/c/d", stat makes sure that objects ""a/b/c""
// "a/b" and "a" do not exist.
func (xl xlObjects) parentDirIsObject(ctx context.Context, bucket, parent string) bool {
	var isParentDirObject func(string) bool
	isParentDirObject = func(p string) bool {
		if p == "." || p == "/" {
			return false
		}
		if xl.isObject(bucket, p) {
			// If there is already a file at prefix "p", return true.
			return true
		}
		// Check if there is a file as one of the parent paths.
		return isParentDirObject(path.Dir(p))
	}
	return isParentDirObject(parent)
}

func (xl xlObjects) isJSONFile(bucket, prefix, jsonFile string) (ok bool) {
	for _, disk := range xl.getLoadBalancedDisks() {
		if disk == nil {
			continue
		}
		// Check if 'prefix' is an object on this 'disk', else continue the check the next disk
		_, err := disk.StatFile(bucket, path.Join(prefix, jsonFile))
		if err == nil {
			return true
		}
	} // Exhausted all disks - return false.
	return false
}

// isObject - returns `true` if the prefix is an object i.e if
// `xl.json` or `versioning.json` exists at the leaf, false otherwise.
func (xl xlObjects) isObject(bucket, prefix string) (ok bool) {
	return xl.isJSONFile(bucket, prefix, xlMetaJSONFile) || xl.isJSONFile(bucket, prefix, xlVersioningJSONFile)
}

// isObjectDir returns if the specified path represents an empty directory.
func (xl xlObjects) isObjectDir(bucket, prefix string) (ok bool) {
	for _, disk := range xl.getLoadBalancedDisks() {
		if disk == nil {
			continue
		}
		// Check if 'prefix' is an object on this 'disk', else continue the check the next disk
		ctnts, err := disk.ListDir(bucket, prefix, 1)
		if err == nil {
			if len(ctnts) == 0 {
				return true
			}
			return false
		}
	} // Exhausted all disks - return false.
	return false
}

// Calculate the space occupied by an object in a single disk
func (xl xlObjects) sizeOnDisk(fileSize int64, blockSize int64, dataBlocks int) int64 {
	numBlocks := fileSize / blockSize
	chunkSize := ceilFrac(blockSize, int64(dataBlocks))
	sizeInDisk := numBlocks * chunkSize
	remaining := fileSize % blockSize
	if remaining > 0 {
		sizeInDisk += ceilFrac(remaining, int64(dataBlocks))
	}

	return sizeInDisk
}
