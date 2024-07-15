package progress

import (
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func Test_NewManager(t *testing.T) {
	m, err := NewManager(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	var desc ocispec.Descriptor

	_, err = m.Add()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = m.SendAndStop(desc, "Done")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = m.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
