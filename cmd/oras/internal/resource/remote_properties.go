package resource

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"

	"oras.land/oras-go/v2/registry/remote"

	"oras.land/oras/internal/crypto"
	onet "oras.land/oras/internal/net"
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
func (prop *RemoteProperties) parseResolve(baseDial onet.DialFunc) (onet.DialFunc, error) {
	if len(prop.ResolveFlag) == 0 {
		return baseDial, nil
	}

	formatError := func(param, message string) error {
		return fmt.Errorf("failed to parse resolve flag %q: %s", param, message)
	}
	var dialer onet.Dialer
	for _, r := range prop.ResolveFlag {
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
