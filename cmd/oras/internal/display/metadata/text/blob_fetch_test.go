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
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

type errorWriterBlobFetch struct{}

// Write implements the io.Writer interface and returns an error in Write.
func (w *errorWriterBlobFetch) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("got an error")
}

func TestNewBlobFetchHandler(t *testing.T) {
	content := []byte("test content")
	testDigest := digest.FromBytes(content)
	desc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    testDigest,
		Size:      int64(len(content)),
	}

	printer := output.NewPrinter(&bytes.Buffer{}, os.Stderr)
	handler := NewBlobFetchHandler(printer, desc)

	if handler == nil {
		t.Fatal("NewBlobFetchHandler() returned nil")
	}

	blobFetchHandler, ok := handler.(*BlobFetchHandler)
	if !ok {
		t.Fatal("NewBlobFetchHandler() did not return a *BlobFetchHandler")
	}

	if blobFetchHandler.printer != printer {
		t.Error("NewBlobFetchHandler() did not set printer correctly")
	}

	if blobFetchHandler.desc.Digest != desc.Digest {
		t.Error("NewBlobFetchHandler() did not set descriptor correctly")
	}
}

func TestBlobFetchHandler_OnBlobFetched(t *testing.T) {
	content := []byte("test content")
	testDigest := digest.FromBytes(content)

	tests := []struct {
		name           string
		target         *option.Target
		desc           ocispec.Descriptor
		out            *bytes.Buffer
		errorOut       bool
		wantErr        bool
		expectedOutput string
	}{
		{
			name: "successful fetch with registry target",
			target: &option.Target{
				Type:         "registry",
				RawReference: "example.com/repo@" + testDigest.String(),
				Path:         "example.com/repo",
			},
			desc: ocispec.Descriptor{
				MediaType: "application/octet-stream",
				Digest:    testDigest,
				Size:      int64(len(content)),
			},
			out:            &bytes.Buffer{},
			wantErr:        false,
			expectedOutput: "Downloaded: [registry] example.com/repo@" + testDigest.String() + "\n",
		},
		{
			name: "successful fetch with oci-layout target",
			target: &option.Target{
				Type:         "oci-layout",
				RawReference: "layout-dir@" + testDigest.String(),
				Path:         "layout-dir",
			},
			desc: ocispec.Descriptor{
				MediaType: "application/octet-stream",
				Digest:    testDigest,
				Size:      int64(len(content)),
			},
			out:            &bytes.Buffer{},
			wantErr:        false,
			expectedOutput: "Downloaded: [oci-layout] layout-dir@" + testDigest.String() + "\n",
		},
		{
			name: "error during write",
			target: &option.Target{
				Type:         "registry",
				RawReference: "example.com/repo@" + testDigest.String(),
				Path:         "example.com/repo",
			},
			desc: ocispec.Descriptor{
				MediaType: "application/octet-stream",
				Digest:    testDigest,
				Size:      int64(len(content)),
			},
			errorOut: true,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var printer *output.Printer
			if tt.errorOut {
				printer = output.NewPrinter(&errorWriterBlobFetch{}, os.Stderr)
			} else {
				printer = output.NewPrinter(tt.out, os.Stderr)
			}

			handler := &BlobFetchHandler{
				printer: printer,
				desc:    tt.desc,
			}

			err := handler.OnBlobFetched(tt.target)

			if (err != nil) != tt.wantErr {
				t.Errorf("BlobFetchHandler.OnBlobFetched() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.out != nil {
				output := tt.out.String()
				if output != tt.expectedOutput {
					t.Errorf("BlobFetchHandler.OnBlobFetched() output = %q, want %q", output, tt.expectedOutput)
				}
			}
		})
	}
}

func TestBlobFetchHandler_Render(t *testing.T) {
	content := []byte("test content")
	testDigest := digest.FromBytes(content)

	tests := []struct {
		name           string
		desc           ocispec.Descriptor
		out            *bytes.Buffer
		errorOut       bool
		wantErr        bool
		expectedOutput string
	}{
		{
			name: "successful render",
			desc: ocispec.Descriptor{
				MediaType: "application/octet-stream",
				Digest:    testDigest,
				Size:      int64(len(content)),
			},
			out:            &bytes.Buffer{},
			wantErr:        false,
			expectedOutput: "Digest: " + testDigest.String() + "\n",
		},
		{
			name: "error during write",
			desc: ocispec.Descriptor{
				MediaType: "application/octet-stream",
				Digest:    testDigest,
				Size:      int64(len(content)),
			},
			errorOut: true,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var printer *output.Printer
			if tt.errorOut {
				printer = output.NewPrinter(&errorWriterBlobFetch{}, os.Stderr)
			} else {
				printer = output.NewPrinter(tt.out, os.Stderr)
			}

			handler := &BlobFetchHandler{
				printer: printer,
				desc:    tt.desc,
			}

			err := handler.Render()

			if (err != nil) != tt.wantErr {
				t.Errorf("BlobFetchHandler.Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.out != nil {
				output := tt.out.String()
				if output != tt.expectedOutput {
					t.Errorf("BlobFetchHandler.Render() output = %q, want %q", output, tt.expectedOutput)
				}
			}
		})
	}
}

func TestBlobFetchHandler_InterfaceCompliance(t *testing.T) {
	var _ metadata.BlobFetchHandler = (*BlobFetchHandler)(nil)
}
