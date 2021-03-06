package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/vault/helper/consts"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/vault"
)

func TestHandler_parseMFAHandler(t *testing.T) {
	var err error
	var expectedMFACreds logical.MFACreds
	req := &logical.Request{
		Headers: make(map[string][]string),
	}

	headerName := textproto.CanonicalMIMEHeaderKey(MFAHeaderName)

	// Set TOTP passcode in the MFA header
	req.Headers[headerName] = []string{
		"my_totp:123456",
		"my_totp:111111",
		"my_second_mfa:hi=hello",
		"my_third_mfa",
	}
	err = parseMFAHeader(req)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that it is being parsed properly
	expectedMFACreds = logical.MFACreds{
		"my_totp": []string{
			"123456",
			"111111",
		},
		"my_second_mfa": []string{
			"hi=hello",
		},
		"my_third_mfa": []string{},
	}
	if !reflect.DeepEqual(expectedMFACreds, req.MFACreds) {
		t.Fatalf("bad: parsed MFACreds; expected: %#v\n actual: %#v\n", expectedMFACreds, req.MFACreds)
	}

	// Split the creds of a method type in different headers and check if they
	// all get merged together
	req.Headers[headerName] = []string{
		"my_mfa:passcode=123456",
		"my_mfa:month=july",
		"my_mfa:day=tuesday",
	}
	err = parseMFAHeader(req)
	if err != nil {
		t.Fatal(err)
	}

	expectedMFACreds = logical.MFACreds{
		"my_mfa": []string{
			"passcode=123456",
			"month=july",
			"day=tuesday",
		},
	}
	if !reflect.DeepEqual(expectedMFACreds, req.MFACreds) {
		t.Fatalf("bad: parsed MFACreds; expected: %#v\n actual: %#v\n", expectedMFACreds, req.MFACreds)
	}

	// Header without method name should error out
	req.Headers[headerName] = []string{
		":passcode=123456",
	}
	err = parseMFAHeader(req)
	if err == nil {
		t.Fatalf("expected an error; actual: %#v\n", req.MFACreds)
	}

	// Header without method name and method value should error out
	req.Headers[headerName] = []string{
		":",
	}
	err = parseMFAHeader(req)
	if err == nil {
		t.Fatalf("expected an error; actual: %#v\n", req.MFACreds)
	}

	// Header without method name and method value should error out
	req.Headers[headerName] = []string{
		"my_totp:",
	}
	err = parseMFAHeader(req)
	if err == nil {
		t.Fatalf("expected an error; actual: %#v\n", req.MFACreds)
	}
}

func TestHandler_cors(t *testing.T) {
	core, _, _ := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()

	// Enable CORS and allow from any origin for testing.
	corsConfig := core.CORSConfig()
	err := corsConfig.Enable(context.Background(), []string{addr}, nil)
	if err != nil {
		t.Fatalf("Error enabling CORS: %s", err)
	}

	req, err := http.NewRequest(http.MethodOptions, addr+"/v1/sys/seal-status", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	req.Header.Set("Origin", "BAD ORIGIN")

	// Requests from unacceptable origins will be rejected with a 403.
	client := cleanhttp.DefaultClient()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Bad status:\nexpected: 403 Forbidden\nactual: %s", resp.Status)
	}

	//
	// Test preflight requests
	//

	// Set a valid origin
	req.Header.Set("Origin", addr)

	// Server should NOT accept arbitrary methods.
	req.Header.Set("Access-Control-Request-Method", "FOO")

	client = cleanhttp.DefaultClient()
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Fail if an arbitrary method is accepted.
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("Bad status:\nexpected: 405 Method Not Allowed\nactual: %s", resp.Status)
	}

	// Server SHOULD accept acceptable methods.
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	client = cleanhttp.DefaultClient()
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	//
	// Test that the CORS headers are applied correctly.
	//
	expHeaders := map[string]string{
		"Access-Control-Allow-Origin":  addr,
		"Access-Control-Allow-Headers": strings.Join(vault.StdAllowedHeaders, ","),
		"Access-Control-Max-Age":       "300",
		"Vary":                         "Origin",
	}

	for expHeader, expected := range expHeaders {
		actual := resp.Header.Get(expHeader)
		if actual == "" {
			t.Fatalf("bad:\nHeader: %#v was not on response.", expHeader)
		}

		if actual != expected {
			t.Fatalf("bad:\nExpected: %#v\nActual: %#v\n", expected, actual)
		}
	}
}

