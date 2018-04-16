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
	"bytes"
	"context"
	"io"
	"math/rand"
	"testing"

	crand "crypto/rand"

	humanize "github.com/dustin/go-humanize"
)

func (d badDisk) ReadFile(volume string, path string, offset int64, buf []byte, verifier *BitrotVerifier) (n int64, err error) {
	return 0, errFaultyDisk
}

var erasureReadFileTests = []struct {
	dataBlocks                   int
	onDisks, offDisks            int
	blocksize, data              int64
	offset                       int64
	length                       int64
	algorithm                    BitrotAlgorithm
	shouldFail, shouldFailQuorum bool
}{
	{dataBlocks: 2, onDisks: 4, offDisks: 0, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: BLAKE2b512, shouldFail: false, shouldFailQuorum: false},                                                         // 0
	{dataBlocks: 3, onDisks: 6, offDisks: 0, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: SHA256, shouldFail: false, shouldFailQuorum: false},                                                             // 1
	{dataBlocks: 4, onDisks: 8, offDisks: 0, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                             // 2
	{dataBlocks: 5, onDisks: 10, offDisks: 0, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 1, length: oneMiByte - 1, algorithm: BLAKE2b512, shouldFail: false, shouldFailQuorum: false},                                                    // 3
	{dataBlocks: 6, onDisks: 12, offDisks: 0, blocksize: int64(oneMiByte), data: oneMiByte, offset: oneMiByte, length: 0, algorithm: BLAKE2b512, shouldFail: false, shouldFailQuorum: false},                                                          // 4
	{dataBlocks: 7, onDisks: 14, offDisks: 0, blocksize: int64(oneMiByte), data: oneMiByte, offset: 3, length: 1024, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                                   // 5
	{dataBlocks: 8, onDisks: 16, offDisks: 0, blocksize: int64(oneMiByte), data: oneMiByte, offset: 4, length: 8 * 1024, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                               // 6
	{dataBlocks: 7, onDisks: 14, offDisks: 7, blocksize: int64(blockSizeV1), data: oneMiByte, offset: oneMiByte, length: 1, algorithm: DefaultBitrotAlgorithm, shouldFail: true, shouldFailQuorum: false},                                             // 7
	{dataBlocks: 6, onDisks: 12, offDisks: 6, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                            // 8
	{dataBlocks: 5, onDisks: 10, offDisks: 5, blocksize: int64(oneMiByte), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: BLAKE2b512, shouldFail: false, shouldFailQuorum: false},                                                          // 9
	{dataBlocks: 4, onDisks: 8, offDisks: 4, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: SHA256, shouldFail: false, shouldFailQuorum: false},                                                             // 10
	{dataBlocks: 3, onDisks: 6, offDisks: 3, blocksize: int64(oneMiByte), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                               // 11
	{dataBlocks: 2, onDisks: 4, offDisks: 2, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                             // 12
	{dataBlocks: 2, onDisks: 4, offDisks: 1, blocksize: int64(oneMiByte), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                               // 13
	{dataBlocks: 3, onDisks: 6, offDisks: 2, blocksize: int64(oneMiByte), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                               // 14
	{dataBlocks: 4, onDisks: 8, offDisks: 3, blocksize: int64(2 * oneMiByte), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                           // 15
	{dataBlocks: 5, onDisks: 10, offDisks: 6, blocksize: int64(oneMiByte), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: true},                                               // 16
	{dataBlocks: 5, onDisks: 10, offDisks: 2, blocksize: int64(blockSizeV1), data: 2 * oneMiByte, offset: oneMiByte, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                // 17
	{dataBlocks: 5, onDisks: 10, offDisks: 1, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: BLAKE2b512, shouldFail: false, shouldFailQuorum: false},                                                        // 18
	{dataBlocks: 6, onDisks: 12, offDisks: 3, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: SHA256, shouldFail: false, shouldFailQuorum: false},                                                            // 19
	{dataBlocks: 6, onDisks: 12, offDisks: 7, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: true},                                             // 20
	{dataBlocks: 8, onDisks: 16, offDisks: 8, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                            // 21
	{dataBlocks: 8, onDisks: 16, offDisks: 9, blocksize: int64(oneMiByte), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: true},                                               // 22
	{dataBlocks: 8, onDisks: 16, offDisks: 7, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                            // 23
	{dataBlocks: 2, onDisks: 4, offDisks: 1, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                             // 24
	{dataBlocks: 2, onDisks: 4, offDisks: 0, blocksize: int64(blockSizeV1), data: oneMiByte, offset: 0, length: oneMiByte, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},                                             // 25
	{dataBlocks: 2, onDisks: 4, offDisks: 0, blocksize: int64(blockSizeV1), data: int64(blockSizeV1) + 1, offset: 0, length: int64(blockSizeV1) + 1, algorithm: BLAKE2b512, shouldFail: false, shouldFailQuorum: false},                               // 27
	{dataBlocks: 2, onDisks: 4, offDisks: 0, blocksize: int64(blockSizeV1), data: int64(2 * blockSizeV1), offset: 12, length: int64(blockSizeV1) + 17, algorithm: BLAKE2b512, shouldFail: false, shouldFailQuorum: false},                             // 28
	{dataBlocks: 3, onDisks: 6, offDisks: 0, blocksize: int64(blockSizeV1), data: int64(2 * blockSizeV1), offset: 1023, length: int64(blockSizeV1) + 1024, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},             // 29
	{dataBlocks: 4, onDisks: 8, offDisks: 0, blocksize: int64(blockSizeV1), data: int64(2 * blockSizeV1), offset: 11, length: int64(blockSizeV1) + 2*1024, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},             // 30
	{dataBlocks: 6, onDisks: 12, offDisks: 0, blocksize: int64(blockSizeV1), data: int64(2 * blockSizeV1), offset: 512, length: int64(blockSizeV1) + 8*1024, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},           // 31
	{dataBlocks: 8, onDisks: 16, offDisks: 0, blocksize: int64(blockSizeV1), data: int64(2 * blockSizeV1), offset: int64(blockSizeV1), length: int64(blockSizeV1) - 1, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false}, // 32
	{dataBlocks: 2, onDisks: 4, offDisks: 0, blocksize: int64(blockSizeV1), data: int64(oneMiByte), offset: -1, length: 3, algorithm: DefaultBitrotAlgorithm, shouldFail: true, shouldFailQuorum: false},                                              // 33
	{dataBlocks: 2, onDisks: 4, offDisks: 0, blocksize: int64(blockSizeV1), data: int64(oneMiByte), offset: 1024, length: -1, algorithm: DefaultBitrotAlgorithm, shouldFail: true, shouldFailQuorum: false},                                           // 34
	{dataBlocks: 4, onDisks: 6, offDisks: 0, blocksize: int64(blockSizeV1), data: int64(blockSizeV1), offset: 0, length: int64(blockSizeV1), algorithm: BLAKE2b512, shouldFail: false, shouldFailQuorum: false},                                       // 35
	{dataBlocks: 4, onDisks: 6, offDisks: 1, blocksize: int64(blockSizeV1), data: int64(2 * blockSizeV1), offset: 12, length: int64(blockSizeV1) + 17, algorithm: BLAKE2b512, shouldFail: false, shouldFailQuorum: false},                             // 36
	{dataBlocks: 4, onDisks: 6, offDisks: 3, blocksize: int64(blockSizeV1), data: int64(2 * blockSizeV1), offset: 1023, length: int64(blockSizeV1) + 1024, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: true},              // 37
	{dataBlocks: 8, onDisks: 12, offDisks: 4, blocksize: int64(blockSizeV1), data: int64(2 * blockSizeV1), offset: 11, length: int64(blockSizeV1) + 2*1024, algorithm: DefaultBitrotAlgorithm, shouldFail: false, shouldFailQuorum: false},            // 38
}

func TestErasureReadFile(t *testing.T) {
	for i, test := range erasureReadFileTests {
		setup, err := newErasureTestSetup(test.dataBlocks, test.onDisks-test.dataBlocks, test.blocksize)
		if err != nil {
			t.Fatalf("Test %d: failed to create test setup: %v", i, err)
		}
		storage, err := NewErasureStorage(context.Background(), setup.disks, test.dataBlocks, test.onDisks-test.dataBlocks, test.blocksize)
		if err != nil {
			setup.Remove()
			t.Fatalf("Test %d: failed to create ErasureStorage: %v", i, err)
		}

		data := make([]byte, test.data)
		if _, err = io.ReadFull(crand.Reader, data); err != nil {
			setup.Remove()
			t.Fatalf("Test %d: failed to generate random test data: %v", i, err)
		}

		writeAlgorithm := test.algorithm
		if !test.algorithm.Available() {
			writeAlgorithm = DefaultBitrotAlgorithm
		}
		buffer := make([]byte, test.blocksize)
		writers := make([]*bitrotWriter, len(storage.disks))
		for i, disk := range storage.disks {
			if disk == nil {
				continue
			}
			writers[i] = NewBitrotWriter(disk, "testbucket", "object", writeAlgorithm, DefaultBitrotBlockSize)
		}
		n, err := storage.CreateFile(context.Background(), bytes.NewReader(data[:]), writers, buffer)
		if err != nil {
			setup.Remove()
			t.Fatalf("Test %d: failed to create erasure test file: %v", i, err)
		}
		if n != test.data {
			setup.Remove()
			t.Fatalf("Test %d: failed to create erasure test file", i)
		}

		// Get the checksums of the current part.
		bitrotReaders := make([]*bitrotReader, len(storage.disks))
		for index, disk := range storage.disks {
			if disk == OfflineDisk {
				continue
			}
			bitrotReaders[index] = NewBitrotReader(disk, "testbucket", "object", writeAlgorithm, DefaultBitrotBlockSize, writers[index].Close(), getErasureFileSize(int(test.blocksize), n, storage.dataBlocks))
		}

		writer := bytes.NewBuffer(nil)
		err = storage.ReadFile(context.Background(), writer, bitrotReaders, test.offset, test.length, test.data)
		if err != nil && !test.shouldFail {
			t.Errorf("Test %d: should pass but failed with: %v", i, err)
		}
		if err == nil && test.shouldFail {
			t.Errorf("Test %d: should fail but it passed", i)
		}
		if err == nil {
			if content := writer.Bytes(); !bytes.Equal(content, data[test.offset:test.offset+test.length]) {
				t.Errorf("Test %d: read retruns wrong file content", i)
			}
		}
		if err == nil && !test.shouldFail {
			bitrotReaders = make([]*bitrotReader, len(storage.disks))
			for index, disk := range storage.disks {
				if disk == OfflineDisk {
					continue
				}
				bitrotReaders[index] = NewBitrotReader(disk, "testbucket", "object", writeAlgorithm, DefaultBitrotBlockSize, writers[index].Close(), getErasureFileSize(int(test.blocksize), n, storage.dataBlocks))
			}
			for j := range storage.disks[:test.offDisks] {
				bitrotReaders[j].disk = badDisk{nil}
			}
			if test.offDisks > 0 {
				bitrotReaders[0] = nil
			}
			writer.Reset()
			err = storage.ReadFile(context.Background(), writer, bitrotReaders, test.offset, test.length, test.data)
			if err != nil && !test.shouldFailQuorum {
				t.Errorf("Test %d: should pass but failed with: %v", i, err)
			}
			if err == nil && test.shouldFailQuorum {
				t.Errorf("Test %d: should fail but it passed", i)
			}
			if !test.shouldFailQuorum {
				if content := writer.Bytes(); !bytes.Equal(content, data[test.offset:test.offset+test.length]) {
					t.Errorf("Test %d: read retruns wrong file content", i)
				}
			}
		}
		setup.Remove()
	}
}

// Test erasureReadFile with random offset and lengths.
// This test is t.Skip()ed as it a long time to run, hence should be run
// explicitly after commenting out t.Skip()
func TestErasureReadFileRandomOffsetLength(t *testing.T) {
	// Comment the following line to run this test.
	t.SkipNow()
	// Initialize environment needed for the test.
	dataBlocks := 7
	parityBlocks := 7
	blockSize := int64(1 * humanize.MiByte)
	setup, err := newErasureTestSetup(dataBlocks, parityBlocks, blockSize)
	if err != nil {
		t.Error(err)
		return
	}
	defer setup.Remove()

	storage, err := NewErasureStorage(context.Background(), setup.disks, dataBlocks, parityBlocks, blockSize)
	if err != nil {
		t.Fatalf("failed to create ErasureStorage: %v", err)
	}
	// Prepare a slice of 5MiB with random data.
	data := make([]byte, 5*humanize.MiByte)
	length := int64(len(data))
	_, err = rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}

	writers := make([]*bitrotWriter, len(storage.disks))
	for i, disk := range storage.disks {
		if disk == nil {
			continue
		}
		writers[i] = NewBitrotWriter(disk, "testbucket", "object", DefaultBitrotAlgorithm, DefaultBitrotBlockSize)
	}

	// 10000 iterations with random offsets and lengths.
	iterations := 10000

	// Create a test file to read from.
	buffer := make([]byte, blockSize, 2*blockSize)
	n, err := storage.CreateFile(context.Background(), bytes.NewReader(data), writers, buffer)
	if err != nil {
		t.Fatal(err)
	}
	if n != length {
		t.Errorf("erasureCreateFile returned %d, expected %d", n, length)
	}

	// To generate random offset/length.
	r := rand.New(rand.NewSource(UTCNow().UnixNano()))

	buf := &bytes.Buffer{}

	// Verify erasureReadFile() for random offsets and lengths.
	for i := 0; i < iterations; i++ {
		offset := r.Int63n(length)
		readLen := r.Int63n(length - offset)

		expected := data[offset : offset+readLen]

		// Get the checksums of the current part.
		bitrotReaders := make([]*bitrotReader, len(storage.disks))
		for index, disk := range storage.disks {
			if disk == OfflineDisk {
				continue
			}
			bitrotReaders[index] = NewBitrotReader(disk, "testbucket", "object", DefaultBitrotAlgorithm, DefaultBitrotBlockSize, writers[index].Close(), getErasureFileSize(int(blockSize), n, storage.dataBlocks))
		}
		err = storage.ReadFile(context.Background(), buf, bitrotReaders, offset, readLen, length)
		if err != nil {
			t.Fatal(err, offset, readLen)
		}
		got := buf.Bytes()
		if !bytes.Equal(expected, got) {
			t.Fatalf("read data is different from what was expected, offset=%d length=%d", offset, readLen)
		}
		buf.Reset()
	}
}

// Benchmarks

func benchmarkErasureRead(data, parity, dataDown, parityDown int, size int64, b *testing.B) {
	setup, err := newErasureTestSetup(data, parity, blockSizeV1)
	if err != nil {
		b.Fatalf("failed to create test setup: %v", err)
	}
	defer setup.Remove()
	storage, err := NewErasureStorage(context.Background(), setup.disks, data, parity, blockSizeV1)
	if err != nil {
		b.Fatalf("failed to create ErasureStorage: %v", err)
	}

	writers := make([]*bitrotWriter, len(storage.disks))
	for i, disk := range storage.disks {
		if disk == nil {
			continue
		}
		writers[i] = NewBitrotWriter(disk, "testbucket", "object", DefaultBitrotAlgorithm, DefaultBitrotBlockSize)
	}

	content := make([]byte, size)
	buffer := make([]byte, blockSizeV1, 2*blockSizeV1)
	n, err := storage.CreateFile(context.Background(), bytes.NewReader(content), writers, buffer)
	if err != nil {
		b.Fatalf("failed to create erasure test file: %v", err)
	}
	checksums := file.Checksums

	for i := 0; i < dataDown; i++ {
		writers[i] = nil
	}
	for i := data; i < data+parityDown; i++ {
		writers[i] = nil
	}

	b.ResetTimer()
	b.SetBytes(size)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bitrotReaders := make([]*bitrotReader, len(storage.disks))
		for index, disk := range storage.disks {
			if disk == OfflineDisk {
				continue
			}
			bitrotReaders[index] = NewBitrotReader(disk, "testbucket", "object", DefaultBitrotAlgorithm, DefaultBitrotBlockSize, writers[index].Close(), getErasureFileSize(int(storage.blockSize), n, storage.dataBlocks))
		}
		if err = storage.ReadFile(context.Background(), bytes.NewBuffer(content[:0]), bitrotReaders, 0, size, size); err != nil {
			panic(err)
		}
	}
}

func BenchmarkErasureReadQuick(b *testing.B) {
	const size = 12 * 1024 * 1024
	b.Run(" 00|00 ", func(b *testing.B) { benchmarkErasureRead(2, 2, 0, 0, size, b) })
	b.Run(" 00|X0 ", func(b *testing.B) { benchmarkErasureRead(2, 2, 0, 1, size, b) })
	b.Run(" X0|00 ", func(b *testing.B) { benchmarkErasureRead(2, 2, 1, 0, size, b) })
	b.Run(" X0|X0 ", func(b *testing.B) { benchmarkErasureRead(2, 2, 1, 1, size, b) })
}

func BenchmarkErasureRead_4_64KB(b *testing.B) {
	const size = 64 * 1024
	b.Run(" 00|00 ", func(b *testing.B) { benchmarkErasureRead(2, 2, 0, 0, size, b) })
	b.Run(" 00|X0 ", func(b *testing.B) { benchmarkErasureRead(2, 2, 0, 1, size, b) })
	b.Run(" X0|00 ", func(b *testing.B) { benchmarkErasureRead(2, 2, 1, 0, size, b) })
	b.Run(" X0|X0 ", func(b *testing.B) { benchmarkErasureRead(2, 2, 1, 1, size, b) })
	b.Run(" 00|XX ", func(b *testing.B) { benchmarkErasureRead(2, 2, 0, 2, size, b) })
	b.Run(" XX|00 ", func(b *testing.B) { benchmarkErasureRead(2, 2, 2, 0, size, b) })
}

func BenchmarkErasureRead_8_20MB(b *testing.B) {
	const size = 20 * 1024 * 1024
	b.Run(" 0000|0000 ", func(b *testing.B) { benchmarkErasureRead(4, 4, 0, 0, size, b) })
	b.Run(" 0000|X000 ", func(b *testing.B) { benchmarkErasureRead(4, 4, 0, 1, size, b) })
	b.Run(" X000|0000 ", func(b *testing.B) { benchmarkErasureRead(4, 4, 1, 0, size, b) })
	b.Run(" X000|X000 ", func(b *testing.B) { benchmarkErasureRead(4, 4, 1, 1, size, b) })
	b.Run(" 0000|XXXX ", func(b *testing.B) { benchmarkErasureRead(4, 4, 0, 4, size, b) })
	b.Run(" XX00|XX00 ", func(b *testing.B) { benchmarkErasureRead(4, 4, 2, 2, size, b) })
	b.Run(" XXXX|0000 ", func(b *testing.B) { benchmarkErasureRead(4, 4, 4, 0, size, b) })
}

func BenchmarkErasureRead_12_30MB(b *testing.B) {
	const size = 30 * 1024 * 1024
	b.Run(" 000000|000000 ", func(b *testing.B) { benchmarkErasureRead(6, 6, 0, 0, size, b) })
	b.Run(" 000000|X00000 ", func(b *testing.B) { benchmarkErasureRead(6, 6, 0, 1, size, b) })
	b.Run(" X00000|000000 ", func(b *testing.B) { benchmarkErasureRead(6, 6, 1, 0, size, b) })
	b.Run(" X00000|X00000 ", func(b *testing.B) { benchmarkErasureRead(6, 6, 1, 1, size, b) })
	b.Run(" 000000|XXXXXX ", func(b *testing.B) { benchmarkErasureRead(6, 6, 0, 6, size, b) })
	b.Run(" XXX000|XXX000 ", func(b *testing.B) { benchmarkErasureRead(6, 6, 3, 3, size, b) })
	b.Run(" XXXXXX|000000 ", func(b *testing.B) { benchmarkErasureRead(6, 6, 6, 0, size, b) })
}

func BenchmarkErasureRead_16_40MB(b *testing.B) {
	const size = 40 * 1024 * 1024
	b.Run(" 00000000|00000000 ", func(b *testing.B) { benchmarkErasureRead(8, 8, 0, 0, size, b) })
	b.Run(" 00000000|X0000000 ", func(b *testing.B) { benchmarkErasureRead(8, 8, 0, 1, size, b) })
	b.Run(" X0000000|00000000 ", func(b *testing.B) { benchmarkErasureRead(8, 8, 1, 0, size, b) })
	b.Run(" X0000000|X0000000 ", func(b *testing.B) { benchmarkErasureRead(8, 8, 1, 1, size, b) })
	b.Run(" 00000000|XXXXXXXX ", func(b *testing.B) { benchmarkErasureRead(8, 8, 0, 8, size, b) })
	b.Run(" XXXX0000|XXXX0000 ", func(b *testing.B) { benchmarkErasureRead(8, 8, 4, 4, size, b) })
	b.Run(" XXXXXXXX|00000000 ", func(b *testing.B) { benchmarkErasureRead(8, 8, 8, 0, size, b) })
}
