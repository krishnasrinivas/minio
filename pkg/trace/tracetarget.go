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

// Package trace implements http trace target client
package trace

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	xnet "github.com/minio/minio/pkg/net"
	"github.com/skyrings/skyring-common/tools/uuid"
)

// TargetID - holds identification and name strings of notification target.
type TargetID struct {
	ID   string
	Name string
}

// String - returns string representation.
func (tid TargetID) String() string {
	return tid.ID + ":" + tid.Name
}

// Target - trace target interface
type Target interface {
	ID() TargetID
	Send(TraceInfo) error
	Close() error
}

// HTTPClientTraceTarget - HTTP client target.
type HTTPClientTraceTarget struct {
	id        TargetID
	w         http.ResponseWriter
	traceCh   chan []byte
	DoneCh    chan struct{}
	stopCh    chan struct{}
	isStopped uint32
	isRunning uint32
}

func (target *HTTPClientTraceTarget) start() {
	go func() {
		defer func() {
			atomic.AddUint32(&target.isRunning, 1)

			// Close DoneCh to indicate we are done.
			close(target.DoneCh)
		}()

		write := func(trace []byte) error {
			if _, err := target.w.Write(trace); err != nil {
				return err
			}

			target.w.(http.Flusher).Flush()
			return nil
		}

		keepAliveTicker := time.NewTicker(500 * time.Millisecond)
		defer keepAliveTicker.Stop()

		for {
			select {
			case <-target.stopCh:
				// We are asked to stop.
				return
			case trace, ok := <-target.traceCh:
				if !ok {
					// Got read error.  Exit the goroutine.
					return
				}
				if err := write(trace); err != nil {
					// Got write error to the client.  Exit the goroutine.
					return
				}
			case <-keepAliveTicker.C:
				if err := write([]byte(" ")); err != nil {
					// Got write error to the client.  Exit the goroutine.
					return
				}
			}
		}
	}()
}

// Send - sends trace to HTTP client.
func (target *HTTPClientTraceTarget) Send(ti TraceInfo) error {
	if atomic.LoadUint32(&target.isRunning) != 0 {
		return errors.New("closed http connection")
	}
	data, err := json.Marshal(ti)
	if err != nil {
		return err
	}
	data = append(data, byte('\n'))
	select {
	case target.traceCh <- data:
		return nil
	case <-target.DoneCh:
		return errors.New("error in sending trace")
	}
}

// Close - closes underneath goroutine.
func (target *HTTPClientTraceTarget) Close() error {
	atomic.AddUint32(&target.isStopped, 1)
	if atomic.LoadUint32(&target.isStopped) == 1 {
		close(target.stopCh)
	}

	return nil
}

// ID - return id of target
func (target *HTTPClientTraceTarget) ID() TargetID {
	return target.id
}
func getNewUUID() (string, error) {
	uuid, err := uuid.New()
	if err != nil {
		return "", err
	}

	return uuid.String(), nil
}

// NewHTTPClientTraceTarget - creates new HTTP client target.
func NewHTTPClientTraceTarget(host xnet.Host, w http.ResponseWriter) (*HTTPClientTraceTarget, error) {
	uuid, err := getNewUUID()
	if err != nil {
		return nil, err
	}
	c := &HTTPClientTraceTarget{
		id:      TargetID{ID: "httpclient" + "+" + uuid + "+" + host.Name, Name: host.Name + ":" + host.Port.String()},
		w:       w,
		traceCh: make(chan []byte),
		DoneCh:  make(chan struct{}),
		stopCh:  make(chan struct{}),
	}
	c.start()
	return c, nil
}
