package resource

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
	"oras.land/oras/internal/credential"
	"oras.land/oras/internal/crypto"
	onet "oras.land/oras/internal/net"
	"oras.land/oras/internal/trace"
	"oras.land/oras/internal/version"
	"strconv"
	"strings"
)

// RemoteProperties contains all the attributes to access one registry.
type RemoteProperties struct {
	RawReference string
	Repository   *remote.Repository
	Reference    string
	Path         string

	CACertFilePath  string
	CertFilePath    string
	KeyFilePath     string
	Insecure        bool
	Configs         []string
	Username        string
	SecretFromStdin bool
	Secret          string

	ResolveFlag       []string
	HeaderFlags       []string
	PlainHTTP         bool
	PlainHTTPEnforced bool
}

// IsPlainHttp returns the plain http flag for a given registry.
func (prop *RemoteProperties) IsPlainHttp(registry string) bool {
	if prop.PlainHTTPEnforced {
		return prop.PlainHTTP
	}
	host, _, _ := net.SplitHostPort(registry)
	if host == "localhost" || registry == "localhost" {
		// not specified, defaults to plain http for localhost
		return true
	}
	return prop.PlainHTTP
}

// tlsConfig assembles the tls config.
func (prop *RemoteProperties) tlsConfig() (*tls.Config, error) {
	config := &tls.Config{
		InsecureSkipVerify: prop.Insecure,
	}
	if prop.CACertFilePath != "" {
		var err error
		config.RootCAs, err = crypto.LoadCertPool(prop.CACertFilePath)
		if err != nil {
			return nil, err
		}
	}
	if prop.CertFilePath != "" && prop.KeyFilePath != "" {
		cert, err := tls.LoadX509KeyPair(prop.CertFilePath, prop.KeyFilePath)
		if err != nil {
			return nil, err
		}
		config.Certificates = []tls.Certificate{cert}
	}
	return config, nil
}

type Remote struct {
	RemoteProperties
	store          credentials.Store
	headers        http.Header
	warningHandler *WarningHandler
	debug          bool
}

func NewRemote(rawReference string, logger logrus.FieldLogger, debug bool) Remote {
	return Remote{
		RemoteProperties: RemoteProperties{
			RawReference: rawReference,
		},
		warningHandler: NewWarningHandler(logger),
		debug:          debug,
	}
}

// Parse parses the raw in format of <Path>[:<tag>|@<digest>]
func (r *Remote) Parse() (err error) {
	r.Repository, err = remote.NewRepository(r.RawReference)
	if err != nil {
		return err
	}
	r.Repository.PlainHTTP = r.IsPlainHttp(r.Repository.Reference.Registry)
	r.Repository.SkipReferrersGC = true
	//if r.ReferrersAPI != nil {
	//	if err := repo.SetReferrersCapability(*r.ReferrersAPI); err != nil {
	//		return nil, err
	//	}
	//}

	r.Reference = r.Repository.Reference.Reference
	repo := r.Repository.Reference
	repo.Reference = ""
	r.Path = repo.String()
	return r.parseCustomHeaders()
}

// parseResolve parses resolve flag.
func (r *Remote) parseResolve(baseDial onet.DialFunc) (onet.DialFunc, error) {
	if len(r.ResolveFlag) == 0 {
		return baseDial, nil
	}

	formatError := func(param, message string) error {
		return fmt.Errorf("failed to parse resolve flag %q: %s", param, message)
	}
	var dialer onet.Dialer
	for _, r := range r.ResolveFlag {
		parts := strings.SplitN(r, ":", 4)
		length := len(parts)
		if length < 3 {
			return nil, formatError(r, "expecting host:port:address[:address_port]")
		}
		host := parts[0]
		hostPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, formatError(r, "expecting uint64 host port")
		}
		// ipv6 zone is not parsed
		address := net.ParseIP(parts[2])
		if address == nil {
			return nil, formatError(r, "invalid IP address")
		}
		addressPort := hostPort
		if length > 3 {
			addressPort, err = strconv.Atoi(parts[3])
			if err != nil {
				return nil, formatError(r, "expecting uint64 address port")
			}
		}
		dialer.Add(host, hostPort, address, addressPort)
	}
	dialer.BaseDialContext = baseDial
	return dialer.DialContext, nil
}

func (r *Remote) parseCustomHeaders() error {
	if len(r.HeaderFlags) != 0 {
		headers := map[string][]string{}
		for _, h := range r.HeaderFlags {
			name, value, found := strings.Cut(h, ":")
			if !found || strings.TrimSpace(name) == "" {
				// In conformance to the RFC 2616 specification
				// Reference: https://www.rfc-editor.org/rfc/rfc2616#section-4.2
				return fmt.Errorf("invalid header: %q", h)
			}
			headers[name] = append(headers[name], value)
		}
		r.headers = headers
	}
	return nil
}

// Credential returns a credential based on the remote options.
func (r *Remote) Credential() auth.Credential {
	return credential.Credential(r.Username, r.Secret)
}

// authClient assembles a oras auth client.
func (r *Remote) authClient() (client *auth.Client, err error) {
	config, err := r.tlsConfig()
	if err != nil {
		return nil, err
	}
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	baseTransport.TLSClientConfig = config
	dialContext, err := r.parseResolve(baseTransport.DialContext)
	if err != nil {
		return nil, err
	}
	baseTransport.DialContext = dialContext
	client = &auth.Client{
		Client: &http.Client{
			// http.RoundTripper with a retry using the DefaultPolicy
			// see: https://pkg.go.dev/oras.land/oras-go/v2/registry/remote/retry#Policy
			Transport: retry.NewTransport(baseTransport),
		},
		Cache:  auth.NewCache(),
		Header: r.headers,
	}
	client.SetUserAgent("oras/" + version.GetVersion())
	if r.debug {
		client.Client.Transport = trace.NewTransport(client.Client.Transport)
	}

	cred := r.Credential()
	if cred != auth.EmptyCredential {
		client.Credential = func(ctx context.Context, s string) (auth.Credential, error) {
			return cred, nil
		}
	} else {
		var err error
		r.store, err = credential.NewStore(r.Configs...)
		if err != nil {
			return nil, err
		}
		client.Credential = credentials.Credential(r.store)
	}
	return
}

// NewRegistry assembles a oras remote registry.
func (r *Remote) NewRegistry(registry string, warningHandler *WarningHandler) (reg *remote.Registry, err error) {
	reg, err = remote.NewRegistry(registry)
	if err != nil {
		return nil, err
	}
	registry = reg.Reference.Registry
	reg.PlainHTTP = r.IsPlainHttp(registry)
	reg.HandleWarning = warningHandler.GetHandler(registry)
	if reg.Client, err = r.authClient(); err != nil {
		return nil, err
	}
	return
}

// NewRepository assembles a remote repository.
func (r *Remote) NewRepository() (_ *remote.Repository, err error) {
	r.Repository.HandleWarning = r.warningHandler.GetHandler(r.Repository.Reference.Registry)
	if r.Repository.Client, err = r.authClient(); err != nil {
		return nil, err
	}
	return r.Repository, nil
}
