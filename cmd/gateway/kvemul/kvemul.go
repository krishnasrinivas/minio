package kvemul

/*
#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>
#include <stdint.h>
#include <inttypes.h>
#include <string.h>
#include <dlfcn.h>

#include "kvs_types.h"

int32_t (*pointer_kvs_init_env_opts)(kvs_init_options* options);
extern int32_t _Z17kvs_init_env_optsP16kvs_init_options(kvs_init_options* options);

int32_t (*pointer_kvs_init_env)(kvs_init_options* options);
extern int32_t _Z12kvs_init_envP16kvs_init_options(kvs_init_options* options);

struct kv_device_api *(*pointer_kvs_open_device)(const char *dev_path, const char *configfile);
extern struct kv_device_api *_Z15kvs_open_devicePKcS0_(const char *dev_path, const char *configfile);

int32_t (*pointer_kvs_close_device)(struct kv_device_api *user_dev);
extern int32_t _Z16kvs_close_deviceP13kv_device_api(struct kv_device_api *user_dev);

int32_t (*pointer_kvs_get_ioevents)(struct kv_device_api *dev, int maxevents);
extern int32_t _Z16kvs_get_ioeventsP13kv_device_apii(struct kv_device_api *dev, int maxevents);

int32_t (*pointer_kvs_store_tuple)(struct kv_device_api *dev, const kvs_key *key, const kvs_value *value, const kvs_store_context *ctx);
extern int32_t _Z15kvs_store_tupleP13kv_device_apiPK7kvs_keyPK9kvs_valuePK17kvs_store_context(struct kv_device_api *dev, const kvs_key *key, const kvs_value *value, const kvs_store_context *ctx);

int32_t (*pointer_kvs_retrieve_tuple)(struct kv_device_api *dev, const kvs_key *key, kvs_value *value, const kvs_retrieve_context *ctx);
extern int32_t _Z18kvs_retrieve_tupleP13kv_device_apiPK7kvs_keyP9kvs_valuePK20kvs_retrieve_context(struct kv_device_api *dev, const kvs_key *key, kvs_value *value, const kvs_retrieve_context *ctx);

int32_t (*pointer_kvs_delete_tuple)(struct kv_device_api *dev, const kvs_key *key, const kvs_delete_context *ctx);
extern int32_t _Z16kvs_delete_tupleP13kv_device_apiPK7kvs_keyPK18kvs_delete_context(struct kv_device_api *dev, const kvs_key *key, const kvs_delete_context *ctx);

int32_t (*pointer_kvs_open_iterator)(struct kv_device_api *dev, const kvs_iterator_context *ctx);
extern int32_t _Z17kvs_open_iteratorP13kv_device_apiPK20kvs_iterator_context(struct kv_device_api *dev, const kvs_iterator_context *ctx);

int32_t (*pointer_kvs_iterator_next)(struct kv_device_api *dev, kvs_iterator_handle *hiter, kvs_iterator_list *iter_list, const kvs_iterator_context *ctx);
extern int32_t _Z17kvs_iterator_nextP13kv_device_apiP19kvs_iterator_handleP17kvs_iterator_listPK20kvs_iterator_context(struct kv_device_api *dev, kvs_iterator_handle *hiter, kvs_iterator_list *iter_list, const kvs_iterator_context *ctx);

int32_t (*pointer_kvs_close_iterator)(struct kv_device_api *dev, kvs_iterator_handle *hiter, const kvs_iterator_context *ctx, uint32_t handler);
int32_t _Z18kvs_close_iteratorP13kv_device_apiP19kvs_iterator_handlePK20kvs_iterator_contextj(struct kv_device_api *dev, kvs_iterator_handle *hiter, const kvs_iterator_context *ctx, uint32_t handler);


const char *(*pointer_kvs_errstr)(int64_t errorno);
extern const char *_Z10kvs_errstrl(int64_t errorno);

extern void on_io_complete_callback();

static kvs_iterator_handle *handle;
static kvs_iterator_list iter_list;

static void on_io_complete(kv_iocb* ioctx) {
    const char *errStr = NULL;
    if(ioctx->result != 0) {
        errStr = pointer_kvs_errstr(ioctx->result);
    } else if (ioctx->opcode == IOCB_ASYNC_ITER_OPEN_CMD) {
        handle = ioctx->iter_handle;
        return;
    }
    on_io_complete_callback(ioctx->private1, errStr);
}

static void minio_kvs_init_env() {
  pointer_kvs_init_env_opts = _Z17kvs_init_env_optsP16kvs_init_options;
  pointer_kvs_init_env = _Z12kvs_init_envP16kvs_init_options;
  pointer_kvs_open_device = _Z15kvs_open_devicePKcS0_;
  pointer_kvs_close_device = _Z16kvs_close_deviceP13kv_device_api;
  pointer_kvs_get_ioevents = _Z16kvs_get_ioeventsP13kv_device_apii;
  pointer_kvs_store_tuple = _Z15kvs_store_tupleP13kv_device_apiPK7kvs_keyPK9kvs_valuePK17kvs_store_context;
  pointer_kvs_retrieve_tuple = _Z18kvs_retrieve_tupleP13kv_device_apiPK7kvs_keyP9kvs_valuePK20kvs_retrieve_context;
  pointer_kvs_delete_tuple = _Z16kvs_delete_tupleP13kv_device_apiPK7kvs_keyPK18kvs_delete_context;
  pointer_kvs_errstr = _Z10kvs_errstrl;
  pointer_kvs_open_iterator = _Z17kvs_open_iteratorP13kv_device_apiPK20kvs_iterator_context;
  pointer_kvs_iterator_next = _Z17kvs_iterator_nextP13kv_device_apiP19kvs_iterator_handleP17kvs_iterator_listPK20kvs_iterator_context;
  pointer_kvs_close_iterator = _Z18kvs_close_iteratorP13kv_device_apiP19kvs_iterator_handlePK20kvs_iterator_contextj;

  kvs_init_options options;

  int32_t i;
  i = pointer_kvs_init_env_opts(&options);

  options.memory.use_dpdk=0;

  options.aio.iocoremask=1;
  options.aio.queuedepth=64;
  options.aio.iocomplete_fn=on_io_complete;
  memset(options.udd.core_mask_str, 0, 256);
  options.udd.core_mask_str[0] = 123;
  options.udd.core_mask_str[1] = 125;
  memset(options.udd.cq_thread_mask, 0, 256);
  options.udd.cq_thread_mask[0] = 50;

  pointer_kvs_init_env(&options);
}

static struct kv_device_api* minio_kvs_open_device() {
    return pointer_kvs_open_device("/dev/kvemul", "./kvemul.conf");
}

static void minio_kvs_close_device(struct kv_device_api* kvd) {
    pointer_kvs_close_device(kvd);
}

static int32_t minio_kvs_put(struct kv_device_api *kvd, void *key, int keyLen, void *value, int valueLen, uint64_t chPtr) {
    kvs_key kvskey = {key, keyLen};
    kvs_value kvsvalue = {value, valueLen, 0};
    const kvs_store_context put_ctx = { KVS_STORE_POST, 0, (void*)chPtr, NULL};

    int i = pointer_kvs_store_tuple(kvd, &kvskey, &kvsvalue, &put_ctx);
    printf("kvs_store_tuple retval = %d\n", i);
}

static int32_t minio_kvs_get(struct kv_device_api *kvd,  void *key, int keyLen, void *value, int valueLen, uint64_t chPtr) {
    kvs_key kvskey = {key, keyLen};
    kvs_value kvsvalue = {value, valueLen, 0};
    const kvs_retrieve_context ret_ctx = { KVS_RETRIEVE_IDEMPOTENT, 0, (void*)chPtr, NULL };
    printf("minio_kvs_get key %s\n", (char *)key);
    pointer_kvs_retrieve_tuple(kvd, &kvskey, &kvsvalue, &ret_ctx);
    printf("valueLength %d\n", kvsvalue.length);
}

static int32_t minio_kvs_list(struct kv_device_api *kvd,  void *value, uint64_t chPtr) {
    kvs_iterator_context iter_ctx;
    iter_ctx.bitmask = 0x00000000;
    iter_ctx.bit_pattern = 0x00000000;
    iter_ctx.option = KVS_ITERATOR_OPT_KEY;
    int32_t i = pointer_kvs_open_iterator(kvd, &iter_ctx);
    while (pointer_kvs_get_ioevents(kvd, 1) == 0){};
    iter_list.it_list = (uint8_t*) value;
    iter_list.size = 32*1024;
    iter_ctx.option = KVS_ITER_DEFAULT;
    iter_ctx.private1 = (void*)chPtr;
    pointer_kvs_iterator_next(kvd, handle, &iter_list, &iter_ctx);
    while (pointer_kvs_get_ioevents(kvd, 1) == 0){};
    pointer_kvs_close_iterator(kvd, handle, &iter_ctx, i);
    handle = NULL;
}

static int32_t minio_kvs_get_ioevents(struct kv_device_api *kvd, int numEvents) {
    return pointer_kvs_get_ioevents(kvd, numEvents);
}

*/
import "C"

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"time"
	"unsafe"

	"encoding/gob"

	"strings"

	"github.com/minio/cli"
	minio "github.com/minio/minio/cmd"
	"github.com/minio/minio/pkg/auth"
	"github.com/minio/minio/pkg/hash"
)

