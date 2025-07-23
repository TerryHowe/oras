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

package resource

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"oras.land/oras-go/v2/registry/remote/auth"
)

var ts *httptest.Server
var testRepo = "test-repo"
var testTagList = struct {
	Tags []string `json:"tags"`
}{
	Tags: []string{"tag"},
}

// localhostServerCert is a PEM-encoded TLS cert with SAN IPs
// "127.0.0.1" and "[::1]", expiring at Jan 29 16:00:00 2084 GMT.
// adapted from golang crypto/tls:
// go run generate_cert.go  --rsa-bits 4096 --host 127.0.0.1,::1,oras.land --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
//
//go:embed testdata/localhostServer.crt
var localhostServerCert []byte

// localhostServerKey is the private key for localhostServerCert.
//
//go:embed testdata/localhostServer.key
var localhostServerKey []byte

// localhostClientCert is a PEM-encoded TLS cert with SAN IPs
// "127.0.0.1" and "[::1]", expiring at Jan 29 16:00:00 2084 GMT.
// adapted from golang crypto/tls (added Client Auth usage):
// go run generate_cert.go  --rsa-bits 4096 --host 127.0.0.1,::1,oras.land --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
//
//go:embed testdata/localhostClient.crt
var localhostClientCert []byte

// localhostClientKey is the private key for localhostClientCert.
//
//go:embed testdata/localhostClient.key
var localhostClientKey []byte

func testingKey(s []byte) []byte {
	return bytes.ReplaceAll(s, []byte("TESTING KEY"), []byte("PRIVATE KEY"))
}

func loadTestingTLSConfig() *tls.Config {

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(localhostClientCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{loadTestingCert(localhostServerCert, testingKey(localhostServerKey))},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    clientCertPool,
	}

	return tlsConfig
}

func loadTestingCert(certificate, key []byte) tls.Certificate {
	cert, err := tls.X509KeyPair(certificate, key)
	if err != nil {
		panic(fmt.Sprintf("Unable to load testing certificate: %v", err))
	}

	return cert

}

func TestMain(m *testing.M) {
	// Test server
	ts = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		m := r.Method
		switch {
		case p == "/v2/" && m == "GET":
			w.WriteHeader(http.StatusOK)
		case p == fmt.Sprintf("/v2/%s/tags/list", testRepo) && m == "GET":
			if err := json.NewEncoder(w).Encode(testTagList); err != nil {
				http.Error(w, "error encoding", http.StatusBadRequest)
			}
		}
	}))
	ts.TLS = loadTestingTLSConfig()
	ts.StartTLS()
	defer ts.Close()
	os.Exit(m.Run())
}

