/*
 * MinIO Cloud Storage, (C) 2018-2019 MinIO, Inc.
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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path"
	"sync"

	"github.com/minio/minio/pkg/hash"
	"github.com/minio/minio/pkg/madmin"
)

var transitionStorageClassConfigPath string = path.Join(minioConfigPrefix, "transition-storage-class-config.json")

type TransitionStorageClassConfigMgr struct {
	sync.RWMutex
	drivercache map[string]warmBackend
	S3          map[string]madmin.TransitionStorageClassS3    `json:"s3"`
	Azure       map[string]madmin.TransitionStorageClassAzure `json:"azure"`
	GCS         map[string]madmin.TransitionStorageClassGCS   `json:"gcs"`
}

func (config *TransitionStorageClassConfigMgr) isStorageClassNameInUse(scName string) bool {
	for name := range config.S3 {
		if scName == name {
			return true
		}
	}
	for name := range config.Azure {
		if scName == name {
			return true
		}
	}

	for name := range config.GCS {
		if scName == name {
			return true
		}
	}
	return false
}

func (config *TransitionStorageClassConfigMgr) Add(sc madmin.TransitionStorageClassConfig) error {
	config.Lock()
	defer config.Unlock()

	scName := sc.Name()
	// storage-class name already in use
	if config.isStorageClassNameInUse(scName) {
		return errTransitionStorageClassAlreadyExists
	}

	switch sc.Type {
	case madmin.S3:
		config.S3[scName] = *sc.S3

	case madmin.Azure:
		config.Azure[scName] = *sc.Azure

	case madmin.GCS:
		config.GCS[scName] = *sc.GCS

	default:
		return errors.New("Unsupported transition storage-class type")
	}

	return nil
}

func (config *TransitionStorageClassConfigMgr) Edit(sc madmin.TransitionStorageClassConfig) error {
	config.Lock()
	defer config.Unlock()

	scName := sc.Name()
	// no storage-class by this name exists
	if !config.isStorageClassNameInUse(scName) {
		return errTransitionStorageClassNotFound
	}

	switch sc.Type {
	case madmin.S3:
		config.S3[scName] = *sc.S3

	case madmin.Azure:
		config.Azure[scName] = *sc.Azure

	case madmin.GCS:
		config.GCS[scName] = *sc.GCS
	}
	return errInvalidArgument
}

func (config *TransitionStorageClassConfigMgr) RemoveStorageClass(name string) {
	config.Lock()
	defer config.Unlock()

	// FIXME: check if the SC is used by any of the ILM policies.

	delete(config.S3, name)
	delete(config.Azure, name)
	delete(config.GCS, name)
}

func (config *TransitionStorageClassConfigMgr) Bytes() ([]byte, error) {
	config.Lock()
	defer config.Unlock()
	return json.Marshal(config)
}

func (config *TransitionStorageClassConfigMgr) GetDriver(sc string) (d warmBackend, err error) {
	config.Lock()
	defer config.Unlock()

	d = config.drivercache[sc]
	if d != nil {
		return d, nil
	}
	for k, v := range config.S3 {
		if k == sc {
			d, err = newWarmBackendS3(v)
			break
		}
	}
	for k, v := range config.Azure {
		if k == sc {
			d, err = newWarmBackendAzure(v)
			break
		}
	}
	for k, v := range config.GCS {
		if k == sc {
			d, err = newWarmBackendGCS(v)
			break
		}
	}
	if d != nil {
		config.drivercache[sc] = d
		return d, nil
	}
	return nil, errInvalidArgument
}

func saveGlobalTransitionStorageClassConfig() error {
	b, err := json.Marshal(globalTransitionStorageClassConfigMgr)
	if err != nil {
		return err
	}
	r, err := hash.NewReader(bytes.NewReader(b), int64(len(b)), "", "", int64(len(b)), false)
	if err != nil {
		return err
	}
	_, err = globalObjectAPI.PutObject(context.Background(), minioMetaBucket, transitionStorageClassConfigPath, NewPutObjReader(r, nil, nil), ObjectOptions{})
	return err
}

func loadGlobalTransitionStorageClassConfig() error {
	var buf bytes.Buffer
	err := globalObjectAPI.GetObject(context.Background(), minioMetaBucket, transitionStorageClassConfigPath, 0, -1, &buf, "", ObjectOptions{})
	if err != nil {
		if isErrObjectNotFound(err) {
			globalTransitionStorageClassConfigMgr = &TransitionStorageClassConfigMgr{
				RWMutex:     sync.RWMutex{},
				drivercache: make(map[string]warmBackend),
				S3:          make(map[string]madmin.TransitionStorageClassS3),
				Azure:       make(map[string]madmin.TransitionStorageClassAzure),
				GCS:         make(map[string]madmin.TransitionStorageClassGCS),
			}
		}
		return err
	}
	var config TransitionStorageClassConfigMgr
	err = json.Unmarshal(buf.Bytes(), &config)
	if err != nil {
		return err
	}
	if config.drivercache == nil {
		config.drivercache = make(map[string]warmBackend)
	}
	if config.S3 == nil {
		config.S3 = make(map[string]madmin.TransitionStorageClassS3)
	}
	if config.Azure == nil {
		config.Azure = make(map[string]madmin.TransitionStorageClassAzure)
	}
	if config.GCS == nil {
		config.GCS = make(map[string]madmin.TransitionStorageClassGCS)
	}

	globalTransitionStorageClassConfigMgr = &config
	return nil
}
