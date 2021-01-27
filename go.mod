module github.com/minio/minio

go 1.14

require (
	cloud.google.com/go/storage v1.8.0
	git.apache.org/thrift.git v0.13.0
	github.com/Azure/azure-pipeline-go v0.2.2
	github.com/Azure/azure-storage-blob-go v0.10.0
	github.com/Azure/go-autorest/autorest/adal v0.9.1 // indirect
	github.com/Shopify/sarama v1.27.2
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/alecthomas/participle v0.2.1
	github.com/bcicen/jstream v1.0.1
	github.com/beevik/ntp v0.3.0
	github.com/cespare/xxhash/v2 v2.1.1
	github.com/cheggaaa/pb v1.0.29
	github.com/colinmarc/hdfs/v2 v2.1.1
	github.com/coredns/coredns v1.4.0
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/dchest/siphash v1.2.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/djherbis/atime v1.0.0
	github.com/dswarbrick/smart v0.0.0-20190505152634-909a45200d6d
	github.com/dustin/go-humanize v1.0.0
	github.com/eclipse/paho.mqtt.golang v1.3.0
	github.com/elazarl/go-bindata-assetfs v1.0.0
	github.com/fatih/color v1.10.0
	github.com/fatih/structs v1.1.0
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/google/martian v2.1.1-0.20190517191504-25dcb96d9e51+incompatible // indirect
	github.com/google/uuid v1.1.2
	github.com/gopherjs/gopherjs v0.0.0-20190328170749-bb2674552d8f // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.12.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.7 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/hashicorp/vault/api v1.0.4
	github.com/json-iterator/go v1.1.10
	github.com/klauspost/compress v1.11.3
	github.com/klauspost/cpuid v1.3.1
	github.com/klauspost/pgzip v1.2.5
	github.com/klauspost/readahead v1.3.1
	github.com/klauspost/reedsolomon v1.9.9
	github.com/lib/pq v1.8.0
	github.com/mattn/go-colorable v0.1.8
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mattn/go-isatty v0.0.12
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/miekg/dns v1.1.35
	github.com/minio/cli v1.22.0
	github.com/minio/highwayhash v1.0.0
	github.com/minio/md5-simd v1.1.2-0.20210107194844-776275e0c9a7 // indirect
	github.com/minio/minio-go/v7 v7.0.8-0.20210113181159-b8e436cfee9a
	github.com/minio/selfupdate v0.3.1
	github.com/minio/sha256-simd v0.1.1
	github.com/minio/simdjson-go v0.1.5
	github.com/minio/sio v0.2.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.3.1 // indirect
	github.com/mmcloughlin/avo v0.0.0-20200803215136-443f81d77104 // indirect
	github.com/montanaflynn/stats v0.5.0
	github.com/nats-io/nats-server/v2 v2.1.9
	github.com/nats-io/nats-streaming-server v0.19.0 // indirect
	github.com/nats-io/nats.go v1.10.0
	github.com/nats-io/nkeys v0.2.0 // indirect
	github.com/nats-io/stan.go v0.7.0
	github.com/ncw/directio v1.0.5
	github.com/nsqio/go-nsq v1.0.8
	github.com/olivere/elastic/v7 v7.0.22
	github.com/philhofer/fwd v1.1.1 // indirect
	github.com/pierrec/lz4 v2.5.2+incompatible
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.8.0
	github.com/rjeczalik/notify v0.9.2
	github.com/rs/cors v1.7.0
	github.com/secure-io/sio-go v0.3.0
	github.com/shirou/gopsutil v3.20.11+incompatible
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/streadway/amqp v1.0.0
	github.com/tidwall/gjson v1.3.5
	github.com/tidwall/sjson v1.0.4
	github.com/tinylib/msgp v1.1.3
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/valyala/tcplisten v0.0.0-20161114210144-ceec8f93295a
	github.com/willf/bitset v1.1.11 // indirect
	github.com/willf/bloom v2.0.3+incompatible
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200819165624-17cef6e3e9d5
	go.uber.org/zap v1.14.1 // indirect
	golang.org/x/crypto v0.0.0-20201124201722-c8d3bf9c5392
	golang.org/x/mod v0.4.0 // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	golang.org/x/tools v0.1.1-0.20210119222907-0a1a9685734a // indirect
	google.golang.org/api v0.25.0
	google.golang.org/genproto v0.0.0-20200527145253-8367513e4ece // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.5.0
	gopkg.in/ldap.v3 v3.0.3
	gopkg.in/yaml.v2 v2.3.0
	honnef.co/go/tools v0.0.1-2020.1.6 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)
