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

package status

import (
	"context"
	"oras.land/oras/cmd/oras/internal/display/status/progress"
	"sync"

	"oras.land/oras/cmd/oras/internal/output"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/display/status/track"
)

// TTYPushHandler handles TTY status output for push command.
type TTYPushHandler struct {
	tracked  track.GraphTarget
	notifier progress.Notifier
}

// NewTTYPushHandler returns a new handler for push status events.
func NewTTYPushHandler(notifier progress.Notifier) PushHandler {
	return &TTYPushHandler{
		notifier: notifier,
	}
}

// OnFileLoading is called before loading a file.
func (ph *TTYPushHandler) OnFileLoading(_ string) error {
	return nil
}

// OnEmptyArtifact is called when no file is loaded for an artifact push.
func (ph *TTYPushHandler) OnEmptyArtifact() error {
	return nil
}

// TrackTarget returns a tracked target.
func (ph *TTYPushHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, StopTrackTargetFunc, error) {
	err := ph.notifier.Open()
	if err != nil {
		return nil, nil, err
	}
	tracked := track.NewTarget(gt, ph.notifier)

	ph.tracked = tracked
	return tracked, tracked.Close, nil
}

// UpdateCopyOptions adds TTY status output to the copy options.
func (ph *TTYPushHandler) UpdateCopyOptions(opts *oras.CopyGraphOptions, fetcher content.Fetcher) {
	committed := &sync.Map{}
	opts.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return ph.notifier.Prompt(desc, PushPromptExists)
	}
	opts.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return output.PrintSuccessorStatus(ctx, desc, fetcher, committed, func(d ocispec.Descriptor) error {
			return ph.notifier.Prompt(d, PushPromptSkipped)
		})
	}
}

// NewTTYAttachHandler returns a new handler for attach status events.
func NewTTYAttachHandler(notifier progress.Notifier) AttachHandler {
	return NewTTYPushHandler(notifier)
}

// TTYPullHandler handles TTY status output for pull events.
type TTYPullHandler struct {
	tracked  track.GraphTarget
	notifier progress.Notifier
}

// NewTTYPullHandler returns a new handler for Pull status events.
func NewTTYPullHandler(notifier progress.Notifier) PullHandler {
	return &TTYPullHandler{
		notifier: notifier,
	}
}

// OnNodeDownloading implements PullHandler.
func (ph *TTYPullHandler) OnNodeDownloading(_ ocispec.Descriptor) error {
	return nil
}

// OnNodeDownloaded implements PullHandler.
func (ph *TTYPullHandler) OnNodeDownloaded(_ ocispec.Descriptor) error {
	return nil
}

// OnNodeProcessing implements PullHandler.
func (ph *TTYPullHandler) OnNodeProcessing(_ ocispec.Descriptor) error {
	return nil
}

// OnNodeRestored implements PullHandler.
func (ph *TTYPullHandler) OnNodeRestored(desc ocispec.Descriptor) error {
	return ph.notifier.Prompt(desc, PullPromptRestored)
}

// OnNodeSkipped implements PullHandler.
func (ph *TTYPullHandler) OnNodeSkipped(desc ocispec.Descriptor) error {
	return ph.notifier.Prompt(desc, PullPromptSkipped)
}

// TrackTarget returns a tracked target.
func (ph *TTYPullHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, StopTrackTargetFunc, error) {
	tracked := track.NewTarget(gt, ph.notifier)
	ph.tracked = tracked
	return tracked, tracked.Close, nil
}
