/*
 * Minio Cloud Storage, (C) 2019 Minio, Inc.
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
	"github.com/ncw/directio"
)

// Calculates bitrot in chunks and writes the hash into the stream.
type streamingBitrotWriter struct {
	iow       *io.PipeWriter
	h         hash.Hash
	shardSize int64
	canClose  chan struct{} // Needed to avoid race explained in Close() call.

	// Following two fields are used only to make sure that Write(p) is called such that
	// len(p) is always the block size except the last block, i.e prevent programmer errors.
	currentBlockIdx int
	verifyTillIdx   int
}

func (b *streamingBitrotWriter) Write(p []byte) (int, error) {
	if b.currentBlockIdx < b.verifyTillIdx && int64(len(p)) != b.shardSize {
		// All blocks except last should be of the length b.shardSize
		logger.LogIf(context.Background(), errUnexpected)
		return 0, errUnexpected
	}
	if len(p) == 0 {
		return 0, nil
	}
	b.h.Reset()
	b.h.Write(p)
	hashBytes := b.h.Sum(nil)
	n, err := b.iow.Write(hashBytes)
	if n != len(hashBytes) {
		logger.LogIf(context.Background(), err)
		return 0, err
	}
	n, err = b.iow.Write(p)
	b.currentBlockIdx++
	return n, err
}

func (b *streamingBitrotWriter) Close() error {
	err := b.iow.Close()
	// Wait for all data to be written before returning else it causes race conditions.
	// Race condition is because of io.PipeWriter implementation. i.e consider the following
	// sequent of operations:
	// 1) pipe.Write()
	// 2) pipe.Close()
	// Now pipe.Close() can return before the data is read on the other end of the pipe and written to the disk
	// Hence an immediate Read() on the file can return incorrect data.
	<-b.canClose
	return err
}

// Returns streaming bitrot writer implementation.
func newStreamingBitrotWriter(disk StorageAPI, volume, filePath string, length int64, algo BitrotAlgorithm, shardSize int64) io.WriteCloser {
	r, w := io.Pipe()
	h := algo.New()
	bw := &streamingBitrotWriter{w, h, shardSize, make(chan struct{}), 0, int(length / shardSize)}
	go func() {
		bitrotSumsTotalSize := ceilFrac(length, shardSize) * int64(h.Size()) // Size used for storing bitrot checksums.
		totalFileSize := bitrotSumsTotalSize + length
		err := disk.CreateFile(volume, filePath, totalFileSize, r)
		if err != nil {
			logger.LogIf(context.Background(), err)
			r.CloseWithError(err)
		}
		close(bw.canClose)
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
	buf        []byte
}

func (b *streamingBitrotReader) Close() error {
	if b.rc == nil {
		return nil
	}
	return b.rc.Close()
}

func (b *streamingBitrotReader) BitrotReadAt(offset int64) ([]byte, error) {
	var err error
	if offset%b.shardSize != 0 {
		// Offset should always be aligned to b.shardSize
		logger.LogIf(context.Background(), errUnexpected)
		return nil, errUnexpected
	}
	if b.buf == nil {
		b.buf = directio.AlignedBlock(b.h.Size() + int(b.shardSize))
	}
	if b.rc == nil {
		// For the first ReadAt() call we need to open the stream for reading.
		b.currOffset = offset
		streamOffset := (offset/b.shardSize)*int64(b.h.Size()) + offset
		b.rc, err = b.disk.ReadFileStream(b.volume, b.filePath, streamOffset, b.tillOffset-streamOffset)
		if err != nil {
			logger.LogIf(context.Background(), err)
			return nil, err
		}
	}
	if offset != b.currOffset {
		logger.LogIf(context.Background(), errUnexpected)
		return nil, errUnexpected
	}
	b.h.Reset()
	n, err := b.rc.Read(b.buf)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}
	if err != nil {
		logger.LogIf(context.Background(), err)
		return nil, err
	}
	b.h.Write(b.buf[b.h.Size():n])

	if !bytes.Equal(b.h.Sum(nil), b.buf[:b.h.Size()]) {
		err = hashMismatchError{hex.EncodeToString(b.buf[:b.h.Size()]), hex.EncodeToString(b.h.Sum(nil))}
		logger.LogIf(context.Background(), err)
		return nil, err
	}
	b.currOffset += int64(n - b.h.Size())
	return b.buf[b.h.Size():n], nil
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
		nil,
	}
}
