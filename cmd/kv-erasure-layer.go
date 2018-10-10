package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"strings"

	"encoding/json"

	"github.com/minio/minio/cmd/logger"
	"github.com/minio/minio/pkg/hash"
	"github.com/tchap/go-patricia/patricia"
)

type KVErasureLayer struct {
	bucketName string
	disks      []KVAPI
	GatewayUnsupported
	listMap   map[string]*patricia.Trie
	listMapMu sync.Mutex
}

func (k *KVErasureLayer) listMapInsert(bucket, object string) {
	k.listMapMu.Lock()
	defer k.listMapMu.Unlock()

	trie := k.listMap[bucket]
	if trie == nil {
		return
	}
	trie.Insert(patricia.Prefix(object), 1)
}

func (k *KVErasureLayer) listMapDelete(bucket, object string) {
	k.listMapMu.Lock()
	defer k.listMapMu.Unlock()

	trie := k.listMap[bucket]
	if trie == nil {
		return
	}
	trie.Delete(patricia.Prefix(object))
}

func newKVErasureLayer(endpoints EndpointList) (*KVErasureLayer, error) {
	bucketName := os.Getenv("MINIO_BUCKET")
	if bucketName == "" {
		bucketName = "default"
	}
	kv := KVErasureLayer{
		bucketName: bucketName,
		listMap:    make(map[string]*patricia.Trie),
	}
	for _, endpoint := range endpoints {
		var disk KVAPI
		var err error
		if endpoint.IsLocal {
			disk, err = newKVSSD(endpoint.Path)
		} else {
			// disk = newKVRPC(endpoint)
			logger.FatalIf(fmt.Errorf("distributed minio not supported"), "exiting")
		}
		if err != nil {
			return nil, err
		}
		if disk == nil {
			return nil, fmt.Errorf("open failed on device %s", disk)
		}
		kv.disks = append(kv.disks, disk)
	}
	return &kv, nil
}

func (k *KVErasureLayer) ListBuckets(ctx context.Context) (buckets []BucketInfo, err error) {
	k.listMapMu.Lock()
	defer k.listMapMu.Unlock()
	for key := range k.listMap {
		buckets = append(buckets, BucketInfo{key, time.Now()})
	}
	return buckets, nil
}

func (k *KVErasureLayer) MakeBucketWithLocation(ctx context.Context, bucket, location string) error {
	k.listMapMu.Lock()
	defer k.listMapMu.Unlock()

	trie := k.listMap[bucket]
	if trie != nil {
		return BucketAlreadyExists{}
	}
	k.listMap[bucket] = patricia.NewTrie()
	return nil
}

func (k *KVErasureLayer) DeleteBucket(ctx context.Context, bucket string) error {
	k.listMapMu.Lock()
	defer k.listMapMu.Unlock()

	delete(k.listMap, bucket)
	return nil
}

func (k *KVErasureLayer) GetBucketInfo(ctx context.Context, bucket string) (bucketInfo BucketInfo, err error) {
	k.listMapMu.Lock()
	defer k.listMapMu.Unlock()
	if k.listMap[bucket] == nil {
		return bucketInfo, BucketNotFound{}
	}
	return BucketInfo{bucket, time.Now()}, nil
}

func kvGetOrderedDisks(disks []KVAPI, object string) []KVAPI {
	distribution := hashOrder(object, len(disks))
	shuffledDisks := make([]KVAPI, len(disks))
	// Shuffle disks for expected distribution.
	for index := range disks {
		blockIndex := distribution[index]
		shuffledDisks[blockIndex-1] = disks[index]
	}
	return shuffledDisks
}

