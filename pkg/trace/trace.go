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

package trace

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TraceInfo - represents a trace record, additionally
// also reports errors if any while listening on trace.
type TraceInfo struct {
	NodeName string
	FuncName string
	ReqID    string
	ReqInfo  RequestInfo
	RespInfo ResponseInfo
	Err      error
}

// RequestInfo represents trace of http request
type RequestInfo struct {
	Time    time.Time
	Method  string
	Host    string
	URL     url.URL
	Headers map[string]string
	Body    string
}

// ResponseInfo represents trace of http request
type ResponseInfo struct {
	Time       time.Time
	Headers    map[string]string
	Body       string
	StatusCode int
}

// TargetIDErr returns error associated for a targetID
type TargetIDErr struct {
	// ID where the remove or send were initiated.
	ID TargetID
	// Stores any error while removing a target or while sending an event.
	Err error
}

func (ti TraceInfo) String() string {
	const timeFormat = "2006-01-02 15:04:05 -0700"
	ri := ti.ReqInfo
	rs := ti.RespInfo
	var s strings.Builder
	s.WriteString(fmt.Sprintf("[REQUEST %s] [%s] [%s]\n", ti.FuncName, ti.ReqID, ri.Time.Format(timeFormat)))
	s.WriteString(fmt.Sprintf("%s %s", ri.Method, ri.URL.Path))
	if ri.URL.RawQuery != "" {
		s.WriteString(fmt.Sprintf("?%s", ri.URL.RawQuery))
	}
	s.WriteString(fmt.Sprintf("\n"))
	s.WriteString(fmt.Sprintf("Host: %s\n", ri.Host))
	for k, v := range ri.Headers {
		s.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}
	s.WriteString(fmt.Sprintf("\n"))
	s.WriteString(fmt.Sprintf("%s", ri.Body))
	s.WriteString(fmt.Sprintf("\n"))
	s.WriteString(fmt.Sprintf("[RESPONSE] "))
	s.WriteString(fmt.Sprintf("[%s] [%s]\n", ti.ReqID, rs.Time.Format(timeFormat)))
	s.WriteString(fmt.Sprintf("%d %s\n", rs.StatusCode, http.StatusText(rs.StatusCode)))
	for k, v := range rs.Headers {
		s.WriteString(k + ":" + v)
		s.WriteString("\n")
	}
	s.WriteString(rs.Body)
	return s.String()
}

// Remove - closes and removes targets by given target IDs.
func (t *Listeners) Remove(targetids ...TargetID) <-chan TargetIDErr {
	errCh := make(chan TargetIDErr)

	go func() {
		defer close(errCh)

		var wg sync.WaitGroup
		for _, id := range targetids {
			target, ok := t.targets[id]
			if ok {
				wg.Add(1)
				go func(id TargetID, t Target) {
					defer wg.Done()
					if err := t.Close(); err != nil {
						errCh <- TargetIDErr{
							ID:  id,
							Err: err,
						}
					}
				}(id, target)
			}
		}
		wg.Wait()

		for _, id := range targetids {
			delete(t.targets, id)
		}
	}()

	return errCh
}

// List - returns available target IDs.
func (t *Listeners) List() []TargetID {
	keys := []TargetID{}
	for k := range t.targets {
		keys = append(keys, k)
	}
	return keys
}

// SendTraceByID - sends events to targets identified by target IDs.
func (t *Listeners) SendTraceByID(ti TraceInfo, targetIDs ...TargetID) <-chan TargetIDErr {
	errCh := make(chan TargetIDErr)

	go func() {
		defer close(errCh)
		var wg sync.WaitGroup
		for _, id := range targetIDs {
			target, ok := t.targets[id]
			if ok {
				wg.Add(1)
				go func(id TargetID, t Target) {
					defer wg.Done()
					if err := t.Send(ti); err != nil {
						errCh <- TargetIDErr{
							ID:  id,
							Err: err,
						}
					}
				}(id, target)
			}
		}
		wg.Wait()
	}()

	return errCh
}

// Listeners has a list of trace listeners.
type Listeners struct {
	sync.RWMutex
	targets map[TargetID]Target
}

// Add - adds event rules map, HTTP/PeerRPC client target to bucket name.
func (t *Listeners) Add(target Target) error {
	if _, ok := t.targets[target.ID()]; ok {
		return fmt.Errorf("target %v already exists", target.ID())
	}

	t.targets[target.ID()] = target
	return nil
}

// Exist - checks whether given target ID is a HTTP/PeerRPC client target or not.
func (t *Listeners) Exist(targetID TargetID) bool {
	_, ok := t.targets[targetID]
	return ok
}

// HasListener - returns true if any trace target is registered
func (t *Listeners) HasListener() bool {
	return len(t.targets) != 0
}

// RemoveListeners - closes and removes all HTTP/PeerRPC client targets.
func (t *Listeners) RemoveListeners() {
	t.Remove(t.List()...)
}

// RemoveListener - closes and removes target by target ID.
func (t *Listeners) RemoveListener(targetID TargetID) {
	t.Remove(targetID)
}

// Send - sends event data to all matching targets.
func (t *Listeners) Send(trace TraceInfo) (errs []TargetIDErr) {
	errCh := t.SendTraceByID(trace, t.List()...)
	for terr := range errCh {
		errs = append(errs, terr)
		if t.Exist(terr.ID) {
			t.RemoveListener(terr.ID)
		}
	}

	return errs
}

// NewTraceListeners - creates new tracelisteners system object.
func NewTraceListeners() *Listeners {
	return &Listeners{
		targets: make(map[TargetID]Target),
	}
}