func TestRemote_authClient_RawCredential(t *testing.T) {
	password := make([]byte, 12)
	if _, err := rand.Read(password); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := auth.Credential{
		Username: "mocked^^??oras-@@!#",
		Password: base64.StdEncoding.EncodeToString(password),
	}
	remoteResource := Remote{

		RemoteProperties: RemoteProperties{
			RawReference: "hostname:5000/test/test:latest",
			Registry:     "localhost:5000",
			Repository:   "/test/test",
			Reference:    "latest",
			Username:     want.Username,
			Secret:       want.Password,
		},
	}
	client, err := remoteResource.authClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := client.Credential(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Username != want.Username || got.Password != want.Password {
		t.Fatalf("expect: %v, got: %v", want, got)
	}
}

func TestRemote_authClient_skipTlsVerify(t *testing.T) {
	remote := Remote{
		RemoteProperties: RemoteProperties{
			Insecure:   true,
			Registry:   "localhost:5000",
			Repository: "/test/test",
			Reference:  "latest",
		},
	}
	client, err := remote.authClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_authClient_CARoots(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	remo := Remote{
		RemoteProperties: RemoteProperties{
			CACertFilePath: caPath,
		},
	}
	client, err := remo.authClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_authClient_resolve(t *testing.T) {
	URL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("invalid url in test server: %s", ts.URL)
	}

	testHost := "test.unit.oras"
	remo := Remote{
		RemoteProperties: RemoteProperties{
			Registry:    testHost,
			ResolveFlag: []string{fmt.Sprintf("%s:%s:%s", testHost, URL.Port(), URL.Hostname())},
			Insecure:    true,
		},
	}
	client, err := remo.authClient()
	if err != nil {
		t.Fatalf("unexpected error when creating auth client: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("https://%s:%s", testHost, URL.Port()), nil)
	if err != nil {
		t.Fatalf("unexpected error when generating request: %v", err)
	}
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error when sending request: %v", err)
	}
}

func TestRemote_NewRegistry(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	remo := Remote{
		RemoteProperties: RemoteProperties{
			RawReference:   ts.URL,
			CACertFilePath: caPath,
			PlainHTTP:      true,
		},
	}

	reg, err := remo.NewRegistry()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err = reg.Ping(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_NewRepository(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	uri, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	remo := Remote{
		RemoteProperties: RemoteProperties{
			RawReference:   uri.Host + "/" + testRepo,
			CACertFilePath: caPath,
			PlainHTTP:      true,
		},
	}

	repo := remo.GetRemoteRepository()
	if err = repo.Tags(context.Background(), "", func(got []string) error {
		want := []string{"tag"}
		if len(got) != len(testTagList.Tags) || !reflect.DeepEqual(got, want) {
			return fmt.Errorf("expect: %v, got: %v", testTagList.Tags, got)
		}
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_NewRepositoryMTLS(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	clientCertPath := filepath.Join(t.TempDir(), "oras-test-client.pem")
	if err := os.WriteFile(clientCertPath, localhostClientCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	clientKeyPath := filepath.Join(t.TempDir(), "oras-test-client.key")
	if err := os.WriteFile(clientKeyPath, testingKey(localhostClientKey), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	uri, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	remo := Remote{
		RemoteProperties: RemoteProperties{
			RawReference:   uri.Host + "/" + testRepo,
			CACertFilePath: caPath,
			CertFilePath:   clientCertPath,
			KeyFilePath:    clientKeyPath,
			PlainHTTP:      true,
		},
	}

	repo := remo.GetRemoteRepository()
	if err = repo.Tags(context.Background(), "", func(got []string) error {
		want := []string{"tag"}
		if len(got) != len(testTagList.Tags) || !reflect.DeepEqual(got, want) {
			return fmt.Errorf("expect: %v, got: %v", testTagList.Tags, got)
		}
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_NewRepository_Retry(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	retries, count := 3, 0
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if count < retries {
			http.Error(w, "error", http.StatusTooManyRequests)
			return
		}
		err := json.NewEncoder(w).Encode(testTagList)
		if err != nil {
			http.Error(w, "error encoding", http.StatusBadRequest)
		}
	}))
	ts.TLS = loadTestingTLSConfig()
	ts.StartTLS()
	defer ts.Close()

	uri, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	remo := Remote{
		RemoteProperties: RemoteProperties{
			RawReference:   uri.Host + "/" + testRepo,
			CACertFilePath: caPath,
			PlainHTTP:      true,
		},
	}

	repo := remo.GetRemoteRepository()

	if err = repo.Tags(context.Background(), "", func(got []string) error {
		want := []string{"tag"}
		if len(got) != len(testTagList.Tags) || !reflect.DeepEqual(got, want) {
			return fmt.Errorf("expect: %v, got: %v", testTagList.Tags, got)
		}
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != retries {
		t.Errorf("expected %d retries, got %d", retries, count)
	}
}

func TestRemote_IsPlainHttp_Domain(t *testing.T) {
	props := RemoteProperties{
		Registry:  "registry.example.com",
		PlainHTTP: false,
	}

	if props.IsPlainHttp() != false {
		t.Errorf("tls should be %v for registry %s", !props.IsPlainHttp(), props.Registry)
	}

	props.PlainHTTP = true
	if props.IsPlainHttp() != true {
		t.Errorf("tls should be %v for registry %s", !props.IsPlainHttp(), props.Registry)
	}
}

func TestRemote_IsPlainHttp_localhost(t *testing.T) {
	props := RemoteProperties{
		Registry:  "localhost",
		PlainHTTP: false,
	}

	if props.IsPlainHttp() != true {
		t.Errorf("tls should be %v for registry %s", !props.IsPlainHttp(), props.Registry)
	}

	props.PlainHTTP = true
	if props.IsPlainHttp() != true {
		t.Errorf("tls should be %v for registry %s", !props.IsPlainHttp(), props.Registry)
	}
}

func TestRemote_IsPlainHttp_localhost9000(t *testing.T) {
	props := RemoteProperties{
		Registry:  "localhost:9000",
		PlainHTTP: false,
	}

	if props.IsPlainHttp() != true {
		t.Errorf("tls should be %v for registry %s", !props.IsPlainHttp(), props.Registry)
	}

	props.PlainHTTP = true
	if props.IsPlainHttp() != true {
		t.Errorf("tls should be %v for registry %s", !props.IsPlainHttp(), props.Registry)
	}
}

func TestRemote_parseResolve_err(t *testing.T) {
	tests := []struct {
		name string
		remo *RemoteProperties
	}{
		{
			name: "invalid flag",
			remo: &RemoteProperties{ResolveFlag: []string{"this-shouldn't_work"}},
		},
		{
			name: "no host",
			remo: &RemoteProperties{ResolveFlag: []string{":port:address"}},
		},
		{
			name: "no address",
			remo: &RemoteProperties{ResolveFlag: []string{"host:port:"}},
		},
		{
			name: "invalid address",
			remo: &RemoteProperties{ResolveFlag: []string{"host:port:invalid-ip"}},
		},
		{
			name: "no port",
			remo: &RemoteProperties{ResolveFlag: []string{"host::address"}},
		},
		{
			name: "invalid source port",
			remo: &RemoteProperties{ResolveFlag: []string{"host:port:address"}},
		},
		{
			name: "invalid destination port",
			remo: &RemoteProperties{ResolveFlag: []string{"host:443:address:port"}},
		},
		{
			name: "no source port",
			remo: &RemoteProperties{ResolveFlag: []string{"host::address"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remo := &Remote{RemoteProperties: *tt.remo}
			if err := remo.Parse(); err == nil {
				t.Errorf("Expecting error in Remote.parseResolve()")
			}
		})
	}
}

func TestRemote_parseResolve(t *testing.T) {
	tests := []struct {
		name string
		remo *Remote
	}{
		{
			name: "fromHost:fromPort:toIp",
			remo: &Remote{RemoteProperties: RemoteProperties{ResolveFlag: []string{"host:443:0.0.0.0"}}},
		},
		{
			name: "fromHost:fromPort:toIp:toPort",
			remo: &Remote{RemoteProperties: RemoteProperties{ResolveFlag: []string{"host:443:0.0.0.0:5000"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.remo.parseCustomHeaders(); err != nil {
				t.Errorf("Remote.parseResolve() error = %v", err)
			}
		})
	}
}

func TestRemote_parseCustomHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headerFlags []string
		want        http.Header
		wantErr     bool
	}{
		{
			name:        "no custom header is provided",
			headerFlags: []string{},
			want:        nil,
			wantErr:     false,
		},
		{
			name:        "one name-value pair",
			headerFlags: []string{"key:value"},
			want:        map[string][]string{"key": {"value"}},
			wantErr:     false,
		},
		{
			name:        "multiple name-value pairs",
			headerFlags: []string{"key:value", "k:v"},
			want:        map[string][]string{"key": {"value"}, "k": {"v"}},
			wantErr:     false,
		},
		{
			name:        "multiple name-value pairs with commas",
			headerFlags: []string{"key:value,value2,value3", "k:v,v2,v3"},
			want:        map[string][]string{"key": {"value,value2,value3"}, "k": {"v,v2,v3"}},
			wantErr:     false,
		},
		{
			name:        "empty string is a valid value",
			headerFlags: []string{"k:", "key:value,value2,value3"},
			want:        map[string][]string{"k": {""}, "key": {"value,value2,value3"}},
			wantErr:     false,
		},
		{
			name:        "multiple colons are allowed",
			headerFlags: []string{"k::::v,v2,v3", "key:value,value2,value3"},
			want:        map[string][]string{"k": {":::v,v2,v3"}, "key": {"value,value2,value3"}},
			wantErr:     false,
		},
		{
			name:        "name with spaces",
			headerFlags: []string{"bar   :b"},
			want:        map[string][]string{"bar   ": {"b"}},
			wantErr:     false,
		},
		{
			name:        "value with spaces",
			headerFlags: []string{"foo:   a"},
			want:        map[string][]string{"foo": {"   a"}},
			wantErr:     false,
		},
		{
			name:        "repeated pairs",
			headerFlags: []string{"key:value", "key:value"},
			want:        map[string][]string{"key": {"value", "value"}},
			wantErr:     false,
		},
		{
			name:        "repeated name with different values",
			headerFlags: []string{"key:value", "key:value2"},
			want:        map[string][]string{"key": {"value", "value2"}},
			wantErr:     false,
		},
		{
			name:        "one valid header and one invalid header(no pair)",
			headerFlags: []string{"key:value,value2,value3", "vk"},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "one valid header and one invalid header(empty name)",
			headerFlags: []string{":v", "key:value,value2,value3"},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "pure-space name is invalid",
			headerFlags: []string{" :  foo "},
			want:        nil,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remo := &Remote{
				RemoteProperties: RemoteProperties{
					HeaderFlags: tt.headerFlags,
				},
			}
			if err := remo.parseCustomHeaders(); (err != nil) != tt.wantErr {
				t.Errorf("Remote.parseCustomHeaders() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.want, remo.headers) {
				t.Errorf("Remote.parseCustomHeaders() = %v, want %v", remo.headers, tt.want)
			}
		})
	}
}
