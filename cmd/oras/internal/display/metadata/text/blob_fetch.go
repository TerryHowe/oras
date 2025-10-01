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

package text

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

// BlobFetchHandler handles text metadata output for blob fetch events.
type BlobFetchHandler struct {
	printer *output.Printer
	desc    ocispec.Descriptor
}

// NewBlobFetchHandler returns a new handler for Blob fetch events.
func NewBlobFetchHandler(printer *output.Printer, desc ocispec.Descriptor) metadata.BlobFetchHandler {
	return &BlobFetchHandler{
		printer: printer,
		desc:    desc,
	}
}

// OnBlobFetched implements metadata.BlobFetchHandler.
func (h *BlobFetchHandler) OnBlobFetched(target *option.Target) error {
	return h.printer.Println("Downloaded:", target.GetDisplayReference())
}

// Render implements metadata.BlobFetchHandler.
func (h *BlobFetchHandler) Render() error {
	return h.printer.Println("Digest:", h.desc.Digest)
}
