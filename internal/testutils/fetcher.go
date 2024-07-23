package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/internal/docker"
	"testing"
)

type MockFetcher struct {
	t           *testing.T
	store       *memory.Store
	Fetcher     content.Fetcher
	Subject     ocispec.Descriptor
	Config      ocispec.Descriptor
	OciImage    ocispec.Descriptor
	DockerImage ocispec.Descriptor
	Index       ocispec.Descriptor
}

func NewMockFetcher(t *testing.T) (mockFetcher MockFetcher) {
	mockFetcher = MockFetcher{store: memory.New(), t: t}
	mockFetcher.Subject = mockFetcher.PushBlob(ocispec.MediaTypeImageLayer, []byte("blob"))
	imageType := "test.image"
	mockFetcher.Config = mockFetcher.PushBlob(imageType, []byte("config content"))
	mockFetcher.OciImage = mockFetcher.PushOCIImage(&mockFetcher.Subject, mockFetcher.Config)
	mockFetcher.DockerImage = mockFetcher.PushDockerImage(&mockFetcher.Subject, mockFetcher.Config)
	mockFetcher.Index = mockFetcher.PushIndex(mockFetcher.Subject)
	mockFetcher.Fetcher = mockFetcher.store
	return mockFetcher
}

func (mf *MockFetcher) PushBlob(mediaType string, blob []byte) ocispec.Descriptor {
	var blobs [][]byte
	blobs = append(blobs, blob)
	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    digest.FromBytes(blob),
		Size:      int64(len(blob)),
	}
	if err := mf.store.Push(context.Background(), desc, bytes.NewReader(blob)); err != nil {
		mf.t.Fatal(err)
	}
	return desc
}

func (mf *MockFetcher) pushImage(subject *ocispec.Descriptor, mediaType string, config ocispec.Descriptor, layers ...ocispec.Descriptor) ocispec.Descriptor {
	manifest := ocispec.Manifest{
		MediaType: mediaType,
		Subject:   subject,
		Config:    config,
		Layers:    layers,
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		mf.t.Fatal(err)
	}
	return mf.PushBlob(mediaType, manifestJSON)
}

func (mf *MockFetcher) PushOCIImage(subject *ocispec.Descriptor, config ocispec.Descriptor, layers ...ocispec.Descriptor) ocispec.Descriptor {
	return mf.pushImage(subject, ocispec.MediaTypeImageManifest, config, layers...)
}

func (mf *MockFetcher) PushDockerImage(subject *ocispec.Descriptor, config ocispec.Descriptor, layers ...ocispec.Descriptor) ocispec.Descriptor {
	return mf.pushImage(subject, docker.MediaTypeManifest, config, layers...)
}

func (mf *MockFetcher) PushIndex(manifests ...ocispec.Descriptor) ocispec.Descriptor {
	index := ocispec.Index{
		Manifests: manifests,
	}
	indexJSON, err := json.Marshal(index)
	if err != nil {
		mf.t.Fatal(err)
	}
	return mf.PushBlob(ocispec.MediaTypeImageIndex, indexJSON)
}
