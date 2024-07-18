package version2

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/google/go-cmp/cmp"
)

func TestContainsSubstring(t *testing.T) {
	t.Parallel()

	tmpl := newContainsTemplate(t)
	testCases := []struct {
		InputString string
		Substring   string
		expected    string
	}{
		{InputString: "foo", Substring: "foo", expected: "true"},
		{InputString: "foobar", Substring: "foo", expected: "true"},
		{InputString: "foo", Substring: "", expected: "true"},
		{InputString: "foo", Substring: "bar", expected: "false"},
		{InputString: "foo", Substring: "foobar", expected: "false"},
		{InputString: "", Substring: "foo", expected: "false"},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, tc)
		if err != nil {
			t.Fatalf("Failed to execute the template %v", err)
		}
		if buf.String() != tc.expected {
			t.Errorf("Template generated wrong config, got %v but expected %v.", buf.String(), tc.expected)
		}
	}
}

func TestHasPrefix(t *testing.T) {
	t.Parallel()

	tmpl := newHasPrefixTemplate(t)
	testCases := []struct {
		InputString string
		Prefix      string
		expected    string
	}{
		{InputString: "foo", Prefix: "foo", expected: "true"},
		{InputString: "foo", Prefix: "f", expected: "true"},
		{InputString: "foo", Prefix: "", expected: "true"},
		{InputString: "foo", Prefix: "oo", expected: "false"},
		{InputString: "foo", Prefix: "bar", expected: "false"},
		{InputString: "foo", Prefix: "foobar", expected: "false"},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, tc)
		if err != nil {
			t.Fatalf("Failed to execute the template %v", err)
		}
		if buf.String() != tc.expected {
			t.Errorf("Template generated wrong config, got %v but expected %v.", buf.String(), tc.expected)
		}
	}
}

func TestHasSuffix(t *testing.T) {
	t.Parallel()

	tmpl := newHasSuffixTemplate(t)
	testCases := []struct {
		InputString string
		Suffix      string
		expected    string
	}{
		{InputString: "bar", Suffix: "bar", expected: "true"},
		{InputString: "bar", Suffix: "r", expected: "true"},
		{InputString: "bar", Suffix: "", expected: "true"},
		{InputString: "bar", Suffix: "ba", expected: "false"},
		{InputString: "bar", Suffix: "foo", expected: "false"},
		{InputString: "bar", Suffix: "foobar", expected: "false"},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, tc)
		if err != nil {
			t.Fatalf("Failed to execute the template %v", err)
		}
		if buf.String() != tc.expected {
			t.Errorf("Template generated wrong config, got %v but expected %v.", buf.String(), tc.expected)
		}
	}
}

func TestToLowerInputString(t *testing.T) {
	t.Parallel()

	tmpl := newToLowerTemplate(t)
	testCases := []struct {
		InputString string
		expected    string
	}{
		{InputString: "foobar", expected: "foobar"},
		{InputString: "FOOBAR", expected: "foobar"},
		{InputString: "fOoBaR", expected: "foobar"},
		{InputString: "", expected: ""},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, tc)
		if err != nil {
			t.Fatalf("Failed to execute the template %v", err)
		}
		if buf.String() != tc.expected {
			t.Errorf("Template generated wrong config, got %v but expected %v.", buf.String(), tc.expected)
		}
	}
}

func TestToUpperInputString(t *testing.T) {
	t.Parallel()

	tmpl := newToUpperTemplate(t)
	testCases := []struct {
		InputString string
		expected    string
	}{
		{InputString: "foobar", expected: "FOOBAR"},
		{InputString: "FOOBAR", expected: "FOOBAR"},
		{InputString: "fOoBaR", expected: "FOOBAR"},
		{InputString: "", expected: ""},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, tc)
		if err != nil {
			t.Fatalf("Failed to execute the template %v", err)
		}
		if buf.String() != tc.expected {
			t.Errorf("Template generated wrong config, got %v but expected %v.", buf.String(), tc.expected)
		}
	}
}