type chanContainer struct {
	c chan error
}

//export on_io_complete_callback
func on_io_complete_callback(chPtr unsafe.Pointer, errStr *C.char) {
	var err error
	if errStr != nil {
		err = errors.New(C.GoString(errStr))
	}
	if chPtr == nil {
		return
	}
	chContainer := (*chanContainer)(chPtr)
	go func() {
		chContainer.c <- err
		// close(chContainer.c)
		// FiXME: take care of pointer_kvs_open_iterator cbk
	}()
}

func kvs_open_device() *C.struct_kv_device_api {
	return C.minio_kvs_open_device()
}

func kvs_close_device(kvd *C.struct_kv_device_api) {
	C.minio_kvs_close_device(kvd)
}

func kvs_put(kvd *C.struct_kv_device_api, key string, value []byte) error {
	key = key + ":"
	keyCStrinc := C.CString(key)
	errCh := make(chan error)
	chContainer := chanContainer{errCh}
	C.minio_kvs_put(kvd, unsafe.Pointer(keyCStrinc), C.int(len(key)), unsafe.Pointer(&value[0]), C.int(len(value)), C.ulong(uintptr(unsafe.Pointer(&chContainer))))
	for C.minio_kvs_get_ioevents(kvd, 1) == 0 {
	}

	err := <-errCh
	return err
}

