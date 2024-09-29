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

package track

import (
	"fmt"
	"io"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/status/progress"
)

type Reader interface {
	io.Reader
	Done()
	Close()
	Start()
	StopManager()
}

type reader struct {
	base         io.Reader
	offset       int64
	actionPrompt string
	donePrompt   string
	descriptor   ocispec.Descriptor
	manager      progress.Manager
	messenger    *progress.Messenger
}

// NewReader returns a new reader with tracked progress.
func NewReader(r io.Reader, descriptor ocispec.Descriptor, actionPrompt string, donePrompt string, tty *os.File) (Reader, error) {
	fmt.Println("************* NewReader")
	fmt.Println(donePrompt)
	manager, err := progress.NewManager(tty)
	if err != nil {
		return nil, err
	}
	tr := reader{
		base:         r,
		descriptor:   descriptor,
		actionPrompt: actionPrompt,
		donePrompt:   donePrompt,
		manager:      manager,
	}
	tr.messenger, err = manager.Add()
	return &tr, err
}

// StopManager stops the messenger channel and related manager.
func (r *reader) StopManager() {
	r.Close()
	_ = r.manager.Close()
}

// Done sends message to mark the tracked progress as complete.
func (r *reader) Done() {
	fmt.Println("************* Done")
	fmt.Println(r.donePrompt)
	r.messenger.Send(r.donePrompt, r.descriptor, r.descriptor.Size)
	r.messenger.Stop()
}

// Close closes the update channel.
func (r *reader) Close() {
	r.messenger.Stop()
}

// Start sends the start timing to the messenger channel.
func (r *reader) Start() {
	r.messenger.Start()
}

// Read reads from the underlying reader and updates the progress.
func (r *reader) Read(p []byte) (int, error) {
	n, err := r.base.Read(p)
	if err != nil && err != io.EOF {
		return n, err
	}

	r.offset = r.offset + int64(n)
	if err == io.EOF {
		if r.offset != r.descriptor.Size {
			return n, io.ErrUnexpectedEOF
		}
	}
	r.messenger.Send(r.actionPrompt, r.descriptor, r.offset)
	return n, err
}
