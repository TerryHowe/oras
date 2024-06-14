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
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/output"
	"sync"
)

// Prompts for pull events.
const (
	PullPromptDownloading = "Downloading"
	PullPromptPulled      = "Pulled     "
	PullPromptProcessing  = "Processing "
	PullPromptSkipped     = "Skipped    "
	PullPromptRestored    = "Restored   "
	PullPromptDownloaded  = "Downloaded "
)

// PrintSuccessorStatus prints transfer status of successors.
func PrintSuccessorStatus(ctx context.Context, desc ocispec.Descriptor, fetcher content.Fetcher, committed *sync.Map, print *output.Printer, prompt string) error {
	successors, err := content.Successors(ctx, fetcher, desc)
	if err != nil {
		return err
	}
	for _, s := range successors {
		name := s.Annotations[ocispec.AnnotationTitle]
		if v, ok := committed.Load(s.Digest.String()); ok && v != name {
			// Reprint status for deduplicated content
			if err := print.PrintStatus(s, prompt); err != nil {
				return err
			}
		}
	}
	return nil
}
