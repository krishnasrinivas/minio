/*
 * MinIO Cloud Storage, (C) 2018 MinIO, Inc.
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
	"context"
	"net/http"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"

	"github.com/minio/minio/cmd/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "minio_http_requests_duration_seconds",
			Help:    "Time taken by requests served by current MinIO server instance",
			Buckets: []float64{.001, .003, .005, .1, .5, 1},
		},
		[]string{"request_type"},
	)
	minioVersionInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "minio",
			Name:      "version_info",
			Help:      "Version of current MinIO server instance",
		},
		[]string{
			// current version
			"version",
			// commit-id of the current version
			"commit",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsDuration)
	prometheus.MustRegister(newMinioCollector())
	prometheus.MustRegister(minioVersionInfo)
}

// newMinioCollector describes the collector
// and returns reference of minioCollector
// It creates the Prometheus Description which is used
// to define metric and  help string
func newMinioCollector() *minioCollector {
	return &minioCollector{
		desc: prometheus.NewDesc("minio_stats", "Statistics exposed by MinIO server", nil, nil),
	}
}

// minioCollector is the Custom Collector
type minioCollector struct {
	desc *prometheus.Desc
}

// Describe sends the super-set of all possible descriptors of metrics
func (c *minioCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *minioCollector) Collect(ch chan<- prometheus.Metric) {

	// Expose MinIO's version information
	minioVersionInfo.WithLabelValues(Version, CommitID).Add(1)

	// Fetch disk space info
	objLayer := newObjectLayerFn()
	// Service not initialized yet
	if objLayer == nil {
		return
	}

	for _, endpoint := range globalEndpoints {
		sApi, err := newStorageAPI(endpoint)
		if err != nil {
			continue
		}
		dInfo, dErr := sApi.DiskInfo()
		if dErr != nil {
			continue
		}
		used := humanize.IBytes(dInfo.Total - dInfo.Free)
		unit, err := strconv.ParseFloat(strings.Fields(used)[0], 64)
		if err != nil {
			continue
		}

		// Total disk usage by the disk
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("minio", "disk", "storage_used"),
				"Total disk storage used by the disk",
				[]string{"mountpoint", "unit"}, nil),
			prometheus.GaugeValue,
			unit,
			endpoint.String(),
			strings.Fields(used)[1],
		)

		available := humanize.IBytes(dInfo.Free)
		unit, err = strconv.ParseFloat(strings.Fields(available)[0], 64)
		if err != nil {
			continue
		}

		// Total available space in the disk
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("minio", "disk", "storage_available"),
				"Total available space left in the disk",
				[]string{"mountpoint", "unit"}, nil),
			prometheus.GaugeValue,
			unit,
			endpoint.String(),
			strings.Fields(available)[1],
		)

		total := humanize.IBytes(dInfo.Total)
		unit, err = strconv.ParseFloat(strings.Fields(total)[0], 64)
		if err != nil {
			continue
		}

		// Total storage space of the disk
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("minio", "disk", "storage_total"),
				"Total space of the disk",
				[]string{"mountpoint", "unit"}, nil),
			prometheus.GaugeValue,
			unit,
			endpoint.String(),
			strings.Fields(total)[1],
		)

	}

	serverInfo := globalNotificationSys.ServerInfo(context.Background())
	// Once we have received all the ServerInfo from peers
	// add the local peer server info as well.
	serverInfo = append(serverInfo, ServerInfo{
		Addr: GetLocalPeer(globalEndpoints),
		Data: &ServerInfoData{
			StorageInfo: objLayer.StorageInfo(context.Background()),
			ConnStats:   globalConnStats.toServerConnStats(),
			HTTPStats:   globalHTTPStats.toServerHTTPStats(),
			Properties: ServerProperties{
				Uptime:       UTCNow().Sub(globalBootTime),
				Version:      Version,
				CommitID:     CommitID,
				DeploymentID: globalDeploymentID,
				SQSARN:       globalNotificationSys.GetARNList(),
				Region:       globalServerConfig.GetRegion(),
			},
		},
	})

	var totalDisks, offlineDisks int
	for _, info := range serverInfo {
		// Network Sent/Received Bytes
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("minio", "http", "tx_bytes_total"),
				"Total number of bytes sent by current MinIO server instance",
				[]string{"node"}, nil),
			prometheus.CounterValue,
			float64(info.Data.ConnStats.TotalOutputBytes),
			info.Addr,
		)

		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("minio", "http", "rx_bytes_total"),
				"Total number of bytes received by current MinIO server instance",
				[]string{"node"}, nil),
			prometheus.CounterValue,
			float64(info.Data.ConnStats.TotalInputBytes),
			info.Addr,
		)

		// Network Sent/Received Bytes (Outbound)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("minio", "http", "s3_tx_bytes_total"),
				"Total number of s3 bytes sent by current MinIO server instance",
				[]string{"node"}, nil),
			prometheus.CounterValue,
			float64(info.Data.ConnStats.S3OutputBytes),
			info.Addr,
		)

		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("minio", "http", "s3_rx_bytes_total"),
				"Total number of s3 bytes received by current MinIO server instance",
				[]string{"node"}, nil),
			prometheus.CounterValue,
			float64(info.Data.ConnStats.S3InputBytes),
			info.Addr,
		)

		// Total number of s3 requests.
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("minio", "http", "s3_request_count"),
				"Total number of s3 requests received by current MinIO server instance",
				[]string{"node"}, nil),
			prometheus.CounterValue,
			float64(info.Data.HTTPStats.TotalS3REQUESTStats),
			info.Addr,
		)

		// No of s3 requests failed.
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("minio", "http", "s3_error_count"),
				"Total number of failed s3 requests received by current MinIO server instance",
				[]string{"node"}, nil),
			prometheus.CounterValue,
			float64(info.Data.HTTPStats.FailedS3REQUESTStats),
			info.Addr,
		)

		// Setting totalDisks to 1 and offlineDisks to 0 in FS mode
		if info.Data.StorageInfo.Backend.Type == BackendFS {
			totalDisks = 1
			offlineDisks = 0
		} else {
			offlineDisks = info.Data.StorageInfo.Backend.OfflineDisks
			totalDisks = info.Data.StorageInfo.Backend.OfflineDisks + info.Data.StorageInfo.Backend.OnlineDisks
		}

		// MinIO Total Disk/Offline Disk
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("minio", "disks", "total"),
				"Total number of disks for current MinIO server instance",
				[]string{"node"}, nil),
			prometheus.GaugeValue,
			float64(totalDisks),
			info.Addr,
		)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName("minio", "disks", "offline"),
				"Total number of offline disks for current MinIO server instance",
				[]string{"node"}, nil),
			prometheus.GaugeValue,
			float64(offlineDisks),
			info.Addr,
		)

	}

}

func metricsHandler() http.Handler {
	registry := prometheus.NewRegistry()

	err := registry.Register(minioVersionInfo)
	logger.LogIf(context.Background(), err)

	err = registry.Register(httpRequestsDuration)
	logger.LogIf(context.Background(), err)

	err = registry.Register(newMinioCollector())
	logger.LogIf(context.Background(), err)

	gatherers := prometheus.Gatherers{
		registry,
	}
	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	return promhttp.InstrumentMetricHandler(
		registry,
		promhttp.HandlerFor(gatherers,
			promhttp.HandlerOpts{
				ErrorHandling: promhttp.ContinueOnError,
			}),
	)
}