func (k *KVErasureLayer) PutObject(ctx context.Context, bucket, object string, data *hash.Reader, metadata map[string]string) (objInfo ObjectInfo, err error) {
	objectLock := globalNSMutex.NewNSLock(bucket, object)
	if err := objectLock.GetLock(globalObjectTimeout); err != nil {
		return objInfo, err
	}
	defer objectLock.Unlock()

	dataDrives, parityDrives := getRedundancyCount(metadata[amzStorageClass], len(k.disks))
	erasure := newKVErasure(dataDrives, parityDrives)
	disks := kvGetOrderedDisks(k.disks, object)
	// disks := make([]KVAPI, len(k.disks))
	// copy(disks, k.disks)
	ids, n, err := erasure.Encode(ctx, disks, bucket, data)
	if err != nil {
		return objInfo, err
	}
	part := KVPart{ids, "", n, 1}
	nsEntry := KVNSEntry{
		"1",
		bucket,
		object,
		dataDrives, parityDrives,
		n,
		time.Now(),
		[]KVPart{part},
		mustGetUUID(),
	}
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(nsEntry)
	// err = gob.NewEncoder(&buf).Encode(nsEntry)
	if err != nil {
		return objInfo, err
	}
	if buf.Len() > kvValueSize {
		logger.LogIf(ctx, errUnexpected)
		return objInfo, errUnexpected
	}
	buf.Write(make([]byte, kvValueSize-buf.Len()))
	errs := make([]error, len(k.disks))
	var wg sync.WaitGroup
	for i := range k.disks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := kvAlloc()
			defer kvFree(b)
			copy(b, buf.Bytes())
			errs[i] = k.disks[i].Put(bucket, object, b)
		}(i)
	}
	wg.Wait()
	if err = reduceWriteQuorumErrs(ctx, errs, nil, (len(k.disks)/2)+1); err != nil {
		return objInfo, err
	}
	objInfo.Name = object
	objInfo.Bucket = bucket
	objInfo.ModTime = nsEntry.ModTime
	objInfo.Size = nsEntry.Size
	objInfo.ETag = nsEntry.ETag
	k.listMapInsert(bucket, object)
	return objInfo, nil
}

func (k *KVErasureLayer) GetObjectInfo(ctx context.Context, bucket, object string) (objInfo ObjectInfo, err error) {
	objectLock := globalNSMutex.NewNSLock(bucket, object)
	if err := objectLock.GetRLock(globalObjectTimeout); err != nil {
		return objInfo, err
	}
	defer objectLock.RUnlock()

	entries := make([]KVNSEntry, len(k.disks))
	errs := make([]error, len(k.disks))

	var wg sync.WaitGroup
	for i := range k.disks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := kvAlloc()
			defer kvFree(b)
			errs[i] = k.disks[i].Get(bucket, object, b)
			if errs[i] != nil {
				return
			}
			errs[i] = json.NewDecoder(bytes.NewBuffer(b)).Decode(&entries[i])
			// errs[i] = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&entries[i])
			if errs[i] != nil {
				errs[i] = errFileNotFound
			}
		}(i)
	}
	wg.Wait()
	entry, err := kvQuorumPart(ctx, entries, errs)
	if err != nil {
		return objInfo, toObjectErr(err, bucket, object)
	}

	objInfo.Bucket = bucket
	objInfo.Name = object
	objInfo.ModTime = entry.ModTime
	objInfo.Size = int64(entry.Size)
	objInfo.ETag = entry.ETag
	return objInfo, nil
}

func (k *KVErasureLayer) DeleteObject(ctx context.Context, bucket, object string) error {
	objectLock := globalNSMutex.NewNSLock(bucket, object)
	if err := objectLock.GetLock(globalObjectTimeout); err != nil {
		return err
	}
	defer objectLock.Unlock()

	entries := make([]KVNSEntry, len(k.disks))
	errs := make([]error, len(k.disks))

	var wg sync.WaitGroup
	for i := range k.disks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := kvAlloc()
			defer kvFree(b)
			errs[i] = k.disks[i].Get(bucket, object, b)
			if errs[i] != nil {
				return
			}
			errs[i] = json.NewDecoder(bytes.NewBuffer(b)).Decode(&entries[i])
			// errs[i] = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&entries[i])
			if errs[i] != nil {
				errs[i] = errFileNotFound
			}
		}(i)
	}
	wg.Wait()
	entry, err := kvQuorumPart(ctx, entries, errs)
	if err != nil {
		return toObjectErr(err, bucket, object)
	}

	var blockIDs []string
	for _, part := range entry.Parts {
		blockIDs = append(blockIDs, part.IDs...)
	}

	for _, blockID := range blockIDs {
		for i := range k.disks {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				k.disks[i].Delete(bucket, blockID)
			}(i)
		}
		wg.Wait()
	}
	for i := range k.disks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = k.disks[i].Delete(bucket, object)
		}(i)
	}
	wg.Wait()
	quorum := (len(k.disks) / 2) + 1
	if err := reduceWriteQuorumErrs(context.Background(), errs, nil, quorum); err != nil {
		return ObjectNotFound{}
	}
	k.listMapDelete(bucket, object)
	return nil
}

