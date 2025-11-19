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

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	defaultDockerRegistryHost = "localhost:5000"
	defaultZotRegistryHost    = "localhost:5001"
)

// getRegistryConfig returns the registry configuration from environment variables
// or defaults for local testing
func getRegistryConfig() (dockerHost, zotHost string, plainHTTP bool) {
	dockerHost = os.Getenv("DOCKER_REGISTRY_HOST")
	if dockerHost == "" {
		dockerHost = defaultDockerRegistryHost
	}

	zotHost = os.Getenv("ZOT_REGISTRY_HOST")
	if zotHost == "" {
		zotHost = defaultZotRegistryHost
	}

	plainHTTP = os.Getenv("ORAS_E2E_PLAIN_HTTP") == "true"

	return dockerHost, zotHost, plainHTTP
}

// TestDockerRegistry tests basic operations against Docker Registry v2
func TestDockerRegistry(t *testing.T) {
	dockerHost, _, plainHTTP := getRegistryConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create a test repository
	repoName := dockerHost + "/test/e2e-artifact"
	repo, err := remote.NewRepository(repoName)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	repo.PlainHTTP = plainHTTP

	// Test pushing an artifact
	testPushPull(ctx, t, repo, "docker-registry")
}

// TestZotRegistry tests basic operations against Zot Registry
func TestZotRegistry(t *testing.T) {
	_, zotHost, plainHTTP := getRegistryConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create a test repository
	repoName := zotHost + "/test/e2e-artifact"
	repo, err := remote.NewRepository(repoName)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	repo.PlainHTTP = plainHTTP

	// Test pushing an artifact
	testPushPull(ctx, t, repo, "zot-registry")
}

// testPushPull is a helper function that tests push and pull operations
func testPushPull(ctx context.Context, t *testing.T, repo *remote.Repository, registryName string) {
	// Create a test artifact in memory
	memoryStore := memory.New()

	// Create a simple manifest with a blob
	blobContent := []byte("Hello from ORAS e2e test on " + registryName)
	blobDesc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    "sha256:...", // Will be calculated by Push
		Size:      int64(len(blobContent)),
		Annotations: map[string]string{
			ocispec.AnnotationTitle: "test-blob.txt",
		},
	}

	// Push blob to memory store
	if err := memoryStore.Push(ctx, blobDesc, bytes.NewReader(blobContent)); err != nil {
		t.Fatalf("failed to push blob to memory: %v", err)
	}

	// Create manifest
	manifestContent := ocispec.Manifest{
		Versioned: ocispec.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageManifest,
		Config: ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a", // empty json {}
			Size:      2,
		},
		Layers: []ocispec.Descriptor{blobDesc},
		Annotations: map[string]string{
			ocispec.AnnotationCreated: time.Now().UTC().Format(time.RFC3339),
		},
	}

	configContent := []byte("{}")
	configDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.config.v1+json",
		Digest:    "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
		Size:      int64(len(configContent)),
	}

	if err := memoryStore.Push(ctx, configDesc, bytes.NewReader(configContent)); err != nil {
		t.Fatalf("failed to push config to memory: %v", err)
	}

	manifestBytes, err := json.Marshal(manifestContent)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}

	manifestDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    "sha256:...", // Will be calculated
		Size:      int64(len(manifestBytes)),
	}

	if err := memoryStore.Push(ctx, manifestDesc, bytes.NewReader(manifestBytes)); err != nil {
		t.Fatalf("failed to push manifest to memory: %v", err)
	}

	// Copy from memory to registry
	tag := "test-tag"
	t.Logf("Pushing artifact to %s with tag %s", repo.Reference.Repository, tag)

	desc, err := oras.Copy(ctx, memoryStore, manifestDesc.Digest.String(), repo, tag, oras.DefaultCopyOptions)
	if err != nil {
		t.Fatalf("failed to push artifact to registry: %v", err)
	}

	t.Logf("Successfully pushed artifact: %s@%s", repo.Reference.Repository, desc.Digest)

	// Pull back from registry to verify
	t.Logf("Pulling artifact from %s with tag %s", repo.Reference.Repository, tag)

	pullStore := memory.New()
	pulledDesc, err := oras.Copy(ctx, repo, tag, pullStore, tag, oras.DefaultCopyOptions)
	if err != nil {
		t.Fatalf("failed to pull artifact from registry: %v", err)
	}

	if pulledDesc.Digest != desc.Digest {
		t.Errorf("digest mismatch: pushed %s, pulled %s", desc.Digest, pulledDesc.Digest)
	}

	t.Logf("Successfully pulled and verified artifact from %s", registryName)
}

// TestRegistryConnectivity tests that both registries are accessible
func TestRegistryConnectivity(t *testing.T) {
	dockerHost, zotHost, plainHTTP := getRegistryConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name string
		host string
	}{
		{"Docker Registry", dockerHost},
		{"Zot Registry", zotHost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, err := remote.NewRegistry(tt.host)
			if err != nil {
				t.Fatalf("failed to create registry client: %v", err)
			}
			reg.PlainHTTP = plainHTTP

			if err := reg.Ping(ctx); err != nil {
				t.Fatalf("failed to ping %s at %s: %v", tt.name, tt.host, err)
			}

			t.Logf("✓ Successfully connected to %s at %s", tt.name, tt.host)
		})
	}
}
