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

package rpc

import (
	"bytes"
	"encoding/gob"
	"io"
)

func gobEncode(e interface{}) ([]byte, error) {
	var buf bytes.Buffer

	if err := gob.NewEncoder(&buf).Encode(e); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func gobDecode(data []byte, e interface{}) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(e)
}

type gobReader struct {
	io.ReadCloser
}

func (r *gobReader) ReadByte() (byte, error) {
	b := []byte{0}
	_, err := r.Read(b)
	return b[0], err
}

func newGobReader(rc io.ReadCloser) *gobReader {
	return &gobReader{rc}
}
