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
	"context"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"io"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/display/status/progress"
)

// GraphTarget is a tracked oras.GraphTarget.
type GraphTarget interface {
	oras.GraphTarget
	io.Closer
}

type graphTarget struct {
	oras.GraphTarget
	notifier     progress.Notifier
	actionPrompt string
	donePrompt   string
}

type referenceGraphTarget struct {
	*graphTarget
}

// NewTarget creates a new tracked Target.
func NewTarget(t oras.GraphTarget, notifier progress.Notifier) GraphTarget {
	var gt GraphTarget
	gt = &graphTarget{
		GraphTarget: t,
		notifier:    notifier,
	}
	return gt
}

// NewReferenceTarget creates a new tracked Target.
func NewReferenceTarget(t oras.GraphTarget, notifier progress.Notifier) GraphTarget {
	gt := &graphTarget{
		GraphTarget: t,
		notifier:    notifier,
	}
	var rgt GraphTarget
	rgt = &referenceGraphTarget{
		graphTarget: gt,
	}
	return rgt
}

// Mount mounts a blob from a specified repository. This method is invoked only
// by the `*remote.Repository` target.
func (t *graphTarget) Mount(ctx context.Context, desc ocispec.Descriptor, fromRepo string, getContent func() (io.ReadCloser, error)) error {
	mounter := t.GraphTarget.(registry.Mounter)
	return mounter.Mount(ctx, desc, fromRepo, getContent)
}

// Push pushes the content to the base oras.GraphTarget with tracking.
func (t *graphTarget) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	r, err := managedReader(content, expected, nil, t.actionPrompt, t.donePrompt)
	if err != nil {
		return err
	}
	defer r.Close()
	r.Start()
	if err := t.GraphTarget.Push(ctx, expected, r); err != nil {
		return err
	}
	r.Done()
	return nil
}

// PushReference pushes the content to the base oras.GraphTarget with tracking.
func (rgt *referenceGraphTarget) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	r, err := managedReader(content, expected, nil, rgt.actionPrompt, rgt.donePrompt)
	if err != nil {
		return err
	}
	defer r.Close()
	r.Start()
	err = rgt.GraphTarget.(registry.ReferencePusher).PushReference(ctx, expected, r, reference)
	if err != nil {
		return err
	}
	r.Done()
	return nil
}

// Close closes the tracking manager.
func (t *graphTarget) Close() error {
	return t.notifier.Close()
}
