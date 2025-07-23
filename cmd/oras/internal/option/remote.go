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
	"errors"
	"fmt"
	"io"
	"oras.land/oras/cmd/oras/internal/resource"
	"oras.land/oras/internal/trace"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/errcode"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

const (
	caFileFlag                 = "ca-file"
	certFileFlag               = "cert-file"
	keyFileFlag                = "key-file"
	usernameFlag               = "username"
	passwordFlag               = "password"
	passwordFromStdinFlag      = "password-stdin"
	identityTokenFlag          = "identity-token"
	identityTokenFromStdinFlag = "identity-token-stdin"
)

// Remote options struct contains flags and arguments specifying one registry.
// Remote implements oerrors.Handler and interface.
type Remote struct {
	RawReference string
	DistributionSpec
	resource.RemoteProperties
	flagPrefix string

	applyDistributionSpec bool
	RemoteResource        resource.Remote
	store                 credentials.Store
}

// EnableDistributionSpecFlag set distribution specification flag as applicable.
func (remo *Remote) EnableDistributionSpecFlag() {
	remo.applyDistributionSpec = true
}

// ApplyFlags applies flags to a command flag set.
func (remo *Remote) ApplyFlags(fs *pflag.FlagSet) {
	remo.ApplyFlagsWithPrefix(fs, "", "")
	remo.applyStdinFlags(fs)
}

func (remo *Remote) applyStdinFlags(fs *pflag.FlagSet) {
	props := &remo.RemoteProperties
	fs.BoolVar(&props.SecretFromStdin, passwordFromStdinFlag, false, "read password from stdin")
	fs.BoolVar(&props.SecretFromStdin, identityTokenFromStdinFlag, false, "read identity token from stdin")
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary RemoteResource targets.
func (remo *Remote) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	var (
		shortUser     string
		shortPassword string
		shortHeader   string
	)
	if prefix == "" {
		shortUser, shortPassword = "u", "p"
		shortHeader = "H"
	}
	remo.flagPrefix = prefix

	if remo.applyDistributionSpec {
		remo.DistributionSpec.ApplyFlagsWithPrefix(fs, prefix, description)
	}
	props := &remo.RemoteProperties
	fs.StringVarP(&props.Username, remo.flagPrefix+usernameFlag, shortUser, "", description+"registry username")
	fs.StringVarP(&props.Secret, remo.flagPrefix+passwordFlag, shortPassword, "", description+"registry password or identity token")
	fs.StringVar(&props.Secret, remo.flagPrefix+identityTokenFlag, "", description+"registry identity token")
	fs.BoolVar(&props.Insecure, remo.flagPrefix+"insecure", false, "allow connections to "+description+"SSL registry without certs")
	fs.BoolVar(&props.PlainHTTP, remo.flagPrefix+"plain-http", false, "allow insecure connections to "+description+"registry without SSL check")
	fs.StringVar(&props.CACertFilePath, remo.flagPrefix+caFileFlag, "", "server certificate authority file for the RemoteResource "+description+"registry")
	fs.StringVarP(&props.CertFilePath, remo.flagPrefix+certFileFlag, "", "", "client certificate file for the RemoteResource "+description+"registry")
	fs.StringVarP(&props.KeyFilePath, remo.flagPrefix+keyFileFlag, "", "", "client private key file for the RemoteResource "+description+"registry")
	fs.StringArrayVar(&props.ResolveFlag, remo.flagPrefix+"resolve", nil, "customized DNS for "+description+"registry, formatted in `host:port:address[:address_port]`")
	fs.StringArrayVar(&props.Configs, remo.flagPrefix+"registry-config", nil, "`path` of the authentication file for "+description+"registry")
	fs.StringArrayVarP(&props.HeaderFlags, remo.flagPrefix+"header", shortHeader, nil, "add custom headers to "+description+"requests")
}

// CheckStdinConflict checks if PasswordFromStdin or IdentityTokenFromStdin of a
// *pflag.FlagSet conflicts with read file from input.
func CheckStdinConflict(flags *pflag.FlagSet) error {
	switch {
	case flags.Changed(passwordFromStdinFlag):
		return fmt.Errorf("`-` read file from input and `--%s` read password from input cannot be both used", passwordFromStdinFlag)
	case flags.Changed(identityTokenFromStdinFlag):
		return fmt.Errorf("`-` read file from input and `--%s` read identity token from input cannot be both used", identityTokenFromStdinFlag)
	}
	return nil
}

