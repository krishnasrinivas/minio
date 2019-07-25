/*
 * MinIO Cloud Storage, (C) 2019 MinIO, Inc.
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
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// records the incoming bytes from the underlying request.Body.
type recordTrafficRequest struct {
	// wrapper for the underlying req.Body.
	reader io.Reader
	// incoming request.
	request *http.Request
}

// Records the bytes read.
func (r *recordTrafficRequest) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	globalConnStats.incInputBytes(n)
	if !strings.HasPrefix(r.request.URL.Path, "/minio") {
		globalConnStats.incS3InputBytes(n)
	}
	return n, err
}

// Records the outgoing bytes through the responseWriter.
type recordTrafficResponse struct {
	// wrapper for underlying http.ResponseWriter.
	writer  http.ResponseWriter
	request *http.Request
}

// Calls the underlying WriteHeader.
func (r *recordTrafficResponse) WriteHeader(i int) {
	r.writer.WriteHeader(i)
}

// Calls the underlying Header.
func (r *recordTrafficResponse) Header() http.Header {
	return r.writer.Header()
}

// Records the output bytes
func (r *recordTrafficResponse) Write(p []byte) (n int, err error) {
	n, err = r.writer.Write(p)
	globalConnStats.incOutputBytes(n)
	// Check if it is s3 request
	if !strings.HasPrefix(r.request.URL.Path, "/minio") {
		globalConnStats.incS3OutputBytes(n)
	}
	return n, err
}

// Calls the underlying Flush.
func (r *recordTrafficResponse) Flush() {
	r.writer.(http.Flusher).Flush()
}

// RecordTraffic - a wrapper for the underlying HandlerFunc to record the traffic.
func RecordTraffic(f http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	reqRecorder := &recordTrafficRequest{reader: r.Body, request: r}
	r.Body = ioutil.NopCloser(reqRecorder)

	respBodyRecorder := &recordTrafficResponse{writer: w, request: r}
	f(respBodyRecorder, r)
}