func kvs_get(kvd *C.struct_kv_device_api, key string) ([]byte, error) {
	key = key + ":"
	keyBytes := []byte(key)
	valueLen := 2 * 1024 * 1024
	value := make([]byte, valueLen)
	errCh := make(chan error, 1)
	chContainer := chanContainer{errCh}
	C.minio_kvs_get(kvd, unsafe.Pointer(&keyBytes[0]), C.int(len(keyBytes)), unsafe.Pointer(&value[0]), C.int(len(value)), C.ulong(uintptr(unsafe.Pointer(&chContainer))))
	for C.minio_kvs_get_ioevents(kvd, 1) == 0 {
	}
	err := <-errCh
	return value, err
}

func kvs_list(kvd *C.struct_kv_device_api) ([]byte, error) {
	valueLen := 2 * 1024 * 1024
	value := make([]byte, valueLen)

	errCh := make(chan error, 1)
	chContainer := chanContainer{errCh}
	C.minio_kvs_list(kvd, unsafe.Pointer(&value[0]), C.ulong(uintptr(unsafe.Pointer(&chContainer))))
	for C.minio_kvs_get_ioevents(kvd, 1) == 0 {
	}
	err := <-errCh
	return value, err
}

func init() {
	const kvemulGatewayTemplate = `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  minio server /dev/kvemul [BUCKETNAME]

BUCKETNAME: optional bucket name (default bucket name is "default")

ENVIRONMENT VARIABLES:
  ACCESS:
     MINIO_ACCESS_KEY: Username or access key of S3 storage.
     MINIO_SECRET_KEY: Password or secret key of S3 storage.

  BROWSER:
     MINIO_BROWSER: To disable web browser access, set this value to "off".

  DOMAIN:
     MINIO_DOMAIN: To enable virtual-host-style requests, set this value to Minio host domain name.

  CACHE:
     MINIO_CACHE_DRIVES: List of mounted drives or directories delimited by ";".
     MINIO_CACHE_EXCLUDE: List of cache exclusion patterns delimited by ";".
     MINIO_CACHE_EXPIRY: Cache expiry duration in days.
     MINIO_CACHE_MAXUSE: Maximum permitted usage of the cache in percentage (0-100).

EXAMPLES:
  1. Start minio gateway kvemul backend.
     $ export MINIO_ACCESS_KEY=accesskey
     $ export MINIO_SECRET_KEY=secretkey
     $ {{.HelpName}}

`

	minio.RegisterGatewayCommand(cli.Command{
		Name:               "/dev/kvemul",
		Usage:              "KV Emulator",
		Action:             kvemulGatewayMain,
		CustomHelpTemplate: kvemulGatewayTemplate,
		HideHelpCommand:    true,
	})
}