func TestHandler_CacheControlNoStore(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()

	req, err := http.NewRequest("GET", addr+"/v1/sys/mounts", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	req.Header.Set(consts.AuthHeaderName, token)
	req.Header.Set(WrapTTLHeaderName, "60s")

	client := cleanhttp.DefaultClient()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if resp == nil {
		t.Fatalf("nil response")
	}

	actual := resp.Header.Get("Cache-Control")

	if actual == "" {
		t.Fatalf("missing 'Cache-Control' header entry in response writer")
	}

	if actual != "no-store" {
		t.Fatalf("bad: Cache-Control. Expected: 'no-store', Actual: %q", actual)
	}
}

func TestHandler_Accepted(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()

	req, err := http.NewRequest("POST", addr+"/v1/auth/token/tidy", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	req.Header.Set(consts.AuthHeaderName, token)

	client := cleanhttp.DefaultClient()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testResponseStatus(t, resp, 202)
}

// We use this test to verify header auth
func TestSysMounts_headerAuth(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()

	req, err := http.NewRequest("GET", addr+"/v1/sys/mounts", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	req.Header.Set(consts.AuthHeaderName, token)

	client := cleanhttp.DefaultClient()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var actual map[string]interface{}
	expected := map[string]interface{}{
		"lease_id":       "",
		"renewable":      false,
		"lease_duration": json.Number("0"),
		"wrap_info":      nil,
		"warnings":       nil,
		"auth":           nil,
		"data": map[string]interface{}{
			"secret/": map[string]interface{}{
				"description": "key/value secret storage",
				"type":        "kv",
				"config": map[string]interface{}{
					"default_lease_ttl": json.Number("0"),
					"max_lease_ttl":     json.Number("0"),
					"force_no_cache":    false,
					"plugin_name":       "",
				},
				"local":     false,
				"seal_wrap": false,
				"options":   map[string]interface{}{"version": "1"},
			},
			"sys/": map[string]interface{}{
				"description": "system endpoints used for control, policy and debugging",
				"type":        "system",
				"config": map[string]interface{}{
					"default_lease_ttl": json.Number("0"),
					"max_lease_ttl":     json.Number("0"),
					"force_no_cache":    false,
					"plugin_name":       "",
				},
				"local":     false,
				"seal_wrap": false,
				"options":   interface{}(nil),
			},
			"cubbyhole/": map[string]interface{}{
				"description": "per-token private secret storage",
				"type":        "cubbyhole",
				"config": map[string]interface{}{
					"default_lease_ttl": json.Number("0"),
					"max_lease_ttl":     json.Number("0"),
					"force_no_cache":    false,
					"plugin_name":       "",
				},
				"local":     true,
				"seal_wrap": false,
				"options":   interface{}(nil),
			},
			"identity/": map[string]interface{}{
				"description": "identity store",
				"type":        "identity",
				"config": map[string]interface{}{
					"default_lease_ttl": json.Number("0"),
					"max_lease_ttl":     json.Number("0"),
					"force_no_cache":    false,
					"plugin_name":       "",
				},
				"local":     false,
				"seal_wrap": false,
				"options":   interface{}(nil),
			},
		},
		"secret/": map[string]interface{}{
			"description": "key/value secret storage",
			"type":        "kv",
			"config": map[string]interface{}{
				"default_lease_ttl": json.Number("0"),
				"max_lease_ttl":     json.Number("0"),
				"force_no_cache":    false,
				"plugin_name":       "",
			},
			"local":     false,
			"seal_wrap": false,
			"options":   map[string]interface{}{"version": "1"},
		},
		"sys/": map[string]interface{}{
			"description": "system endpoints used for control, policy and debugging",
			"type":        "system",
			"config": map[string]interface{}{
				"default_lease_ttl": json.Number("0"),
				"max_lease_ttl":     json.Number("0"),
				"force_no_cache":    false,
				"plugin_name":       "",
			},
			"local":     false,
			"seal_wrap": false,
			"options":   interface{}(nil),
		},
		"cubbyhole/": map[string]interface{}{
			"description": "per-token private secret storage",
			"type":        "cubbyhole",
			"config": map[string]interface{}{
				"default_lease_ttl": json.Number("0"),
				"max_lease_ttl":     json.Number("0"),
				"force_no_cache":    false,
				"plugin_name":       "",
			},
			"local":     true,
			"seal_wrap": false,
			"options":   interface{}(nil),
		},
		"identity/": map[string]interface{}{
			"description": "identity store",
			"type":        "identity",
			"config": map[string]interface{}{
				"default_lease_ttl": json.Number("0"),
				"max_lease_ttl":     json.Number("0"),
				"force_no_cache":    false,
				"plugin_name":       "",
			},
			"local":     false,
			"seal_wrap": false,
			"options":   interface{}(nil),
		},
	}
	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &actual)

	expected["request_id"] = actual["request_id"]
	for k, v := range actual["data"].(map[string]interface{}) {
		if v.(map[string]interface{})["accessor"] == "" {
			t.Fatalf("no accessor from %s", k)
		}
		expected[k].(map[string]interface{})["accessor"] = v.(map[string]interface{})["accessor"]
		expected["data"].(map[string]interface{})[k].(map[string]interface{})["accessor"] = v.(map[string]interface{})["accessor"]
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad:\nExpected: %#v\nActual: %#v\n", expected, actual)
	}
}

// We use this test to verify header auth wrapping
func TestSysMounts_headerAuth_Wrapped(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()

	req, err := http.NewRequest("GET", addr+"/v1/sys/mounts", nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	req.Header.Set(consts.AuthHeaderName, token)
	req.Header.Set(WrapTTLHeaderName, "60s")

	client := cleanhttp.DefaultClient()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var actual map[string]interface{}
	expected := map[string]interface{}{
		"request_id":     "",
		"lease_id":       "",
		"renewable":      false,
		"lease_duration": json.Number("0"),
		"data":           nil,
		"wrap_info": map[string]interface{}{
			"ttl": json.Number("60"),
		},
		"warnings": nil,
		"auth":     nil,
	}

	testResponseStatus(t, resp, 200)
	testResponseBody(t, resp, &actual)

	actualToken, ok := actual["wrap_info"].(map[string]interface{})["token"]
	if !ok || actualToken == "" {
		t.Fatal("token missing in wrap info")
	}
	expected["wrap_info"].(map[string]interface{})["token"] = actualToken

	actualCreationTime, ok := actual["wrap_info"].(map[string]interface{})["creation_time"]
	if !ok || actualCreationTime == "" {
		t.Fatal("creation_time missing in wrap info")
	}
	expected["wrap_info"].(map[string]interface{})["creation_time"] = actualCreationTime

	actualCreationPath, ok := actual["wrap_info"].(map[string]interface{})["creation_path"]
	if !ok || actualCreationPath == "" {
		t.Fatal("creation_path missing in wrap info")
	}
	expected["wrap_info"].(map[string]interface{})["creation_path"] = actualCreationPath

	actualAccessor, ok := actual["wrap_info"].(map[string]interface{})["accessor"]
	if !ok || actualAccessor == "" {
		t.Fatal("accessor missing in wrap info")
	}
	expected["wrap_info"].(map[string]interface{})["accessor"] = actualAccessor

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad:\nExpected: %#v\nActual: %#v\n%T %T", expected, actual, actual["warnings"], actual["data"])
	}
}

func TestHandler_sealed(t *testing.T) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()

	core.Seal(token)

	resp, err := http.Get(addr + "/v1/secret/foo")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	testResponseStatus(t, resp, 503)
}

func TestHandler_ui_default(t *testing.T) {
	core := vault.TestCoreUI(t, false)
	ln, addr := TestServer(t, core)
	defer ln.Close()

	resp, err := http.Get(addr + "/ui/")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	testResponseStatus(t, resp, 404)
}

func TestHandler_ui_enabled(t *testing.T) {
	core := vault.TestCoreUI(t, true)
	ln, addr := TestServer(t, core)
	defer ln.Close()

	resp, err := http.Get(addr + "/ui/")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	testResponseStatus(t, resp, 200)
}

func TestHandler_error(t *testing.T) {
	w := httptest.NewRecorder()

	respondError(w, 500, errors.New("test Error"))

	if w.Code != 500 {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	// The code inside of the error should override
	// the argument to respondError
	w2 := httptest.NewRecorder()
	e := logical.CodedError(403, "error text")

	respondError(w2, 500, e)

	if w2.Code != 403 {
		t.Fatalf("expected 403, got %d", w2.Code)
	}

	// vault.ErrSealed is a special case
	w3 := httptest.NewRecorder()

	respondError(w3, 400, consts.ErrSealed)

	if w3.Code != 503 {
		t.Fatalf("expected 503, got %d", w3.Code)
	}
}

func TestHandler_nonPrintableChars(t *testing.T) {
	testNonPrintable(t, false)
	testNonPrintable(t, true)
}

func testNonPrintable(t *testing.T, disable bool) {
	core, _, token := vault.TestCoreUnsealed(t)
	ln, addr := TestListener(t)
	props := &vault.HandlerProperties{
		Core:                  core,
		MaxRequestSize:        DefaultMaxRequestSize,
		DisablePrintableCheck: disable,
	}
	TestServerWithListenerAndProperties(t, ln, addr, core, props)
	defer ln.Close()

	req, err := http.NewRequest("PUT", addr+"/v1/cubbyhole/foo\u2028bar", strings.NewReader(`{"zip": "zap"}`))
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	req.Header.Set(consts.AuthHeaderName, token)

	client := cleanhttp.DefaultClient()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if disable {
		testResponseStatus(t, resp, 204)
	} else {
		testResponseStatus(t, resp, 400)
	}
}
