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
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Notifier is a tracked oras.Notifier.
type Notifier interface {
	Open() error
	Close() error
	IsTTY() bool
	ActionPrompt(desc ocispec.Descriptor) error
	DonePrompt(desc ocispec.Descriptor) error
	Prompt(desc ocispec.Descriptor, prompt string) error
}

type notifier struct {
	tty          *os.File
	actionPrompt string
	donePrompt   string
	manager      Manager
}

// NewNotifier creates a tty notifier.
func NewNotifier(tty *os.File, actionPrompt, donePrompt string) Notifier {
	return &notifier{
		tty:          tty,
		actionPrompt: actionPrompt,
		donePrompt:   donePrompt,
	}
}

// Open closes the tracking manager.
func (n *notifier) Open() (err error) {
	n.manager, err = NewManager(n.tty)
	return err
}

// Close closes the tracking manager.
func (n *notifier) Close() error {
	return n.manager.Close()
}

// IsTTY closes the tracking manager.
func (n *notifier) IsTTY() bool {
	return n.tty == nil
}

// ActionPrompt prompts the user with the provided prompt and descriptor.
func (n *notifier) ActionPrompt(desc ocispec.Descriptor) error {
	return n.manager.SendAndStop(desc, n.actionPrompt)
}

// DonePrompt prompts the user with the provided prompt and descriptor.
func (n *notifier) DonePrompt(desc ocispec.Descriptor) error {
	return n.manager.SendAndStop(desc, n.actionPrompt)
}

// Prompt prompts the user with the provided prompt and descriptor.
func (n *notifier) Prompt(desc ocispec.Descriptor, prompt string) error {
	return n.manager.SendAndStop(desc, prompt)
}
