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
	"bytes"
	"context"
	"encoding/hex"
	"hash"
	"io"

	"github.com/minio/minio/cmd/logger"
)

// Calculates bitrot in chunks and writes the hash into the stream.
type streamingBitrotWriter struct {
	w         io.Writer
	h         hash.Hash
	shardSize int64
	// canClose     chan struct{} // Needed to avoid race explained in Close() call.
	writerClosed      chan struct{}
	allowWrites       chan struct{}
	written           int64
	closed            bool
	allowWritesClosed bool
}

func (b *streamingBitrotWriter) Write(p []byte) (int, error) {
	<-b.allowWrites
	if b.w == nil {
		return 0, errDiskNotFound
	}
	if len(p) == 0 {
		return 0, nil
	}
	b.h.Reset()
	b.h.Write(p)
	hashBytes := b.h.Sum(nil)
	n, err := b.w.Write(hashBytes)
	b.written += int64(n)
	if err != nil {
		return 0, err
	}
	n, err = b.w.Write(p)
	b.written += int64(n)
	return n, err
}

func (b *streamingBitrotWriter) WriteTo(w io.Writer) (int64, error) {
	b.w = w
	close(b.allowWrites)
	b.allowWritesClosed = true
	<-b.writerClosed
	return b.written, nil
}

func (b *streamingBitrotWriter) Read(p []byte) (int, error) {
	return 0, errAccessDenied
}

func (b *streamingBitrotWriter) Close() error {
	// err := b.iow.Close()
	// Wait for all data to be written before returning else it causes race conditions.
	// Race condition is because of io.PipeWriter implementation. i.e consider the following
	// sequent of operations:
	// 1) pipe.Write()
	// 2) pipe.Close()
	// Now pipe.Close() can return before the data is read on the other end of the pipe and written to the disk
	// Hence an immediate Read() on the file can return incorrect data.
	// <-b.canClose
	if b.closed {
		return nil
	}
	close(b.writerClosed)
	b.closed = true
	// return err
	return nil
}

// Returns streaming bitrot writer implementation.
func newStreamingBitrotWriter(disk StorageAPI, volume, filePath string, length int64, algo BitrotAlgorithm, shardSize int64) io.WriteCloser {
	// r, w := io.Pipe()
	h := algo.New()
	bw := &streamingBitrotWriter{nil, h, shardSize, make(chan struct{}), make(chan struct{}), 0, false, false}
	go func() {
		totalFileSize := int64(-1) // For compressed objects length will be unknown (represented by length=-1)
		if length != -1 {
			bitrotSumsTotalSize := ceilFrac(length, shardSize) * int64(h.Size()) // Size used for storing bitrot checksums.
			totalFileSize = bitrotSumsTotalSize + length
		}
		// err := disk.CreateFile(volume, filePath, totalFileSize, r)
		// r.CloseWithError(err)
		// close(bw.canClose)
		err := disk.CreateFile(volume, filePath, totalFileSize, bw)
		if err != nil && !bw.allowWritesClosed {
			close(bw.allowWrites)
		}
	}()
	return bw
}

// ReadAt() implementation which verifies the bitrot hash available as part of the stream.
type streamingBitrotReader struct {
	disk       StorageAPI
	rc         io.ReadCloser
	volume     string
	filePath   string
	tillOffset int64
	currOffset int64
	h          hash.Hash
	shardSize  int64
	hashBytes  []byte
}

func (b *streamingBitrotReader) Close() error {
	if b.rc == nil {
		return nil
	}
	return b.rc.Close()
}

func (b *streamingBitrotReader) ReadAt(buf []byte, offset int64) (int, error) {
	var err error
	if offset%b.shardSize != 0 {
		// Offset should always be aligned to b.shardSize
		// Can never happen unless there are programmer bugs
		return 0, errUnexpected
	}
	if b.rc == nil {
		// For the first ReadAt() call we need to open the stream for reading.
		b.currOffset = offset
		streamOffset := (offset/b.shardSize)*int64(b.h.Size()) + offset
		b.rc, err = b.disk.ReadFileStream(b.volume, b.filePath, streamOffset, b.tillOffset-streamOffset)
		if err != nil {
			return 0, err
		}
	}
	if offset != b.currOffset {
		// Can never happen unless there are programmer bugs
		return 0, errUnexpected
	}
	b.h.Reset()
	_, err = io.ReadFull(b.rc, b.hashBytes)
	if err != nil {
		return 0, err
	}
	_, err = io.ReadFull(b.rc, buf)
	if err != nil {
		return 0, err
	}
	b.h.Write(buf)

	if !bytes.Equal(b.h.Sum(nil), b.hashBytes) {
		err = HashMismatchError{hex.EncodeToString(b.hashBytes), hex.EncodeToString(b.h.Sum(nil))}
		logger.LogIf(context.Background(), err)
		return 0, err
	}
	b.currOffset += int64(len(buf))
	return len(buf), nil
}

// Returns streaming bitrot reader implementation.
func newStreamingBitrotReader(disk StorageAPI, volume, filePath string, tillOffset int64, algo BitrotAlgorithm, shardSize int64) *streamingBitrotReader {
	h := algo.New()
	return &streamingBitrotReader{
		disk,
		nil,
		volume,
		filePath,
		ceilFrac(tillOffset, shardSize)*int64(h.Size()) + tillOffset,
		0,
		h,
		shardSize,
		make([]byte, h.Size()),
	}
}