func TestMakeHTTPListener(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		server   Server
		expected string
	}{
		{server: Server{
			CustomListeners: false,
			DisableIPV6:     true,
			ProxyProtocol:   false,
		}, expected: "listen 80;\n"},
		{server: Server{
			CustomListeners: false,
			DisableIPV6:     false,
			ProxyProtocol:   false,
		}, expected: "listen 80;\n    listen [::]:80;\n"},
		{server: Server{
			CustomListeners: false,
			DisableIPV6:     true,
			ProxyProtocol:   true,
		}, expected: "listen 80 proxy_protocol;\n"},
		{server: Server{
			CustomListeners: false,
			DisableIPV6:     false,
			ProxyProtocol:   true,
		}, expected: "listen 80 proxy_protocol;\n    listen [::]:80 proxy_protocol;\n"},
		{server: Server{
			CustomListeners: true,
			HTTPPort:        81,
			DisableIPV6:     true,
			ProxyProtocol:   false,
		}, expected: "listen 81;\n"},
		{server: Server{
			CustomListeners: true,
			HTTPPort:        81,
			DisableIPV6:     false,
			ProxyProtocol:   false,
		}, expected: "listen 81;\n    listen [::]:81;\n"},
		{server: Server{
			CustomListeners: true,
			HTTPPort:        81,
			DisableIPV6:     true,
			ProxyProtocol:   true,
		}, expected: "listen 81 proxy_protocol;\n"},
		{server: Server{
			CustomListeners: true,
			HTTPPort:        81,
			DisableIPV6:     false,
			ProxyProtocol:   true,
		}, expected: "listen 81 proxy_protocol;\n    listen [::]:81 proxy_protocol;\n"},
	}

	for _, tc := range testCases {
		got := makeHTTPListener(tc.server)
		if got != tc.expected {
			t.Errorf("Function generated wrong config, got %v but expected %v.", got, tc.expected)
		}
	}
}

func TestMakeHTTPSListener(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		server   Server
		expected string
	}{
		{server: Server{
			CustomListeners: false,
			DisableIPV6:     true,
			ProxyProtocol:   false,
		}, expected: "listen 443 ssl;\n"},
		{server: Server{
			CustomListeners: false,
			DisableIPV6:     false,
			ProxyProtocol:   false,
		}, expected: "listen 443 ssl;\n    listen [::]:443 ssl;\n"},
		{server: Server{
			CustomListeners: false,
			DisableIPV6:     true,
			ProxyProtocol:   true,
		}, expected: "listen 443 ssl proxy_protocol;\n"},
		{server: Server{
			CustomListeners: false,
			DisableIPV6:     false,
			ProxyProtocol:   true,
		}, expected: "listen 443 ssl proxy_protocol;\n    listen [::]:443 ssl proxy_protocol;\n"},
		{server: Server{
			CustomListeners: true,
			HTTPSPort:       444,
			DisableIPV6:     true,
			ProxyProtocol:   false,
		}, expected: "listen 444 ssl;\n"},
		{server: Server{
			CustomListeners: true,
			HTTPSPort:       444,
			DisableIPV6:     false,
			ProxyProtocol:   false,
		}, expected: "listen 444 ssl;\n    listen [::]:444 ssl;\n"},
		{server: Server{
			CustomListeners: true,
			HTTPSPort:       444,
			DisableIPV6:     true,
			ProxyProtocol:   true,
		}, expected: "listen 444 ssl proxy_protocol;\n"},
		{server: Server{
			CustomListeners: true,
			HTTPSPort:       444,
			DisableIPV6:     false,
			ProxyProtocol:   true,
		}, expected: "listen 444 ssl proxy_protocol;\n    listen [::]:444 ssl proxy_protocol;\n"},
	}
	for _, tc := range testCases {
		got := makeHTTPSListener(tc.server)
		if got != tc.expected {
			t.Errorf("Function generated wrong config, got %v but expected %v.", got, tc.expected)
		}
	}
}

