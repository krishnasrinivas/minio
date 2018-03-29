/*
 * Minio Cloud Storage, (C) 2018 Minio, Inc.
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
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// CacheConfig represents cache config settings
type CacheConfig struct {
	Drives  []string `json:"drives"`
	Expiry  int      `json:"expiry"`
	Exclude []string `json:"exclude"`
}

// Parses given cacheDrivesEnv and returns a list of cache drives.
func parseCacheDrives(cacheDrivesEnv string) ([]string, error) {
	s := strings.Split(cacheDrivesEnv, ";")
	for _, d := range s {
		if !filepath.IsAbs(d) {
			return nil, fmt.Errorf("cache dir should be absolute path: %s", d)
		}
	}
	return s, nil
}

// Parses given cacheExcludesEnv and returns a list of cache exclude patterns.
func parseCacheExcludes(cacheExcludesEnv string) ([]string, error) {
	s := strings.Split(cacheExcludesEnv, ";")
	for _, e := range s {
		if len(e) == 0 {
			return nil, fmt.Errorf("cache exclude path cannot be empty")
		}
		if hasPrefix(e, slashSeparator) {
			return nil, fmt.Errorf("cache exclude pattern (%s) cannot start with / as prefix ", e)
		}
	}
	return s, nil
}

// Parses given cacheExpiryEnv and returns cache expiry in days.
func parseCacheExpiry(cacheExpiryEnv string) (int, error) {
	return strconv.Atoi(cacheExpiryEnv)
}
