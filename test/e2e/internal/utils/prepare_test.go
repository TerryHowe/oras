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

package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile_BlobHardLink(t *testing.T) {
	// Create a temporary directory structure mimicking OCI layout
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src", "blobs", "sha256")
	dstDir := filepath.Join(tempDir, "dst", "blobs", "sha256")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		t.Fatalf("failed to create dst dir: %v", err)
	}

	// Create a source blob file
	srcFile := filepath.Join(srcDir, "abc123")
	content := []byte("test blob content")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}

	// Copy the file
	dstFile := filepath.Join(dstDir, "abc123")
	if err := copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify both files exist and have the same content
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("failed to read dst file: %v", err)
	}
	if string(dstContent) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", dstContent, content)
	}

	// Verify they are hard links (same inode)
	srcInfo, err := os.Stat(srcFile)
	if err != nil {
		t.Fatalf("failed to stat src file: %v", err)
	}
	dstInfo, err := os.Stat(dstFile)
	if err != nil {
		t.Fatalf("failed to stat dst file: %v", err)
	}

	if !os.SameFile(srcInfo, dstInfo) {
		t.Errorf("files are not hard linked: src and dst have different inodes")
	}
}

func TestCopyFile_NonBlobRegularCopy(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	dstDir := filepath.Join(tempDir, "dst")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		t.Fatalf("failed to create dst dir: %v", err)
	}

	// Create a source non-blob file
	srcFile := filepath.Join(srcDir, "index.json")
	content := []byte("{\"test\": \"data\"}")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}

	// Copy the file
	dstFile := filepath.Join(dstDir, "index.json")
	if err := copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify content
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("failed to read dst file: %v", err)
	}
	if string(dstContent) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", dstContent, content)
	}

	// For non-blob files, we don't care if they're hard linked or not
	// Just verify the file was copied successfully
}
