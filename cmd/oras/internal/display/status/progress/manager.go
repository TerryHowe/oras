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

// Manager is progress view master
type Manager interface {
	Add() (*Messenger, error)
	SendAndStop(desc ocispec.Descriptor, prompt string) error
	Close() error
}

type manager struct {
	status       []*status
	statusLock   sync.RWMutex
	console      console.Console
	updating     sync.WaitGroup
	renderDone   chan struct{}
	renderClosed chan struct{}
}

// NewManager initialized a new progress manager.
func NewManager(tty *os.File) (Manager, error) {
	c, err := console.NewConsole(tty)
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
	height, width := m.console.GetHeightWidth()
	lineCount := len(m.status) * 2
	offset := 0
	if lineCount > height {
		// skip statuses that cannot be rendered
		offset = lineCount - height
	}

	for ; offset < lineCount; offset += 2 {
		status, progress := m.status[offset/2].String(width)
		m.console.OutputTo(uint(lineCount-offset), status)
		m.console.OutputTo(uint(lineCount-offset-1), progress)
	}
}

// Add appends a new status with 2-line space for rendering.
func (m *manager) Add() (*Messenger, error) {
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
	messenger, err := m.Add()
	if err != nil {
		return err
	}
	messenger.Send(prompt, desc, desc.Size)
	messenger.Stop()
	return nil
}

func (m *manager) statusChan(s *status) *Messenger {
	ch := make(chan *status, BufferSize)
	m.updating.Add(1)
	go func() {
		defer m.updating.Done()
		for newStatus := range ch {
			s.update(newStatus)
		}
	}()
	return &Messenger{ch: ch}
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
