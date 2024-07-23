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
	"oras.land/oras/cmd/oras/internal/output"
	"oras.land/oras/internal/testutils"
	"os"
	"strings"
	"testing"
)

func TestTextCopyHandler(t *testing.T) {
	fetcher := testutils.NewMockFetcher(t)
	ctx := context.Background()
	builder := &strings.Builder{}
	printer := output.NewPrinter(builder, os.Stderr, false)
	ch := NewTextCopyHandler(printer, fetcher.Fetcher)
	if ch.OnCopySkipped(ctx, fetcher.OciImage) != nil {
		t.Error("OnCopySkipped() should not return an error")
	}
	if ch.PreCopy(ctx, fetcher.OciImage) != nil {
		t.Error("PreCopy() should not return an error")
	}
	if ch.PostCopy(ctx, fetcher.OciImage) != nil {
		t.Error("PostCopy() should not return an error")
	}
	if ch.OnMounted(ctx, fetcher.OciImage) != nil {
		t.Error("OnMounted() should not return an error")
	}
}
