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

	"github.com/minio/minio/cmd/logger"
	"github.com/minio/minio/pkg/pubsub"
	"github.com/minio/minio/pkg/trace"
)

// // peerSubscriber represents the peer trace listener
// type peerSubscriber struct {
// 	mutex  sync.Mutex
// 	client *peerRESTClient
// 	count  uint64 //count of trace clients for that peer
// }

// // IsSubscribed returns true if any of the peers are
// // listening to trace messages
// func (s *peerSubscriber) IsSubscribed() bool {
// 	return s.count > 0
// }

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

	// peers := make(map[xnet.Host]*peerSubscriber)

	// for _, client := range remoteClients {
	// 	peers[*client.host] = &peerSubscriber{client: client, count: 0}
	// 	// subscribe peers to the trace pubsub system
	// 	ch := ps.Subscribe()
	// 	go func(client *peerRESTClient, ch chan interface{}) {
	// 		for {
	// 			msg := <-ch
	// 			if err := client.SendTrace(client.host.Name, msg.(trace.Info)); err != nil {
	// 				logger.GetReqInfo(ctx).AppendTags("remotePeer", client.host.Name)
	// 				logger.LogIf(ctx, err)
	// 			}
	// 		}
	// 	}(client, ch)
	// }

	// pss := &HTTPTraceSys{
	// 	peerSubscribers: peers,
	// 	traceListeners:  trace.NewTraceListeners(),
	// 	pubsub:          ps,
	// }

	// // subscribe self to pubsub system
	// ch := ps.Subscribe()
	// go func(pss *HTTPTraceSys, ch chan interface{}) {
	// 	for {
	// 		msg := <-ch
	// 		pss.traceListeners.Send(msg.(trace.Info))
	// 	}
	// }(pss, ch)

	// return pss
}

// // AddRemoteTraceTarget registers a trace listener
// func (sys *HTTPTraceSys) AddRemoteTraceTarget(ctx context.Context, t trace.Target) error {
// 	thisAddr, err := xnet.ParseHost(GetLocalPeer(globalEndpoints))
// 	if err != nil {
// 		return err
// 	}

// 	if err := sys.traceListeners.Add(t); err != nil {
// 		return err
// 	}

// 	// inform peers that this node registered a new trace listener
// 	for _, peer := range sys.peerSubscribers {
// 		if err := peer.client.AddTraceListener(*thisAddr); err != nil {
// 			logger.GetReqInfo(ctx).AppendTags("remotePeer", peer.client.host.Name)
// 			logger.LogIf(ctx, err)
// 		}
// 	}
// 	return nil
// }

// // RemoveRemoteTraceTarget removes a trace listener by targetID
// func (sys *HTTPTraceSys) RemoveRemoteTraceTarget(ctx context.Context, targetID trace.TargetID) error {
// 	sys.traceListeners.Remove(targetID)

// 	thisAddr, err := xnet.ParseHost(GetLocalPeer(globalEndpoints))
// 	if err != nil {
// 		return err
// 	}
// 	// inform peers that this node de-registered a trace listener
// 	for _, peer := range sys.peerSubscribers {
// 		if err := peer.client.RemoveTraceListener(*thisAddr); err != nil {
// 			logger.GetReqInfo(ctx).AppendTags("remotePeer", peer.client.host.Name)
// 			logger.LogIf(ctx, err)
// 		}
// 	}
// 	return nil
// }

// // RemoveTraceListeners removes all traceListeners
// func (sys *HTTPTraceSys) RemoveTraceListeners() {
// 	sys.traceListeners.RemoveListeners()
// }

// HasTraceListeners returns true if trace listeners are registered
// for this node or peers
func (sys *HTTPTraceSys) HasTraceListeners() bool {
	return sys.pubsub.HasSubscribers()
	// return sys.traceListeners.HasListener() || sys.HasPeerListeners()
}

// // AddPeerTraceListener increments count of trace clients for that peer
// func (sys *HTTPTraceSys) AddPeerTraceListener(p *peerRESTClient) {
// 	for _, peer := range sys.peerSubscribers {
// 		if peer.client.host.String() == p.host.String() {
// 			peer.mutex.Lock()
// 			defer peer.mutex.Unlock()
// 			peer.count++
// 			break
// 		}
// 	}
// }

// // RemovePeerTraceListener decrements count of trace listeners for that peer
// func (sys *HTTPTraceSys) RemovePeerTraceListener(peerAddr string) {
// 	for _, peer := range sys.peerSubscribers {
// 		if peer.client.host.String() == peerAddr {
// 			peer.mutex.Lock()
// 			defer peer.mutex.Unlock()
// 			peer.count--
// 			break
// 		}
// 	}
// }

// // HasPeerListeners returns true if any peer has clients listening to
// // http trace logs.
// func (sys *HTTPTraceSys) HasPeerListeners() bool {
// 	for _, peer := range sys.peerSubscribers {
// 		if peer.IsSubscribed() {
// 			return true
// 		}
// 	}
// 	return false
// }

// Publish - publishes trace message to the http trace pubsub system
func (sys *HTTPTraceSys) Publish(traceMsg trace.Info) {
	sys.pubsub.Publish(traceMsg)
}

func (sys *HTTPTraceSys) Trace(w io.Writer) {
	doneCh := make(chan struct{})
	for _, peer := sys.peers {
		go func(peer *peerRESTClient) {
			ch := peer.Trace(doneCh)
			for entry := range ch {
				err := w.Write(entry)
				if err != nil {
					close(doneCh)
				}
			}
		}(peer)
	}
}
