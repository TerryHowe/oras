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
	RawReference     string
	Registry         string
	Repository       string
	Reference        string
	Path             string
	RemoteRepository *remote.Repository

	CACertFilePath  string
	CertFilePath    string
	KeyFilePath     string
	Insecure        bool
	PlainHTTP       bool
	Configs         []string
	Username        string
	SecretFromStdin bool
	Secret          string

	ResolveFlag  []string
	HeaderFlags  []string
	ReferrersAPI string
}

// IsPlainHttp returns the plain http flag for a given registry.
func (prop *RemoteProperties) IsPlainHttp() bool {
	if prop.PlainHTTP {
		return true
	}
	host, _, _ := net.SplitHostPort(prop.Registry)
	if host == "localhost" || prop.Registry == "localhost" {
		// not specified, defaults to plain http for localhost
		return true
	}
	return false
}

func (prop *RemoteProperties) IsReferrersSet() bool {
	return prop.ReferrersAPI == ""
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

// parseResolve parses resolve flag.
func (r *RemoteProperties) parseResolve(baseDial onet.DialFunc) (onet.DialFunc, error) {
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

type Remote struct {
	RemoteProperties
	store          credentials.Store
	headers        http.Header
	warningHandler *WarningHandler
	debug          bool
}

func NewRemote(props RemoteProperties, logger logrus.FieldLogger, debug bool) Remote {
	return Remote{
		RemoteProperties: props,
		warningHandler:   NewWarningHandler(logger),
		debug:            debug,
	}
}

// Parse parses the raw in format of <Path>[:<tag>|@<digest>]
func (r *Remote) Parse() (err error) {
	r.RemoteRepository, err = remote.NewRepository(r.RawReference)
	if err != nil {
		return err
	}
	r.RemoteRepository.PlainHTTP = r.IsPlainHttp()
	r.RemoteRepository.SkipReferrersGC = true
	r.RemoteRepository.HandleWarning = r.warningHandler.GetHandler(r.RemoteRepository.Reference.Registry)
	if r.RemoteRepository.Client, err = r.authClient(); err != nil {
		return err
	}
	//if r.ReferrersAPI != nil {
	//	if err := repo.SetReferrersCapability(*r.ReferrersAPI); err != nil {
	//		return nil, err
	//	}
	//}

	r.Reference = r.RemoteRepository.Reference.Reference
	repo := r.RemoteRepository.Reference
	repo.Reference = ""
	fmt.Printf("repo: %s\n", r.RemoteRepository.Reference.Registry)
	fmt.Printf("repo: %s\n", r.RemoteRepository.Reference.Repository)
	fmt.Printf("repo: %s\n", r.RemoteRepository.Reference.Reference)
	r.Path = repo.String()
	fmt.Printf("r.Path: %s\n", r.Path)
	return r.parseCustomHeaders()
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
func (r *Remote) NewRegistry() (reg *remote.Registry, err error) {
	registry := r.RemoteRepository.Reference.Registry
	reg, err = remote.NewRegistry(registry)
	if err != nil {
		return nil, err
	}
	registry = reg.Reference.Registry
	reg.PlainHTTP = r.IsPlainHttp()
	reg.HandleWarning = r.warningHandler.GetHandler(registry)
	if reg.Client, err = r.authClient(); err != nil {
		return nil, err
	}
	return
}

// GetRemoteRepository assembles a remote repository.
func (r *Remote) GetRemoteRepository() *remote.Repository {
	return r.RemoteRepository
}
