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
	"context"
	"strings"

	"github.com/minio/minio/cmd/logger"
	"github.com/minio/minio/pkg/pubsub"
	"github.com/minio/minio/pkg/trace"
)

//HTTPTraceSys holds global trace state
type HTTPTraceSys struct {
	peers  []*peerRESTClient
	pubsub *pubsub.PubSub
}

// NewTraceSys - creates new HTTPTraceSys with all nodes subscribed to
// the trace pub sub system
func NewTraceSys(ctx context.Context, endpoints EndpointList) *HTTPTraceSys {
	remoteHosts := getRemoteHosts(endpoints)
	remoteClients, err := getRestClients(remoteHosts)
	if err != nil {
		logger.FatalIf(err, "Unable to start httptrace sub system")
	}

	ps := pubsub.New()
	return &HTTPTraceSys{
		remoteClients, ps,
	}
}

// HasTraceListeners returns true if trace listeners are registered
// for this node or peers
func (sys *HTTPTraceSys) HasTraceListeners() bool {
	return sys != nil && sys.pubsub.HasSubscribers()
}

// Publish - publishes trace message to the http trace pubsub system
func (sys *HTTPTraceSys) Publish(traceMsg trace.Info) {
	sys.pubsub.Publish(traceMsg)
}

// Trace writes http trace to writer
func (sys *HTTPTraceSys) Trace(traceCh chan trace.Info, doneCh chan struct{}, trcAll bool) {
	go func() {
		ch := sys.pubsub.Subscribe()
		for {
			select {
			case entry := <-ch:
				trcInfo := entry.(trace.Info)
				// skip tracing of inter-node traffic if trcAll is false
				if !trcAll && strings.HasPrefix(trcInfo.ReqInfo.Path, "/minio") {
					continue
				}
				traceCh <- trcInfo
			case <-doneCh:
			case <-GlobalServiceDoneCh:
				sys.pubsub.Unsubscribe(ch)
				return
			}
		}
	}()

	for _, peer := range sys.peers {
		go func(peer *peerRESTClient) {
			ch, err := peer.Trace(doneCh, trcAll)
			if err != nil {
				return
			}
			for {
				select {
				case entry := <-ch:
					traceCh <- entry
				case <-doneCh:
				case <-GlobalServiceDoneCh:
					return
				}
			}
		}(peer)
	}
}