func kvQuorumPart(ctx context.Context, entries []KVNSEntry, errs []error) (KVNSEntry, error) {
	for i := range errs {
		if errs[i] == nil {
			return entries[i], nil
		}
	}
	// if err := reduceReadQuorumErrs(ctx, errs, nil, len(entries)); err != nil {
	// 	return KVNSEntry{}, err
	// }

	// modTimes := make([]time.Time, len(entries))
	// for i := range entries {
	// 	modTimes[i] = entries[i].ModTime
	// }
	// modTime, modTimeCount := commonTime(modTimes)
	// if modTimeCount < len(entries)/2 {
	// 	return KVNSEntry{}, errXLReadQuorum
	// }
	// zero := time.Time{}
	// if modTime == zero {
	// 	return KVNSEntry{}, ObjectNotFound{}
	// }
	// for i := range entries {
	// 	if modTime == entries[i].ModTime {
	// 		return entries[i], nil
	// 	}
	// }
	return KVNSEntry{}, errFileNotFound
}

func (k *KVErasureLayer) GetObject(ctx context.Context, bucket, object string, startOffset int64, length int64, writer io.Writer, etag string) (err error) {
	objectLock := globalNSMutex.NewNSLock(bucket, object)
	if err := objectLock.GetRLock(globalObjectTimeout); err != nil {
		return err
	}
	defer objectLock.RUnlock()
	entries := make([]KVNSEntry, len(k.disks))
	errs := make([]error, len(k.disks))

	var wg sync.WaitGroup
	for i := range k.disks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := kvAlloc()
			defer kvFree(b)
			errs[i] = k.disks[i].Get(bucket, object, b)
			if errs[i] != nil {
				return
			}
			errs[i] = json.NewDecoder(bytes.NewBuffer(b)).Decode(&entries[i])
			// errs[i] = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&entries[i])
			if errs[i] != nil {
				errs[i] = errFileNotFound
			}
		}(i)
	}
	wg.Wait()
	entry, err := kvQuorumPart(ctx, entries, errs)
	if err != nil {
		logger.LogIf(ctx, err)
		return err
	}
	if entry.Size == 0 {
		return nil
	}
	disks := kvGetOrderedDisks(k.disks, object)
	// disks := make([]KVAPI, len(k.disks))
	// copy(disks, k.disks)
	erasure := newKVErasure(entry.DataNumber, entry.ParityNumber)
	partIndex, partOffset := entry.partIndexAndOffset(startOffset)
	var totalBytesRead int64
	for ; partIndex < len(entry.Parts); partIndex++ {
		if length == totalBytesRead {
			break
		}
		part := entry.Parts[partIndex]
		partLength := part.Size - partOffset
		if partLength > (length - totalBytesRead) {
			partLength = length - totalBytesRead
		}
		err = erasure.Decode(ctx, disks, bucket, part.IDs, partOffset, partLength, entry.DataNumber, writer)
		if err != nil {
			return err
		}
		totalBytesRead += partLength
		partOffset = 0
	}
	return nil
}

