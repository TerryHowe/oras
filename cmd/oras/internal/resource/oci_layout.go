package resource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras/cmd/oras/internal/fileref"
)

// OciLayout struct contains values describing an OCI image layout.
type OciLayout struct {
	RawReference string
	Reference    string
	Path         string
}

func NewOciLayout(rawReference string) OciLayout {
	return OciLayout{
		RawReference: rawReference,
	}
}

// Parse parses the raw in format of <Path>[:<tag>|@<digest>]
func (l *OciLayout) Parse() error {
	raw := l.RawReference
	var path string
	var ref string
	if idx := strings.LastIndex(raw, "@"); idx != -1 {
		// `digest` found
		path = raw[:idx]
		ref = raw[idx+1:]
	} else {
		// find `tag`
		var err error
		path, ref, err = fileref.Parse(raw, "")
		if err != nil {
			return errors.Join(err, errdef.ErrInvalidReference)
		}
	}
	l.Path = path
	l.Reference = ref

	return nil
}

func (l *OciLayout) GetGraphTarget() (*oci.Store, error) {
	return oci.New(l.Path)
}

func (l *OciLayout) GetBlobDeleter() (*oci.Store, error) {
	return oci.New(l.Path)
}

func (l *OciLayout) GetManifestDeleter() (*oci.Store, error) {
	return oci.New(l.Path)
}

// GetReadonlyTarget generates a new read only graph target based on resource.
func (l *OciLayout) GetReadonlyTarget(ctx context.Context) (ReadOnlyGraphTagFinderTarget, error) {
	info, err := os.Stat(l.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("invalid argument %q: failed to find Path %q: %w", l.RawReference, l.Path, err)
		}
		return nil, err
	}
	if info.IsDir() {
		return oci.NewFromFS(ctx, os.DirFS(l.Path))
	}
	store, err := oci.NewFromTar(ctx, l.Path)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, fmt.Errorf("%q does not look like a tar archive: %w", l.Path, err)
		}
		return nil, err
	}
	return store, nil
}
