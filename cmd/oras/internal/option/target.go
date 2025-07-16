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

package option

import (
	"context"
	"errors"
	"fmt"
	"github.com/opencontainers/go-digest"
	"net/http"
	"oras.land/oras/cmd/oras/internal/resource"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/errcode"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

const (
	TargetTypeRemote    = "registry"
	TargetTypeOCILayout = "oci-layout"
)

// Target struct contains flags and arguments specifying one registry or image
// layout.
// Target implements oerrors.Handler interface.
type Target struct {
	Remote
	ociLayout    resource.OciLayout
	RawReference string
	Type         string
	Reference    string //contains tag or digest
	// Path contains
	//  - path to the OCI image layout target, or
	//  - registry and repository for the remote target
	Path string

	IsOCILayout bool

	prefix      string
	description string
}

// GetDisplayReference returns full printable reference.
func (target *Target) GetDisplayReference() string {
	return fmt.Sprintf("[%s] %s", target.Type, target.RawReference)
}

// setFlagDetails set directional flag prefix and description details
func (target *Target) setFlagDetails(prefix, description string) {
	if prefix != "" {
		target.prefix = prefix + "-"
		target.description = description + " "
	}
}

// ApplyFlags applies flags to a command flag set
// The complete form of the `target` flag is designed to be
//
//	--target type=<type>[[,<key>=<value>][...]]
//
// For better UX, the boolean flag `--oci-layout` is introduced as an alias of
// `--target type=oci-layout`.
// Since there is only one target type besides the default `registry` type,
// the full form is not implemented until a new type comes in.
func (target *Target) ApplyFlags(fs *pflag.FlagSet) {
	target.ApplyFlagsWithPrefix(fs, target.prefix, target.description)
	if target.prefix == "" {
		target.applyStdinFlags(fs)
	}
	fs.BoolVarP(&target.IsOCILayout, target.prefix+"oci-layout", "", false, "set "+target.description+"target as an OCI image layout")
	fs.StringVar(&target.Path, target.prefix+"oci-layout-path", "", "[Experimental] set the path for the "+target.description+"OCI image layout target")
}

// getRawReference generates raw reference string.
func getRawReference(rootPath string, tagOrDigest string) string {
	var delimiter string
	if _, err := digest.Parse(tagOrDigest); err == nil {
		// digest
		delimiter = "@"
	} else {
		// tag
		delimiter = ":"
	}
	return fmt.Sprintf("%s%s%s", rootPath, delimiter, tagOrDigest)
}

// Parse gets target options from user input.
func (target *Target) Parse(cmd *cobra.Command) error {
	if err := oerrors.CheckMutuallyExclusiveFlags(cmd.Flags(), target.flagPrefix+"oci-layout-path", target.flagPrefix+"oci-layout"); err != nil {
		return err
	}

	// oci-layout-path
	if target.Path != "" {
		target.IsOCILayout = true
		target.RawReference = getRawReference(target.Path, target.RawReference)
	}

	switch {
	case target.IsOCILayout:
		target.Type = TargetTypeOCILayout
		if len(target.headerFlags) != 0 {
			return errors.New("custom header flags cannot be used on an OCI image layout target")
		}
		target.ociLayout = resource.NewOciLayout(target.RawReference)
		return target.ociLayout.Parse()
	default:
		target.Type = TargetTypeRemote
		if ref, err := registry.ParseReference(target.RawReference); err != nil {
			return &oerrors.Error{
				OperationType:  oerrors.OperationTypeParseArtifactReference,
				Err:            fmt.Errorf("%q: %w", target.RawReference, err),
				Recommendation: "Please make sure the provided reference is in the form of <registry>/<repo>[:tag|@digest]",
			}
		} else {
			target.Reference = ref.Reference
			ref.Reference = ""
			target.Path = ref.String()
		}
		return target.Remote.Parse(cmd)
	}
}

func (target *Target) newRepository(debug bool, logger logrus.FieldLogger) (*remote.Repository, error) {
	return target.NewRepository(target.RawReference, debug, logger)
}

// NewTarget generates a new target based on target.
func (target *Target) NewTarget(debug bool, logger logrus.FieldLogger) (oras.GraphTarget, error) {
	switch target.Type {
	case TargetTypeOCILayout:
		return target.ociLayout.GetGraphTarget()
	case TargetTypeRemote:
		return target.newRepository(debug, logger)
	}
	return nil, fmt.Errorf("unknown target type: %q", target.Type)
}

// NewBlobDeleter generates a new blob deleter based on target.
func (target *Target) NewBlobDeleter(debug bool, logger logrus.FieldLogger) (resource.ResolvableDeleter, error) {
	switch target.Type {
	case TargetTypeOCILayout:
		return target.ociLayout.GetBlobDeleter()
	case TargetTypeRemote:
		repo, err := target.newRepository(debug, logger)
		if err != nil {
			return nil, err
		}
		return repo.Blobs(), nil
	}
	return nil, fmt.Errorf("unknown target type: %q", target.Type)
}

// NewManifestDeleter generates a new blob deleter based on target.
func (target *Target) NewManifestDeleter(debug bool, logger logrus.FieldLogger) (resource.ResolvableDeleter, error) {
	switch target.Type {
	case TargetTypeOCILayout:
		return target.ociLayout.GetManifestDeleter()
	case TargetTypeRemote:
		repo, err := target.newRepository(debug, logger)
		if err != nil {
			return nil, err
		}
		return repo.Manifests(), nil
	}
	return nil, fmt.Errorf("unknown target type: %q", target.Type)
}

// NewReadonlyTarget generates a new read only target based on target.
func (target *Target) NewReadonlyTarget(ctx context.Context, debug bool, logger logrus.FieldLogger) (resource.ReadOnlyGraphTagFinderTarget, error) {
	switch target.Type {
	case TargetTypeOCILayout:
		return target.ociLayout.GetReadonlyTarget(ctx)
	case TargetTypeRemote:
		return target.NewRepository(target.RawReference, debug, logger)
	}
	return nil, fmt.Errorf("unknown target type: %q", target.Type)
}

// EnsureReferenceNotEmpty returns formalized error when the reference is empty.
func (target *Target) EnsureReferenceNotEmpty(cmd *cobra.Command, allowTag bool) error {
	if target.Reference == "" {
		return oerrors.NewErrEmptyTagOrDigest(target.RawReference, cmd, allowTag)
	}
	return nil
}

// ModifyError handles error during cmd execution.
func (target *Target) ModifyError(cmd *cobra.Command, err error) (error, bool) {
	if target.IsOCILayout {
		// short circuit for non-remote targets
		return err, false
	}

	// handle errors for remote targets
	if errors.Is(err, auth.ErrBasicCredentialNotFound) {
		return target.DecorateCredentialError(err), true
	}

	if errors.Is(err, errdef.ErrNotFound) {
		// special handling for not found error returned by registry target
		cmd.SetErrPrefix(oerrors.RegistryErrorPrefix)
		return err, true
	}

	var errResp *errcode.ErrorResponse
	if !errors.As(err, &errResp) {
		// short circuit if the error is not an ErrorResponse
		return err, false
	}

	ref := registry.Reference{Registry: target.RawReference}
	if errResp.URL.Host != ref.Host() {
		// raw reference is not registry host
		var parseErr error
		ref, parseErr = registry.ParseReference(target.RawReference)
		if parseErr != nil {
			// this should not happen
			return err, false
		}
		if errResp.URL.Host != ref.Host() {
			// not handle if the error is not from the target
			return err, false
		}
	}

	cmd.SetErrPrefix(oerrors.RegistryErrorPrefix)
	ret := &oerrors.Error{
		Err: oerrors.ReportErrResp(errResp),
	}

	if ref.Registry == "docker.io" && errResp.StatusCode == http.StatusUnauthorized {
		if ref.Repository != "" && !strings.Contains(ref.Repository, "/") {
			// docker.io/xxx -> docker.io/library/xxx
			ref.Repository = "library/" + ref.Repository
			ret.Recommendation = fmt.Sprintf("Namespace seems missing. Do you mean `%s %s`?", cmd.CommandPath(), ref)
		}
	}
	return ret, true
}