type KVEmul struct {
	bucketName string
}

func (k *KVEmul) Name() string {
	return "kvemul"
}

func (k *KVEmul) NewGatewayLayer(creds auth.Credentials) (minio.ObjectLayer, error) {
	C.minio_kvs_init_env()
	emul := &KVEmulObject{bucketName: k.bucketName, kvd: kvs_open_device()}
	// go emul.GetEvents()
	return emul, nil
}

func (k *KVEmul) Production() bool {
	return true
}

// Handler for 'minio gateway s3' command line.
func kvemulGatewayMain(ctx *cli.Context) {
	// // Start the gateway..
	// minio.StartGateway(ctx, &S3{args.First()})
	args := ctx.Args()
	if !ctx.Args().Present() {
		args = cli.Args{"default"}
	}

	// Start the gateway..
	minio.StartGateway(ctx, &KVEmul{args.First()})
}

type Object struct {
	Padding  []byte
	Contents []byte
}

type KVEmulObject struct {
	bucketName string
	kvd        *C.struct_kv_device_api
	minio.GatewayUnsupported
}

func (k *KVEmulObject) GetEvents() {
	for {
		C.minio_kvs_get_ioevents(k.kvd, 1)
		time.Sleep(time.Millisecond * 10)
	}
}

func (k *KVEmulObject) ListBuckets(ctx context.Context) (buckets []minio.BucketInfo, err error) {
	return []minio.BucketInfo{{k.bucketName, time.Now()}}, nil
}

func (k *KVEmulObject) GetBucketInfo(ctx context.Context, bucket string) (bucketInfo minio.BucketInfo, err error) {
	if bucket != k.bucketName {
		return bucketInfo, minio.BucketNotFound{Bucket: bucket}
	}
	return minio.BucketInfo{k.bucketName, time.Now()}, nil
}

func (k *KVEmulObject) PutObject(ctx context.Context, bucket, object string, data *hash.Reader, metadata map[string]string) (objInfo minio.ObjectInfo, err error) {
	if len(object) < 16 {
		return objInfo, minio.NotImplemented{}
	}
	b, err := ioutil.ReadAll(data)
	if err != nil {
		return objInfo, minio.ErrorRespToObjectError(err, bucket)
	}
	o := Object{}
	o.Padding = make([]byte, 64)
	o.Contents = b
	var w bytes.Buffer
	if err = gob.NewEncoder(&w).Encode(&o); err != nil {
		return objInfo, err
	}
	if err = kvs_put(k.kvd, object, w.Bytes()); err != nil {
		return objInfo, minio.ErrorRespToObjectError(err, bucket)
	}
	return k.GetObjectInfo(ctx, bucket, object)
}

func (k *KVEmulObject) GetObjectInfo(ctx context.Context, bucket, object string) (objInfo minio.ObjectInfo, err error) {
	if len(object) < 16 {
		return objInfo, minio.NotImplemented{}
	}
	b, err := kvs_get(k.kvd, object)
	if err != nil {
		return objInfo, minio.ObjectNotFound{}
	}
	o := Object{}
	if err = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&o); err != nil {
		return objInfo, err
	}
	objInfo.Bucket = bucket
	objInfo.Name = object
	objInfo.ModTime = time.Now()
	objInfo.Size = int64(len(o.Contents))
	return objInfo, nil
}

func (k *KVEmulObject) GetObject(ctx context.Context, bucket, object string, startOffset int64, length int64, writer io.Writer, etag string) (err error) {
	if len(object) < 16 {
		return minio.NotImplemented{}
	}
	b, err := kvs_get(k.kvd, object)
	if err != nil {
		return minio.ObjectNotFound{}
	}
	o := Object{}
	if err = gob.NewDecoder(bytes.NewBuffer(b)).Decode(&o); err != nil {
		return err
	}
	writer.Write(o.Contents[startOffset:length])
	return nil
}

func (k *KVEmulObject) ListObjects(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (result minio.ListObjectsInfo, err error) {
	b, err := kvs_list(k.kvd)
	i := 0
	for ; i < len(b); i++ {
		if b[i] == 0 {
			break
		}
	}
	b = b[:i]
	for _, object := range strings.Split(string(b), ":") {
		if len(object) == 0 {
			continue
		}
		objInfo, err := k.GetObjectInfo(ctx, bucket, object)
		if err != nil {
			continue
		}
		result.Objects = append(result.Objects, objInfo)
	}
	return result, nil
}