func (k *KVErasureLayer) ListObjectsV2(ctx context.Context, bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (result ListObjectsV2Info, err error) {
	marker := continuationToken
	if marker == "" {
		marker = startAfter
	}

	loi, err := k.ListObjects(ctx, bucket, prefix, marker, delimiter, maxKeys)
	if err != nil {
		return result, err
	}

	listObjectsV2Info := ListObjectsV2Info{
		IsTruncated:           loi.IsTruncated,
		ContinuationToken:     continuationToken,
		NextContinuationToken: loi.NextMarker,
		Objects:               loi.Objects,
		Prefixes:              loi.Prefixes,
	}
	return listObjectsV2Info, err
}

func (k *KVErasureLayer) ListObjects(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (result ListObjectsInfo, err error) {
	var objects []string
	var commonPrefixes []string
	k.listMapMu.Lock()
	defer k.listMapMu.Unlock()

	trie := k.listMap[bucket]
	if trie == nil {
		return result, BucketNotFound{}
	}
	trie.Visit(func(objectByte patricia.Prefix, item patricia.Item) error {
		object := string(objectByte)
		if !strings.HasPrefix(object, prefix) {
			return nil
		}
		if strings.Compare(object, marker) <= 0 {
			return nil
		}
		if delimiter == "" {
			objects = append(objects, object)
			return nil
		}
		suffix := strings.TrimPrefix(object, prefix)
		i := strings.Index(suffix, slashSeparator)
		if i == -1 {
			objects = append(objects, object)
			return nil
		}
		fullPrefix := prefix + suffix[:i+1]
		for _, commonPrefix := range commonPrefixes {
			if commonPrefix == fullPrefix {
				return nil
			}
		}
		commonPrefixes = append(commonPrefixes, fullPrefix)
		return nil
	})
	for _, object := range objects {
		info, err := k.GetObjectInfo(ctx, bucket, object)
		if err != nil {
			continue
		}
		result.Objects = append(result.Objects, ObjectInfo{
			Bucket:  bucket,
			Name:    object,
			Size:    info.Size,
			ModTime: info.ModTime,
		})
	}
	result.Prefixes = commonPrefixes
	return result, nil
}

const kvMultipartPrefix = ".minio.sys/multipart"

func (k *KVErasureLayer) NewMultipartUpload(ctx context.Context, bucket, object string, metadata map[string]string) (uploadID string, err error) {
	uploadID = mustGetUUID()
	mpEntryName := pathJoin(kvMultipartPrefix, uploadID, object)
	objectLock := globalNSMutex.NewNSLock(bucket, mpEntryName)
	if err := objectLock.GetLock(globalObjectTimeout); err != nil {
		return "", err
	}
	defer objectLock.Unlock()

	dataDrives, parityDrives := getRedundancyCount(metadata[amzStorageClass], len(k.disks))
	disks := make([]KVAPI, len(k.disks))
	copy(disks, k.disks)
	nsEntry := KVNSEntry{
		"1",
		bucket,
		object,
		dataDrives, parityDrives,
		0,
		time.Now(),
		nil,
		"",
	}
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(nsEntry)
	// err = gob.NewEncoder(&buf).Encode(nsEntry)
	if err != nil {
		return "", err
	}
	buf.Write(make([]byte, kvValueSize-buf.Len()))
	errs := make([]error, len(k.disks))
	var wg sync.WaitGroup
	for i := range k.disks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := kvAlloc()
			defer kvFree(b)
			copy(b, buf.Bytes())
			errs[i] = k.disks[i].Put(bucket, mpEntryName, b)
		}(i)
	}
	wg.Wait()
	if err = reduceWriteQuorumErrs(ctx, errs, nil, (len(k.disks)/2)+1); err != nil {
		return "", err
	}
	return uploadID, nil
}

func (k *KVErasureLayer) PutObjectPart(ctx context.Context, bucket, object, uploadID string, partID int, data *hash.Reader) (info PartInfo, err error) {
	mpEntryName := pathJoin(kvMultipartPrefix, uploadID, object)
	entries := make([]KVNSEntry, len(k.disks))
	errs := make([]error, len(k.disks))
	disks := make([]KVAPI, len(k.disks))
	copy(disks, k.disks)

	objectLock := globalNSMutex.NewNSLock(bucket, mpEntryName)
	if err := objectLock.GetRLock(globalObjectTimeout); err != nil {
		return info, err
	}

	var wg sync.WaitGroup
	for i := range disks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := kvAlloc()
			defer kvFree(b)
			errs[i] = disks[i].Get(bucket, mpEntryName, b)
			if errs[i] != nil {
				disks[i] = nil
				return
			}
			errs[i] = json.NewDecoder(bytes.NewBuffer(b)).Decode(&entries[i])
			// errs[i] = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&entries[i])
			if errs[i] != nil {
				errs[i] = errFileNotFound
			}
		}(i)
	}
	wg.Wait()
	objectLock.RUnlock()
	entry, err := kvQuorumPart(ctx, entries, errs)
	if err != nil {
		logger.LogIf(ctx, err)
		return info, toObjectErr(err, bucket, object)
	}

	erasure := newKVErasure(entry.DataNumber, entry.ParityNumber)
	ids, n, err := erasure.Encode(ctx, disks, bucket, data)
	if err != nil {
		logger.LogIf(ctx, err)
		return info, err
	}

	objectLock = globalNSMutex.NewNSLock(bucket, mpEntryName)
	if err := objectLock.GetLock(globalObjectTimeout); err != nil {
		return info, err
	}
	defer objectLock.Unlock()

	kvPart := KVPart{}
	kvPart.ETag = GenETag()
	kvPart.IDs = ids
	kvPart.Size = n
	kvPart.PartNumber = partID
	entry.AddPart(kvPart)

	for i := range entries {
		entries[i] = entry
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(entry)
	// err = gob.NewEncoder(&buf).Encode(entry)
	if err != nil {
		return info, err
	}
	if buf.Len() > kvValueSize {
		logger.LogIf(ctx, errUnexpected)
		return info, errUnexpected
	}
	buf.Write(make([]byte, kvValueSize-buf.Len()))

	for i := range k.disks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := kvAlloc()
			defer kvFree(b)
			copy(b, buf.Bytes())
			errs[i] = k.disks[i].Put(bucket, mpEntryName, b)
		}(i)
	}
	wg.Wait()
	if err = reduceWriteQuorumErrs(ctx, errs, nil, (len(k.disks)/2)+1); err != nil {
		return info, err
	}

	info.ETag = kvPart.ETag
	info.PartNumber = partID
	info.LastModified = time.Now()
	info.Size = n
	return info, nil
}