func newContainsTemplate(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("testTemplate").Funcs(helperFunctions).Parse(`{{contains .InputString .Substring}}`)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	return tmpl
}

func TestReplaceAll(t *testing.T) {
	t.Parallel()

	tmpl := newReplaceAll(t)
	testCases := []struct {
		InputString  string
		OldSubstring string
		NewSubstring string
		expected     string
	}{
		{InputString: "foobarfoo", OldSubstring: "bar", NewSubstring: "foo", expected: "foofoofoo"},
		{InputString: "footest", OldSubstring: "test", NewSubstring: "bar", expected: "foobar"},
		{InputString: "barfoo", OldSubstring: "bar", NewSubstring: "test", expected: "testfoo"},
		{InputString: "foofoofoo", OldSubstring: "foo", NewSubstring: "bar", expected: "barbarbar"},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, tc)
		if err != nil {
			t.Fatalf("Failed to execute the template %v", err)
		}
		if buf.String() != tc.expected {
			t.Errorf("Template generated wrong config, got %v but expected %v.", buf.String(), tc.expected)
		}
	}
}

func TestMakeHeaderQueryValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		apiKey   APIKey
		expected string
	}{
		{
			apiKey: APIKey{
				Header: []string{"foo", "bar"},
			},
			expected: `"${http_foo}${http_bar}"`,
		},
		{
			apiKey: APIKey{
				Header: []string{"foo", "bar"},
				Query:  []string{"baz", "qux"},
			},
			expected: `"${http_foo}${http_bar}${arg_baz}${arg_qux}"`,
		},
		{
			apiKey: APIKey{
				Query: []string{"baz", "qux"},
			},
			expected: `"${arg_baz}${arg_qux}"`,
		},
	}

	for _, tc := range testCases {
		got := makeHeaderQueryValue(tc.apiKey)
		if !cmp.Equal(tc.expected, got) {
			t.Error(cmp.Diff(tc.expected, got))
		}
	}
}

func newHasPrefixTemplate(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("testTemplate").Funcs(helperFunctions).Parse(`{{hasPrefix .InputString .Prefix}}`)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	return tmpl
}

func newHasSuffixTemplate(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("testTemplate").Funcs(helperFunctions).Parse(`{{hasSuffix .InputString .Suffix}}`)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	return tmpl
}

func newToLowerTemplate(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("testTemplate").Funcs(helperFunctions).Parse(`{{toLower .InputString}}`)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	return tmpl
}

func newToUpperTemplate(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("testTemplate").Funcs(helperFunctions).Parse(`{{toUpper .InputString}}`)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	return tmpl
}

func TestMakeSecretPath(t *testing.T) {
	t.Parallel()

	tmpl := newMakeSecretPathTemplate(t)
	testCases := []struct {
		Secret   string
		Path     string
		Variable string
		Enabled  bool
		expected string
	}{
		{
			Secret:   "/etc/nginx/secret/thing.crt",
			Path:     "/etc/nginx/secret",
			Variable: "$secrets_path",
			Enabled:  true,
			expected: "$secrets_path/thing.crt",
		},
		{
			Secret:   "/etc/nginx/secret/thing.crt",
			Path:     "/etc/nginx/secret",
			Variable: "$secrets_path",
			Enabled:  false,
			expected: "/etc/nginx/secret/thing.crt",
		},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, tc)
		if err != nil {
			t.Fatalf("Failed to execute the template %v", err)
		}
		if buf.String() != tc.expected {
			t.Errorf("Template generated wrong config, got '%v' but expected '%v'.", buf.String(), tc.expected)
		}
	}
}

func newMakeSecretPathTemplate(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("testTemplate").Funcs(helperFunctions).Parse(`{{makeSecretPath .Secret .Path .Variable .Enabled}}`)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	return tmpl
}

func newReplaceAll(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("testTemplate").Funcs(helperFunctions).Parse(`{{replaceAll .InputString .OldSubstring .NewSubstring}}`)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	return tmpl
}
