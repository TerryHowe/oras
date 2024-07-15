/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package progress

import (
	"errors"
	"flag"
	"os"
	"sync"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/status/console"
)

const (
	// BufferSize is the size of the status channel buffer.
	BufferSize       = 1
	framePerSecond   = 5
	bufFlushDuration = time.Second / framePerSecond
)

var errManagerStopped = errors.New("progress output manager has already been stopped")

// Status is print message channel
type Status chan *status

// Manager is progress view master
type Manager interface {
	Add() (Status, error)
	SendAndStop(desc ocispec.Descriptor, prompt string) error
	Close() error
}

type manager struct {
	status       []*status
	statusLock   sync.RWMutex
	console      *console.Console
	updating     sync.WaitGroup
	renderDone   chan struct{}
	renderClosed chan struct{}
}

type mockManager struct {
	status *status
}

// NewManager initialized a new progress manager.
func NewManager(f *os.File) (man Manager, err error) {
	if flag.Lookup("test.v") != nil {
		return &mockManager{}, nil
	}
	c, err := console.New(f)
	if err != nil {
		return nil, err
	}

	m := &manager{
		console:      c,
		renderDone:   make(chan struct{}),
		renderClosed: make(chan struct{}),
	}
	m.start()
	return m, nil
}

func (m *manager) start() {
	m.console.Save()
	renderTicker := time.NewTicker(bufFlushDuration)
	go func() {
		defer m.console.Restore()
		defer renderTicker.Stop()
		for {
			select {
			case <-m.renderDone:
				m.render()
				close(m.renderClosed)
				return
			case <-renderTicker.C:
				m.render()
			}
		}
	}()
}

func (m *manager) render() {
	m.statusLock.RLock()
	defer m.statusLock.RUnlock()
	// todo: update size in another routine
	width, height := m.console.Size()
	len := len(m.status) * 2
	offset := 0
	if len > height {
		// skip statuses that cannot be rendered
		offset = len - height
	}

	for ; offset < len; offset += 2 {
		status, progress := m.status[offset/2].String(width)
		m.console.OutputTo(uint(len-offset), status)
		m.console.OutputTo(uint(len-offset-1), progress)
	}
}

// Add appends a new status with 2-line space for rendering.
func (m *manager) Add() (Status, error) {
	if m.closed() {
		return nil, errManagerStopped
	}

	s := newStatus()
	m.statusLock.Lock()
	m.status = append(m.status, s)
	m.statusLock.Unlock()

	defer m.console.NewRow()
	defer m.console.NewRow()
	return m.statusChan(s), nil
}

// SendAndStop send message for descriptor and stop timing.
func (m *manager) SendAndStop(desc ocispec.Descriptor, prompt string) error {
	status, err := m.Add()
	if err != nil {
		return err
	}
	defer close(status)
	status <- NewStatusMessage(prompt, desc, desc.Size)
	status <- EndTiming()
	return nil
}

func (m *manager) statusChan(s *status) Status {
	ch := make(chan *status, BufferSize)
	m.updating.Add(1)
	go func() {
		defer m.updating.Done()
		for newStatus := range ch {
			s.Update(newStatus)
		}
	}()
	return ch
}

// Close stops all status and waits for updating and rendering.
func (m *manager) Close() error {
	if m.closed() {
		return errManagerStopped
	}
	// 1. wait for update to stop
	m.updating.Wait()
	// 2. stop periodic rendering
	close(m.renderDone)
	// 3. wait for the render stop
	<-m.renderClosed
	return nil
}

func (m *manager) closed() bool {
	select {
	case <-m.renderClosed:
		return true
	default:
		return false
	}
}

// Add appends a new status with 2-line space for rendering.
func (m *mockManager) Add() (Status, error) {
	ch := make(chan *status, BufferSize)
	return ch, nil
}

// SendAndStop send message for descriptor and stop timing.
func (m *mockManager) SendAndStop(_ ocispec.Descriptor, _ string) error {
	return nil
}

// Close stops all status and waits for updating and rendering.
func (m *mockManager) Close() error {
	return nil
}