func (k *KVErasureLayer) ListMultipartUploads(ctx context.Context, bucket, prefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (result ListMultipartsInfo, err error) {
	return result, nil
}

func (k *KVErasureLayer) ListObjectParts(ctx context.Context, bucket, object, uploadID string, partNumberMarker int, maxParts int) (result ListPartsInfo, err error) {
	return result, nil
}

func (k *KVErasureLayer) CompleteMultipartUpload(ctx context.Context, bucket, object, uploadID string, uploadedParts []CompletePart) (objInfo ObjectInfo, err error) {
	objectLock := globalNSMutex.NewNSLock(bucket, object)
	if err := objectLock.GetLock(globalObjectTimeout); err != nil {
		return objInfo, err
	}
	defer objectLock.Unlock()

	mpEntryName := pathJoin(kvMultipartPrefix, uploadID, object)
	objectLock = globalNSMutex.NewNSLock(bucket, mpEntryName)
	if err := objectLock.GetLock(globalObjectTimeout); err != nil {
		return objInfo, err
	}
	defer objectLock.Unlock()

	entries := make([]KVNSEntry, len(k.disks))
	errs := make([]error, len(k.disks))
	disks := make([]KVAPI, len(k.disks))
	copy(disks, k.disks)

	var wg sync.WaitGroup
	for i := range disks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := kvAlloc()
			defer kvFree(b)
			errs[i] = disks[i].Get(bucket, mpEntryName, b)
			if errs[i] != nil {
				disks[i] = nil
				return
			}
			errs[i] = json.NewDecoder(bytes.NewBuffer(b)).Decode(&entries[i])
			// errs[i] = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&entries[i])
			if errs[i] != nil {
				errs[i] = errFileNotFound
			}
		}(i)
	}
	wg.Wait()
	entry, err := kvQuorumPart(ctx, entries, errs)
	if err != nil {
		logger.LogIf(ctx, err)
		return objInfo, toObjectErr(err, bucket, object)
	}
	if len(uploadedParts) != len(entry.Parts) {
		logger.LogIf(ctx, errUnexpected)
		return objInfo, errUnexpected
	}
	for i := range uploadedParts {
		if uploadedParts[i].PartNumber != entry.Parts[i].PartNumber {
			logger.LogIf(ctx, errUnexpected)
			return objInfo, errUnexpected
		}
		if uploadedParts[i].ETag != entry.Parts[i].ETag {
			logger.LogIf(ctx, errUnexpected)
			return objInfo, errUnexpected
		}
	}
	entry.ETag = GenETag()
	for i := range entries {
		entries[i] = entry
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(entry)
	// err = gob.NewEncoder(&buf).Encode(entry)
	if err != nil {
		return objInfo, err
	}
	buf.Write(make([]byte, kvValueSize-buf.Len()))

	for i := range k.disks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := kvAlloc()
			defer kvFree(b)
			copy(b, buf.Bytes())
			errs[i] = k.disks[i].Put(bucket, object, b)
		}(i)
	}
	wg.Wait()
	if err = reduceWriteQuorumErrs(ctx, errs, nil, (len(k.disks)/2)+1); err != nil {
		return objInfo, err
	}
	objInfo.Name = object
	objInfo.Bucket = bucket
	objInfo.ModTime = entry.ModTime
	objInfo.Size = entry.Size
	objInfo.ETag = entry.ETag
	k.listMapInsert(bucket, object)
	return objInfo, nil
}