// Parse tries to read password with optional cmd prompt.
func (remo *Remote) Parse(cmd *cobra.Command) error {
	usernameAndIdTokenFlags := []string{remo.flagPrefix + usernameFlag, remo.flagPrefix + identityTokenFlag}
	passwordAndIdTokenFlags := []string{remo.flagPrefix + passwordFlag, remo.flagPrefix + identityTokenFlag}
	certFileAndKeyFileFlags := []string{remo.flagPrefix + certFileFlag, remo.flagPrefix + keyFileFlag}
	if cmd.Flags().Lookup(identityTokenFromStdinFlag) != nil {
		usernameAndIdTokenFlags = append(usernameAndIdTokenFlags, identityTokenFromStdinFlag)
		passwordAndIdTokenFlags = append(passwordAndIdTokenFlags, identityTokenFromStdinFlag)
	}
	if cmd.Flags().Lookup(passwordFromStdinFlag) != nil {
		passwordAndIdTokenFlags = append(passwordAndIdTokenFlags, passwordFromStdinFlag)
	}
	if err := oerrors.CheckMutuallyExclusiveFlags(cmd.Flags(), usernameAndIdTokenFlags...); err != nil {
		return err
	}
	if err := oerrors.CheckMutuallyExclusiveFlags(cmd.Flags(), passwordAndIdTokenFlags...); err != nil {
		return err
	}
	if err := oerrors.CheckRequiredTogetherFlags(cmd.Flags(), certFileAndKeyFileFlags...); err != nil {
		return err
	}

	err := remo.readSecret(cmd)
	if err != nil {
		return err
	}

	logger := trace.Logger(cmd.Context())
	debug := cmd.Flags().Changed("debug")
	remo.RemoteProperties.RawReference = remo.RawReference
	remo.RemoteResource = resource.NewRemote(remo.RemoteProperties, logger, debug)
	return remo.RemoteResource.Parse()
}

// readSecret tries to read password or identity token with
// optional cmd prompt.
func (remo *Remote) readSecret(cmd *cobra.Command) (err error) {
	if cmd.Flags().Changed(identityTokenFlag) {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "WARNING! Using --identity-token via the CLI is insecure. Use --identity-token-stdin.")
	} else if cmd.Flags().Changed(passwordFlag) {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "WARNING! Using --password via the CLI is insecure. Use --password-stdin.")
	} else if remo.SecretFromStdin {
		// Prompt for credential
		secret, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		remo.Secret = strings.TrimSuffix(string(secret), "\n")
		remo.Secret = strings.TrimSuffix(remo.Secret, "\r")
	}
	return nil
}

// configPath returns the config path of the credential store.
func (remo *Remote) configPath() (string, error) {
	if remo.store == nil {
		return "", errors.New("no credential store initialized")
	}
	if ds, ok := remo.store.(*credentials.DynamicStore); ok {
		return ds.ConfigPath(), nil
	}
	return "", errors.New("store doesn't support getting config path")
}

// ModifyError modifies error during cmd execution.
func (remo *Remote) ModifyError(cmd *cobra.Command, err error) (error, bool) {
	if errors.Is(err, auth.ErrBasicCredentialNotFound) {
		return remo.decorateCredentialError(err), true
	}

	var errResp *errcode.ErrorResponse
	if errors.As(err, &errResp) {
		cmd.SetErrPrefix(oerrors.RegistryErrorPrefix)
		return &oerrors.Error{
			Err: oerrors.ReportErrResp(errResp),
		}, true
	}
	return err, false
}

// decorateCredentialError decorate error with recommendation.
func (remo *Remote) decorateCredentialError(err error) *oerrors.Error {
	configPath := " "
	if path, pathErr := remo.configPath(); pathErr == nil {
		configPath += fmt.Sprintf("at %q ", path)
	}
	return &oerrors.Error{
		Err:            oerrors.TrimErrBasicCredentialNotFound(err),
		Recommendation: fmt.Sprintf(`Please check whether the registry credential stored in the authentication file%sis correct`, configPath),
	}
}
