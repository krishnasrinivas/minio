// +build ignore

package cmd

import (
	"io/ioutil"
	"os"
)

type kvxfs struct {
	dir string
}

func newKVXFS(device string) (*kvxfs, error) {
	dir := pathJoin(os.Getenv("HOME"), "tmp", device)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}
	return &kvxfs{dir}, nil
}

func (k *kvxfs) keyName(container, key string) string {
	return getSHA256Hash([]byte(pathJoin(container, key)))[:16]
}

func (k *kvxfs) Put(container, key string, value []byte) error {
	kvKey := k.keyName(container, key)
	return ioutil.WriteFile(pathJoin(k.dir, kvKey), value, 0666)
}

func (k *kvxfs) Get(container, key string, value []byte) error {
	kvKey := k.keyName(container, key)
	b, err := ioutil.ReadFile(pathJoin(k.dir, kvKey))
	if err != nil {
		if os.IsNotExist(err) {
			return errFileNotFound
		}
		return err
	}
	copy(value, b)
	return nil
}

func (k *kvxfs) Delete(container, key string) error {
	kvKey := k.keyName(container, key)
	err := os.Remove(pathJoin(k.dir, kvKey))
	if err != nil {
		if os.IsNotExist(err) {
			return errFileNotFound
		}
		return err
	}
	return nil
}

func (k *kvxfs) List() ([]string, error) {
	return nil, errDiskNotFound
}

func kvs_init_env() {

}

func kvAlloc() []byte {
	return make([]byte, kvValueSize)
}

func kvAllocBloc() []byte {
	if os.Getenv("KV_ENABLE_ERASURE") != "" {
		return make([]byte, (len(globalEndpoints)-1)*kvValueSize)
	}
	return make([]byte, len(globalEndpoints)*kvValueSize)
}

func kvFree(buf []byte) {
	return
}

func kvFreeBlock(buf []byte) {
	return
}
