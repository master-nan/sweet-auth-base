package integration

import (
	"backend/internal/audit"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type staticHostResolver map[string][]net.IP

func (r staticHostResolver) LookupIP(_ context.Context, host string) ([]net.IP, error) {
	values, ok := r[host]
	if !ok {
		return nil, errors.New("host not found")
	}
	return append([]net.IP(nil), values...), nil
}

type sequenceHostResolver struct {
	values [][]net.IP
	mu     sync.Mutex
	calls  int
}

func (r *sequenceHostResolver) LookupIP(_ context.Context, _ string) ([]net.IP, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	index := r.calls
	r.calls++
	if index >= len(r.values) {
		index = len(r.values) - 1
	}
	return append([]net.IP(nil), r.values[index]...), nil
}

type rewriteRoundTripper struct {
	target *url.URL
	base   http.RoundTripper
	mu     sync.Mutex
	seen   []*http.Request
}

func (r *rewriteRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	copyRequest := request.Clone(request.Context())
	copyURL := *request.URL
	copyURL.Scheme = r.target.Scheme
	copyURL.Host = r.target.Host
	copyRequest.URL = &copyURL
	copyRequest.Host = ""
	r.mu.Lock()
	r.seen = append(r.seen, request.Clone(request.Context()))
	r.mu.Unlock()
	return r.base.RoundTrip(copyRequest)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func newTestTransportClient(t *testing.T, allowHTTP bool, handler http.Handler) (*HTTPTransportClient, *rewriteRoundTripper, func()) {
	t.Helper()
	var server *httptest.Server
	if allowHTTP {
		server = httptest.NewServer(handler)
	} else {
		server = httptest.NewTLSServer(handler)
	}
	target, err := url.Parse(server.URL)
	if err != nil {
		server.Close()
		t.Fatalf("parse server URL: %v", err)
	}
	policy, err := NewEndpointPolicy(allowHTTP, nil, staticHostResolver{
		"api.integration.test": {net.ParseIP("93.184.216.34")},
	})
	if err != nil {
		server.Close()
		t.Fatalf("new policy: %v", err)
	}
	rewriter := &rewriteRoundTripper{target: target, base: server.Client().Transport}
	client, err := NewHTTPTransportClient(policy, TransportClientOptions{RoundTripper: rewriter})
	if err != nil {
		server.Close()
		t.Fatalf("new client: %v", err)
	}
	return client, rewriter, server.Close
}

func newRequest(t *testing.T, method string, relativePath string, configure func(*TransportRequestInput)) TransportRequest {
	t.Helper()
	input := TransportRequestInput{
		Method:       method,
		BaseURL:      "https://api.integration.test/base",
		RelativePath: relativePath,
	}
	if configure != nil {
		configure(&input)
	}
	request, err := NewTransportRequest(input)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return request
}

func assertTransportCategory(t *testing.T, err error, want TransportErrorCategory) {
	t.Helper()
	var transportErr *TransportError
	if !errors.As(err, &transportErr) || transportErr.Category() != want {
		t.Fatalf("expected category %q, got %v", want, err)
	}
}

func TestHTTPTransportClientExecutesHTTPSRequestWithSafeEncoding(t *testing.T) {
	client, _, closeServer := newTestTransportClient(t, false, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/base/orders/A B" {
			t.Errorf("unexpected path %q", request.URL.Path)
		}
		if request.URL.Query().Get("keyword") != "a&b" || strings.Join(request.URL.Query()["multi"], ",") != "first,second" {
			t.Errorf("query was not standard encoded: %s", request.URL.RawQuery)
		}
		if request.Header.Get("X-Request-ID") != "request-123" {
			t.Errorf("missing controlled header")
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.Header().Set("Set-Cookie", "secret-cookie")
		writer.Header().Set("Authorization", "secret-response-authentication")
		writer.Header().Set("X-Request-ID", "remote-request")
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer closeServer()

	request := newRequest(t, http.MethodGet, "/orders/{orderID}", func(input *TransportRequestInput) {
		input.PathParameters = map[string]string{"orderID": "A B"}
		input.QueryParameters = map[string][]string{"keyword": {"a&b"}, "multi": {"first", "second"}}
		input.Headers = map[string]string{"X-Request-ID": "request-123"}
	})
	result, err := client.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.StatusCode != http.StatusOK || result.ContentType != "application/json" || !result.CompleteResponse || result.Determinacy != TransportDeterminacyConfirmed {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.ResponseHash != responseHash([]byte(`{"ok":true}`)) || string(result.Body()) != `{"ok":true}` {
		t.Fatalf("unexpected response body or hash")
	}
	headers := result.ResponseHeaders()
	if headers["x-request-id"] != "remote-request" || headers["set-cookie"] != "" || headers["authorization"] != "" {
		t.Fatalf("response headers leaked or omitted incorrectly: %#v", headers)
	}
	headers["x-request-id"] = "changed"
	if result.ResponseHeaders()["x-request-id"] != "remote-request" {
		t.Fatal("response header accessor leaked internal map")
	}
}

func TestHTTPTransportClientAllowsHTTPOnlyWhenPolicyExplicitlyEnablesIt(t *testing.T) {
	client, _, closeServer := newTestTransportClient(t, true, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer closeServer()
	request := newRequest(t, http.MethodGet, "/health", nil)
	request.baseURL = "http://api.integration.test"
	if _, err := client.Execute(context.Background(), request); err != nil {
		t.Fatalf("explicit development HTTP should work: %v", err)
	}

	strictPolicy, err := NewEndpointPolicy(false, nil, staticHostResolver{"api.integration.test": {net.ParseIP("93.184.216.34")}})
	if err != nil {
		t.Fatal(err)
	}
	strictClient, err := NewHTTPTransportClient(strictPolicy, TransportClientOptions{RoundTripper: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("strict policy must reject before RoundTrip")
		return nil, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	_, err = strictClient.Execute(context.Background(), request)
	assertTransportCategory(t, err, TransportErrorSSRFRejected)
}

func TestTransportRequestRejectsUnsafeInput(t *testing.T) {
	cases := []struct {
		name  string
		input TransportRequestInput
		want  TransportErrorCategory
	}{
		{name: "method", input: TransportRequestInput{Method: http.MethodHead, BaseURL: "https://api.integration.test", RelativePath: "/x"}, want: TransportErrorInvalidConfig},
		{name: "complete URL", input: TransportRequestInput{Method: http.MethodGet, BaseURL: "https://api.integration.test", RelativePath: "https://other.test/x"}, want: TransportErrorInvalidURL},
		{name: "path escape", input: TransportRequestInput{Method: http.MethodGet, BaseURL: "https://api.integration.test", RelativePath: "/x/../secret"}, want: TransportErrorInvalidURL},
		{name: "double slash", input: TransportRequestInput{Method: http.MethodGet, BaseURL: "https://api.integration.test", RelativePath: "/x//secret"}, want: TransportErrorInvalidURL},
		{name: "userinfo", input: TransportRequestInput{Method: http.MethodGet, BaseURL: "https://user@api.integration.test", RelativePath: "/x"}, want: TransportErrorInvalidURL},
		{name: "auth header", input: TransportRequestInput{Method: http.MethodGet, BaseURL: "https://api.integration.test", RelativePath: "/x", Headers: map[string]string{"Authorization": "Bearer secret"}}, want: TransportErrorInvalidConfig},
		{name: "host header", input: TransportRequestInput{Method: http.MethodGet, BaseURL: "https://api.integration.test", RelativePath: "/x", Headers: map[string]string{"Host": "other.test"}}, want: TransportErrorInvalidConfig},
		{name: "GET body", input: TransportRequestInput{Method: http.MethodGet, BaseURL: "https://api.integration.test", RelativePath: "/x", JSONBody: []byte(`{}`)}, want: TransportErrorInvalidConfig},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewTransportRequest(testCase.input)
			assertTransportCategory(t, err, testCase.want)
		})
	}
}

func TestEndpointPolicyRejectsUnsafeAddressesAndDNSRebinding(t *testing.T) {
	unsafe := []string{"127.0.0.1", "169.254.1.1", "169.254.169.254", "100.100.100.200", "10.0.0.8"}
	policy := DefaultEndpointPolicy()
	for _, raw := range unsafe {
		t.Run(raw, func(t *testing.T) {
			target, _ := url.Parse("https://" + raw + "/")
			_, err := policy.validateTarget(context.Background(), target)
			assertTransportCategory(t, err, TransportErrorSSRFRejected)
		})
	}
	approvedPolicy, err := NewEndpointPolicy(false, []string{"10.0.0.0/8", "127.0.0.0/8"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := url.Parse("https://10.0.0.8/")
	if _, err = approvedPolicy.validateTarget(context.Background(), target); err != nil {
		t.Fatalf("explicitly approved private address should pass: %v", err)
	}
	loopbackTarget, _ := url.Parse("https://127.0.0.1/")
	_, err = approvedPolicy.validateTarget(context.Background(), loopbackTarget)
	assertTransportCategory(t, err, TransportErrorSSRFRejected)

	resolver := &sequenceHostResolver{values: [][]net.IP{{net.ParseIP("93.184.216.34")}, {net.ParseIP("10.0.0.8")}}}
	policy, err = NewEndpointPolicy(false, nil, resolver)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewHTTPTransportClient(policy, TransportClientOptions{})
	if err != nil {
		t.Fatal(err)
	}
	target, _ = url.Parse("https://api.integration.test/")
	if _, err = policy.validateTarget(context.Background(), target); err != nil {
		t.Fatalf("first safe resolution should pass: %v", err)
	}
	_, err = client.safeDialContext(time.Second)(context.Background(), "tcp", "api.integration.test:443")
	assertTransportCategory(t, err, TransportErrorSSRFRejected)
}

func TestHTTPTransportClientRejectsRedirectAndInvalidResolvedAddress(t *testing.T) {
	client, _, closeServer := newTestTransportClient(t, false, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Location", "https://other.test/path")
		writer.WriteHeader(http.StatusFound)
	}))
	defer closeServer()
	_, err := client.Execute(context.Background(), newRequest(t, http.MethodGet, "/redirect", nil))
	assertTransportCategory(t, err, TransportErrorRedirectRejected)

	policy, err := NewEndpointPolicy(false, nil, staticHostResolver{"api.integration.test": {net.ParseIP("10.0.0.8")}})
	if err != nil {
		t.Fatal(err)
	}
	called := false
	blockedClient, err := NewHTTPTransportClient(policy, TransportClientOptions{RoundTripper: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	_, err = blockedClient.Execute(context.Background(), newRequest(t, http.MethodGet, "/blocked", nil))
	assertTransportCategory(t, err, TransportErrorSSRFRejected)
	if called {
		t.Fatal("private DNS answer reached RoundTripper")
	}
}

func TestHTTPTransportClientControlsJSONBodyAndResponseLimits(t *testing.T) {
	client, _, closeServer := newTestTransportClient(t, false, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		if request.Method != http.MethodPost || request.Header.Get("Content-Type") != "application/json" || string(body) != `{"name":"demo"}` {
			t.Errorf("unexpected JSON request %s %q", request.Method, string(body))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"accepted":true}`))
	}))
	defer closeServer()
	request := newRequest(t, http.MethodPost, "/orders", func(input *TransportRequestInput) {
		input.JSONBody = []byte(`{"name":"demo"}`)
	})
	if _, err := client.Execute(context.Background(), request); err != nil {
		t.Fatalf("JSON execution failed: %v", err)
	}

	policy, _ := NewEndpointPolicy(false, nil, staticHostResolver{"api.integration.test": {net.ParseIP("93.184.216.34")}})
	oversizeClient, _ := NewHTTPTransportClient(policy, TransportClientOptions{RoundTripper: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}, "Content-Length": {"2048"}}, ContentLength: 2048, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})})
	request = newRequest(t, http.MethodGet, "/large", func(input *TransportRequestInput) { input.MaxResponseBytes = 1024 })
	result, err := oversizeClient.Execute(context.Background(), request)
	assertTransportCategory(t, err, TransportErrorResponseTooLarge)
	if result.ResponseSize != 2048 || result.CompleteResponse || result.Determinacy != TransportDeterminacyUnknown {
		t.Fatalf("declared oversize result was unsafe: %+v", result)
	}

	streamClient, _ := NewHTTPTransportClient(policy, TransportClientOptions{RoundTripper: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, ContentLength: -1, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", 2048)))}, nil
	})})
	result, err = streamClient.Execute(context.Background(), request)
	assertTransportCategory(t, err, TransportErrorResponseTooLarge)
	if result.CompleteResponse || len(result.Body()) != 0 {
		t.Fatalf("stream over limit must not expose partial response: %+v", result)
	}
}

func TestTransportAuthenticationIsSeparatedFromRegularHeaders(t *testing.T) {
	client, _, closeServer := newTestTransportClient(t, false, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer prepared-token" {
			t.Errorf("prepared authentication was not injected")
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer closeServer()
	authentication, err := NewTransportAuthentication(map[string]string{"Authorization": "Bearer prepared-token"})
	if err != nil {
		t.Fatal(err)
	}
	request, err := newRequest(t, http.MethodGet, "/auth", nil).WithAuthentication(authentication)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Execute(context.Background(), request); err != nil {
		t.Fatalf("execute authenticated request: %v", err)
	}
	_, err = NewTransportAuthentication(map[string]string{"Cookie": "not allowed"})
	assertTransportCategory(t, err, TransportErrorInvalidConfig)
}

func TestTransportAuthenticationCannotLeakThroughFormattingOrJSON(t *testing.T) {
	authentication, err := NewTransportAuthentication(map[string]string{
		"Authorization": "Bearer transport-authentication-secret",
	})
	if err != nil {
		t.Fatalf("new authentication: %v", err)
	}
	for _, formatted := range []string{
		fmt.Sprint(authentication),
		fmt.Sprintf("%#v", authentication),
	} {
		if strings.Contains(formatted, "transport-authentication-secret") || strings.Contains(strings.ToLower(formatted), "bearer") {
			t.Fatalf("authentication formatting leaked secret: %s", formatted)
		}
	}
	if encoded, marshalErr := json.Marshal(authentication); marshalErr == nil || len(encoded) != 0 {
		t.Fatalf("authentication JSON must be rejected: encoded=%q err=%v", encoded, marshalErr)
	}
}

func TestHTTPTransportClientClassifiesResponseTimeoutCancellationAndRemoteError(t *testing.T) {
	policy, _ := NewEndpointPolicy(false, nil, staticHostResolver{"api.integration.test": {net.ParseIP("93.184.216.34")}})
	unsupportedClient, _ := NewHTTPTransportClient(policy, TransportClientOptions{RoundTripper: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/html"}}, ContentLength: 13, Body: io.NopCloser(strings.NewReader("not supported"))}, nil
	})})
	_, err := unsupportedClient.Execute(context.Background(), newRequest(t, http.MethodGet, "/type", nil))
	assertTransportCategory(t, err, TransportErrorUnsupportedContentType)

	timeoutClient, _ := NewHTTPTransportClient(policy, TransportClientOptions{RoundTripper: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})})
	request := newRequest(t, http.MethodGet, "/timeout", func(input *TransportRequestInput) { input.Timeouts.Request = time.Millisecond })
	result, err := timeoutClient.Execute(context.Background(), request)
	assertTransportCategory(t, err, TransportErrorTimeout)
	if result.Determinacy != TransportDeterminacyUnknown {
		t.Fatal("timeout must not claim remote result is known")
	}

	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = timeoutClient.Execute(cancelledContext, newRequest(t, http.MethodGet, "/cancel", nil))
	assertTransportCategory(t, err, TransportErrorCancelled)
	if result.Determinacy != TransportDeterminacyUnknown {
		t.Fatal("cancelled request must not claim remote result is known")
	}

	remoteClient, _ := NewHTTPTransportClient(policy, TransportClientOptions{RoundTripper: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{"Content-Type": {"application/json"}}, ContentLength: 17, Body: io.NopCloser(strings.NewReader(`{"error":"remote"}`))}, nil
	})})
	result, err = remoteClient.Execute(context.Background(), newRequest(t, http.MethodGet, "/remote", nil))
	if err != nil || result.ErrorCategory != TransportErrorRemoteHTTP || result.StatusCode != http.StatusBadGateway || !result.CompleteResponse {
		t.Fatalf("remote HTTP result should remain structured: result=%+v err=%v", result, err)
	}
	tlsClient, _ := NewHTTPTransportClient(policy, TransportClientOptions{RoundTripper: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, tls.RecordHeaderError{}
	})})
	_, err = tlsClient.Execute(context.Background(), newRequest(t, http.MethodGet, "/tls", nil))
	assertTransportCategory(t, err, TransportErrorTLS)
}

func TestTransportRequestAndClientAreSafeForConcurrentUse(t *testing.T) {
	client, _, closeServer := newTestTransportClient(t, false, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		if request.URL.Path != "/items/before" || request.URL.Query().Get("name") != "before" ||
			request.Header.Get("X-Request-ID") != "before" || string(body) != `{"value":"before"}` {
			t.Errorf("request input changed after construction: path=%s query=%s header=%s body=%s", request.URL.Path, request.URL.RawQuery, request.Header.Get("X-Request-ID"), string(body))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer closeServer()
	input := TransportRequestInput{
		Method: http.MethodPost, BaseURL: "https://api.integration.test", RelativePath: "/items/{id}",
		PathParameters: map[string]string{"id": "before"}, QueryParameters: map[string][]string{"name": {"before"}},
		Headers: map[string]string{"X-Request-ID": "before"}, JSONBody: []byte(`{"value":"before"}`),
	}
	request, err := NewTransportRequest(input)
	if err != nil {
		t.Fatal(err)
	}
	input.PathParameters["id"] = "after"
	input.QueryParameters["name"][0] = "after"
	input.Headers["X-Request-ID"] = "after"
	input.JSONBody = bytes.Repeat([]byte("x"), len(input.JSONBody))

	var group sync.WaitGroup
	for index := 0; index < 24; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			result, executeErr := client.Execute(context.Background(), request)
			if executeErr != nil || string(result.Body()) != "{}" {
				t.Errorf("concurrent execute failed: %v", executeErr)
			}
		}()
	}
	group.Wait()
}

func TestHTTPTransportClientLogsOnlySafeSummary(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	restoreLogger := zap.ReplaceGlobals(zap.New(core))
	defer restoreLogger()
	client, _, closeServer := newTestTransportClient(t, false, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"token":"response-secret"}`))
	}))
	defer closeServer()
	authentication, err := NewTransportAuthentication(map[string]string{"Authorization": "Bearer authorization-secret"})
	if err != nil {
		t.Fatal(err)
	}
	request, err := newRequest(t, http.MethodPost, "/safe", func(input *TransportRequestInput) {
		input.QueryParameters = map[string][]string{"access_token": {"query-secret"}}
		input.JSONBody = []byte(`{"password":"body-secret"}`)
	}).WithAuthentication(authentication)
	if err != nil {
		t.Fatal(err)
	}
	ctx := audit.WithCorrelationIDs(context.Background(), audit.CorrelationIDs{RequestID: "request-safe", TraceID: "trace-safe"})
	if _, err = client.Execute(ctx, request); err != nil {
		t.Fatal(err)
	}
	entries := observed.FilterMessage("integration transport completed").All()
	if len(entries) != 1 {
		t.Fatalf("expected one transport log entry, got %d", len(entries))
	}
	text := fmt.Sprint(entries[0].ContextMap())
	for _, secret := range []string{"authorization-secret", "query-secret", "body-secret", "response-secret"} {
		if strings.Contains(text, secret) {
			t.Fatalf("transport log leaked %q: %s", secret, text)
		}
	}
	if !strings.Contains(text, "request-safe") || !strings.Contains(text, "trace-safe") || !strings.Contains(text, "target_host_hash") {
		t.Fatalf("transport log missing required safe diagnostics: %s", text)
	}
}
