package version1

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

func TestExecuteMainTemplateForNGINXPlus(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfg)
	if err != nil {
		t.Error(err)
	}
	t.Log(buf.String())
}

func TestExecuteMainTemplateForNGINX(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfg)
	if err != nil {
		t.Error(err)
	}
	t.Log(buf.String())
}

func TestExecuteTemplate_ForIngressForNGINXPlus(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfg)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecuteTemplate_ForIngressForNGINX(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfg)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}
}

func TestExecuteTemplate_ForIngressForNGINXPlusWithRegexAnnotationCaseSensitiveModifier(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgWithRegExAnnotationCaseSensitive)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}

	wantLocation := "~ \"^/tea/[A-Z0-9]{3}\""
	if !strings.Contains(buf.String(), wantLocation) {
		t.Errorf("want %q in generated config", wantLocation)
	}
}

func TestExecuteTemplate_ForIngressForNGINXPlusWithRegexAnnotationCaseInsensitiveModifier(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgWithRegExAnnotationCaseInsensitive)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}

	wantLocation := "~* \"^/tea/[A-Z0-9]{3}\""
	if !strings.Contains(buf.String(), wantLocation) {
		t.Errorf("want %q in generated config", wantLocation)
	}
}

func TestExecuteTemplate_ForIngressForNGINXPlusWithRegexAnnotationExactMatchModifier(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgWithRegExAnnotationExactMatch)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}

	wantLocation := "= \"/tea\""
	if !strings.Contains(buf.String(), wantLocation) {
		t.Errorf("want %q in generated config", wantLocation)
	}
}

func TestExecuteTemplate_ForIngressForNGINXPlusWithRegexAnnotationEmpty(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgWithRegExAnnotationEmptyString)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}

	wantLocation := "/tea"
	if !strings.Contains(buf.String(), wantLocation) {
		t.Errorf("want %q in generated config", wantLocation)
	}
}

func TestExecuteTemplate_ForMergeableIngressForNGINXPlus(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgMasterMinionNGINXPlus)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}
	want := "location /coffee {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
	want = "location /tea {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
}

func TestExecuteTemplate_ForMergeableIngressForNGINXPlusWithMasterPathRegex(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgMasterMinionNGINXPlusMasterMinions)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}
	want := "location /coffee {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
	want = "location /tea {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
}

func TestExecuteTemplate_ForMergeableIngressWithOneMinionWithPathRegexAnnotation(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgMasterMinionNGINXPlusMinionWithPathRegexAnnotation)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}
	// Observe location /coffee updated with regex
	want := "location ~* \"^/coffee\" {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
	// Observe location /tea not updated with regex
	want = "location /tea {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
}

func TestExecuteTemplate_ForMergeableIngressWithSecondMinionWithPathRegexAnnotation(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgMasterMinionNGINXPlusSecondMinionWithPathRegexAnnotation)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}
	// Observe location /coffee not updated
	want := "location /coffee {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
	// Observe location /tea updated with regex
	want = "location ~ \"^/tea\" {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
}

func TestExecuteTemplate_ForMergeableIngressForNGINXPlusWithPathRegexAnnotationOnMaster(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgMasterMinionNGINXPlusMasterWithPathRegexAnnotation)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}

	want := "location /coffee {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
	want = "location /tea {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
}

func TestExecuteTemplate_ForMergeableIngressForNGINXPlusWithPathRegexAnnotationOnMasterAndMinions(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgMasterMinionNGINXPlusMasterAndAllMinionsWithPathRegexAnnotation)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}

	want := "location ~* \"^/coffee\""
	if !strings.Contains(buf.String(), want) {
		t.Errorf("did not get %q in generated config", want)
	}
	want = "location ~* \"^/tea\""
	if !strings.Contains(buf.String(), want) {
		t.Errorf("did not get %q in generated config", want)
	}
}

func TestExecuteTemplate_ForMergeableIngressForNGINXPlusWithPathRegexAnnotationOnMinionsNotOnMaster(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgMasterMinionNGINXPlusMasterWithoutPathRegexMinionsWithPathRegexAnnotation)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}

	want := "location ~* \"^/coffee\" {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
	want = "location ~ \"^/tea\" {"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want %q in generated config", want)
	}
}

func TestExecuteTemplate_ForMainForNGINXWithCustomTLSPassthroughPort(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgCustomTLSPassthroughPort)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 8443;",
		"listen [::]:8443;",
		"proxy_pass $dest_internal_passthrough",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXPlusWithCustomTLSPassthroughPort(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgCustomTLSPassthroughPort)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 8443;",
		"listen [::]:8443;",
		"proxy_pass $dest_internal_passthrough",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXWithoutCustomTLSPassthroughPort(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgDefaultTLSPassthroughPort)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 443;",
		"listen [::]:443;",
		"proxy_pass $dest_internal_passthrough",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXPlusWithoutCustomTLSPassthroughPort(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgDefaultTLSPassthroughPort)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 443;",
		"listen [::]:443;",
		"proxy_pass $dest_internal_passthrough",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXTLSPassthroughDisabled(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgWithoutTLSPassthrough)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	unwantDirectives := []string{
		"listen 8443;",
		"listen [::]:8443;",
		"proxy_pass $dest_internal_passthrough",
	}

	mainConf := buf.String()
	for _, want := range unwantDirectives {
		if strings.Contains(mainConf, want) {
			t.Errorf("unwant %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXPlusTLSPassthroughPortDisabled(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgWithoutTLSPassthrough)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	unwantDirectives := []string{
		"listen 443;",
		"listen [::]:443;",
		"proxy_pass $dest_internal_passthrough",
	}

	mainConf := buf.String()
	for _, want := range unwantDirectives {
		if strings.Contains(mainConf, want) {
			t.Errorf("unwant %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXWithCustomDefaultHTTPAndHTTPSListenerPorts(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgCustomDefaultHTTPAndHTTPSListenerPorts)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 8083 default_server;",
		"listen [::]:8083 default_server;",
		"listen 8443 ssl default_server;",
		"listen [::]:8443 ssl default_server;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXPlusWithCustomDefaultHTTPAndHTTPSListenerPorts(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgCustomDefaultHTTPAndHTTPSListenerPorts)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 8083 default_server;",
		"listen [::]:8083 default_server;",
		"listen 8443 ssl default_server;",
		"listen [::]:8443 ssl default_server;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXWithoutCustomDefaultHTTPAndHTTPSListenerPorts(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfg)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 80 default_server;",
		"listen [::]:80 default_server;",
		"listen 443 ssl default_server;",
		"listen [::]:443 ssl default_server;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXPlusWithoutCustomDefaultHTTPAndHTTPSListenerPorts(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfg)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 80 default_server;",
		"listen [::]:80 default_server;",
		"listen 443 ssl default_server;",
		"listen [::]:443 ssl default_server;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXWithCustomDefaultHTTPListenerPort(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgCustomDefaultHTTPListenerPort)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 8083 default_server;",
		"listen [::]:8083 default_server;",
		"listen 443 ssl default_server;",
		"listen [::]:443 ssl default_server;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXWithCustomDefaultHTTPSListenerPort(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgCustomDefaultHTTPSListenerPort)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 80 default_server;",
		"listen [::]:80 default_server;",
		"listen 8443 ssl default_server;",
		"listen [::]:8443 ssl default_server;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXPlusWithCustomDefaultHTTPListenerPort(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgCustomDefaultHTTPListenerPort)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 8083 default_server;",
		"listen [::]:8083 default_server;",
		"listen 443 ssl default_server;",
		"listen [::]:443 ssl default_server;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXPlusWithCustomDefaultHTTPSListenerPort(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgCustomDefaultHTTPSListenerPort)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 80 default_server;",
		"listen [::]:80 default_server;",
		"listen 8443 ssl default_server;",
		"listen [::]:8443 ssl default_server;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXWithHTTP2On(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgHTTP2On)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 443 ssl default_server;",
		"listen [::]:443 ssl default_server;",
		"http2 on;",
	}

	unwantDirectives := []string{
		"listen 443 ssl default_server http2;",
		"listen [::]:443 ssl default_server http2;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}

	for _, want := range unwantDirectives {
		if strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXPlusWithHTTP2On(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfgHTTP2On)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 443 ssl default_server;",
		"listen [::]:443 ssl default_server;",
		"http2 on;",
	}

	unwantDirectives := []string{
		"listen 443 ssl default_server http2;",
		"listen [::]:443 ssl default_server http2;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}

	for _, want := range unwantDirectives {
		if strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXWithHTTP2Off(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfg)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 443 ssl default_server;",
		"listen [::]:443 ssl default_server;",
	}

	unwantDirectives := []string{
		"http2 on;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}

	for _, want := range unwantDirectives {
		if strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForMainForNGINXPlusWithHTTP2Off(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfg)
	t.Log(buf.String())

	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantDirectives := []string{
		"listen 443 ssl default_server;",
		"listen [::]:443 ssl default_server;",
	}

	unwantDirectives := []string{
		"http2 on;",
	}

	mainConf := buf.String()
	for _, want := range wantDirectives {
		if !strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}

	for _, want := range unwantDirectives {
		if strings.Contains(mainConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForIngressForNGINXPlusWithHTTP2On(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgHTTP2On)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}
	ingConf := buf.String()

	wantDirectives := []string{
		"listen 443 ssl;",
		"listen [::]:443 ssl;",
		"http2 on;",
	}

	unwantDirectives := []string{
		"listen 443 ssl http2;",
		"listen [::]:443 ssl http2;",
	}

	for _, want := range wantDirectives {
		if !strings.Contains(ingConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}

	for _, want := range unwantDirectives {
		if strings.Contains(ingConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForIngressForNGINXWithHTTP2On(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgHTTP2On)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}
	ingConf := buf.String()

	wantDirectives := []string{
		"listen 443 ssl;",
		"listen [::]:443 ssl;",
		"http2 on;",
	}

	unwantDirectives := []string{
		"listen 443 ssl http2;",
		"listen [::]:443 ssl http2;",
	}

	for _, want := range wantDirectives {
		if !strings.Contains(ingConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}

	for _, want := range unwantDirectives {
		if strings.Contains(ingConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForIngressForNGINXPlusWithHTTP2Off(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfg)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}
	ingConf := buf.String()

	wantDirectives := []string{
		"listen 443 ssl;",
		"listen [::]:443 ssl;",
	}

	unwantDirectives := []string{
		"http2 on;",
	}

	for _, want := range wantDirectives {
		if !strings.Contains(ingConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}

	for _, want := range unwantDirectives {
		if strings.Contains(ingConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func TestExecuteTemplate_ForIngressForNGINXWithHTTP2Off(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfg)
	t.Log(buf.String())
	if err != nil {
		t.Fatal(err)
	}
	ingConf := buf.String()

	wantDirectives := []string{
		"listen 443 ssl;",
		"listen [::]:443 ssl;",
	}

	unwantDirectives := []string{
		"http2 on;",
	}

	for _, want := range wantDirectives {
		if !strings.Contains(ingConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}

	for _, want := range unwantDirectives {
		if strings.Contains(ingConf, want) {
			t.Errorf("want %q in generated config", want)
		}
	}
}

func newNGINXPlusIngressTmpl(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("nginx-plus.ingress.tmpl").Funcs(helperFunctions).ParseFiles("nginx-plus.ingress.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

func newNGINXIngressTmpl(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("nginx.ingress.tmpl").Funcs(helperFunctions).ParseFiles("nginx.ingress.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

func newNGINXPlusMainTmpl(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("nginx-plus.tmpl").ParseFiles("nginx-plus.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

func newNGINXMainTmpl(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("nginx.tmpl").ParseFiles("nginx.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

var (
	// Ingress Config example without added annotations
	ingressCfg = IngressNginxConfig{
		Servers: []Server{
			{
				Name:         "test.example.com",
				ServerTokens: "off",
				StatusZone:   "test.example.com",
				JWTAuth: &JWTAuth{
					Key:                  "/etc/nginx/secrets/key.jwk",
					Realm:                "closed site",
					Token:                "$cookie_auth_token",
					RedirectLocationName: "@login_url-default-cafe-ingress",
				},
				SSL:               true,
				SSLCertificate:    "secret.pem",
				SSLCertificateKey: "secret.pem",
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				Locations: []Location{
					{
						Path:                "/tea",
						Upstream:            testUpstream,
						ProxyConnectTimeout: "10s",
						ProxyReadTimeout:    "10s",
						ProxySendTimeout:    "10s",
						ClientMaxBodySize:   "2m",
						JWTAuth: &JWTAuth{
							Key:   "/etc/nginx/secrets/location-key.jwk",
							Realm: "closed site",
							Token: "$cookie_auth_token",
						},
						MinionIngress: &Ingress{
							Name:      "tea-minion",
							Namespace: "default",
						},
					},
				},
				HealthChecks: map[string]HealthCheck{"test": healthCheck},
				JWTRedirectLocations: []JWTRedirectLocation{
					{
						Name:     "@login_url-default-cafe-ingress",
						LoginURL: "https://test.example.com/login",
					},
				},
			},
		},
		Upstreams: []Upstream{testUpstream},
		Keepalive: "16",
		Ingress: Ingress{
			Name:      "cafe-ingress",
			Namespace: "default",
		},
	}

	// Ingress Config example with path-regex annotation value "case_sensitive"
	ingressCfgWithRegExAnnotationCaseSensitive = IngressNginxConfig{
		Servers: []Server{
			{
				Name:         "test.example.com",
				ServerTokens: "off",
				StatusZone:   "test.example.com",
				JWTAuth: &JWTAuth{
					Key:                  "/etc/nginx/secrets/key.jwk",
					Realm:                "closed site",
					Token:                "$cookie_auth_token",
					RedirectLocationName: "@login_url-default-cafe-ingress",
				},
				SSL:               true,
				SSLCertificate:    "secret.pem",
				SSLCertificateKey: "secret.pem",
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				Locations: []Location{
					{
						Path:                "/tea/[A-Z0-9]{3}",
						Upstream:            testUpstream,
						ProxyConnectTimeout: "10s",
						ProxyReadTimeout:    "10s",
						ProxySendTimeout:    "10s",
						ClientMaxBodySize:   "2m",
						JWTAuth: &JWTAuth{
							Key:   "/etc/nginx/secrets/location-key.jwk",
							Realm: "closed site",
							Token: "$cookie_auth_token",
						},
						MinionIngress: &Ingress{
							Name:      "tea-minion",
							Namespace: "default",
						},
					},
				},
				HealthChecks: map[string]HealthCheck{"test": healthCheck},
				JWTRedirectLocations: []JWTRedirectLocation{
					{
						Name:     "@login_url-default-cafe-ingress",
						LoginURL: "https://test.example.com/login",
					},
				},
			},
		},
		Upstreams: []Upstream{testUpstream},
		Keepalive: "16",
		Ingress: Ingress{
			Name:        "cafe-ingress",
			Namespace:   "default",
			Annotations: map[string]string{"nginx.org/path-regex": "case_sensitive"},
		},
	}

	// Ingress Config example with path-regex annotation value "case_insensitive"
	ingressCfgWithRegExAnnotationCaseInsensitive = IngressNginxConfig{
		Servers: []Server{
			{
				Name:         "test.example.com",
				ServerTokens: "off",
				StatusZone:   "test.example.com",
				JWTAuth: &JWTAuth{
					Key:                  "/etc/nginx/secrets/key.jwk",
					Realm:                "closed site",
					Token:                "$cookie_auth_token",
					RedirectLocationName: "@login_url-default-cafe-ingress",
				},
				SSL:               true,
				SSLCertificate:    "secret.pem",
				SSLCertificateKey: "secret.pem",
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				Locations: []Location{
					{
						Path:                "/tea/[A-Z0-9]{3}",
						Upstream:            testUpstream,
						ProxyConnectTimeout: "10s",
						ProxyReadTimeout:    "10s",
						ProxySendTimeout:    "10s",
						ClientMaxBodySize:   "2m",
						JWTAuth: &JWTAuth{
							Key:   "/etc/nginx/secrets/location-key.jwk",
							Realm: "closed site",
							Token: "$cookie_auth_token",
						},
						MinionIngress: &Ingress{
							Name:      "tea-minion",
							Namespace: "default",
						},
					},
				},
				HealthChecks: map[string]HealthCheck{"test": healthCheck},
				JWTRedirectLocations: []JWTRedirectLocation{
					{
						Name:     "@login_url-default-cafe-ingress",
						LoginURL: "https://test.example.com/login",
					},
				},
			},
		},
		Upstreams: []Upstream{testUpstream},
		Keepalive: "16",
		Ingress: Ingress{
			Name:        "cafe-ingress",
			Namespace:   "default",
			Annotations: map[string]string{"nginx.org/path-regex": "case_insensitive"},
		},
	}

	// Ingress Config example with path-regex annotation value "exact"
	ingressCfgWithRegExAnnotationExactMatch = IngressNginxConfig{
		Servers: []Server{
			{
				Name:         "test.example.com",
				ServerTokens: "off",
				StatusZone:   "test.example.com",
				JWTAuth: &JWTAuth{
					Key:                  "/etc/nginx/secrets/key.jwk",
					Realm:                "closed site",
					Token:                "$cookie_auth_token",
					RedirectLocationName: "@login_url-default-cafe-ingress",
				},
				SSL:               true,
				SSLCertificate:    "secret.pem",
				SSLCertificateKey: "secret.pem",
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				Locations: []Location{
					{
						Path:                "/tea",
						Upstream:            testUpstream,
						ProxyConnectTimeout: "10s",
						ProxyReadTimeout:    "10s",
						ProxySendTimeout:    "10s",
						ClientMaxBodySize:   "2m",
						JWTAuth: &JWTAuth{
							Key:   "/etc/nginx/secrets/location-key.jwk",
							Realm: "closed site",
							Token: "$cookie_auth_token",
						},
						MinionIngress: &Ingress{
							Name:      "tea-minion",
							Namespace: "default",
						},
					},
				},
				HealthChecks: map[string]HealthCheck{"test": healthCheck},
				JWTRedirectLocations: []JWTRedirectLocation{
					{
						Name:     "@login_url-default-cafe-ingress",
						LoginURL: "https://test.example.com/login",
					},
				},
			},
		},
		Upstreams: []Upstream{testUpstream},
		Keepalive: "16",
		Ingress: Ingress{
			Name:        "cafe-ingress",
			Namespace:   "default",
			Annotations: map[string]string{"nginx.org/path-regex": "exact"},
		},
	}

	// Ingress Config example with path-regex annotation value of an empty string
	ingressCfgWithRegExAnnotationEmptyString = IngressNginxConfig{
		Servers: []Server{
			{
				Name:         "test.example.com",
				ServerTokens: "off",
				StatusZone:   "test.example.com",
				JWTAuth: &JWTAuth{
					Key:                  "/etc/nginx/secrets/key.jwk",
					Realm:                "closed site",
					Token:                "$cookie_auth_token",
					RedirectLocationName: "@login_url-default-cafe-ingress",
				},
				SSL:               true,
				SSLCertificate:    "secret.pem",
				SSLCertificateKey: "secret.pem",
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				Locations: []Location{
					{
						Path:                "/tea",
						Upstream:            testUpstream,
						ProxyConnectTimeout: "10s",
						ProxyReadTimeout:    "10s",
						ProxySendTimeout:    "10s",
						ClientMaxBodySize:   "2m",
						JWTAuth: &JWTAuth{
							Key:   "/etc/nginx/secrets/location-key.jwk",
							Realm: "closed site",
							Token: "$cookie_auth_token",
						},
						MinionIngress: &Ingress{
							Name:      "tea-minion",
							Namespace: "default",
						},
					},
				},
				HealthChecks: map[string]HealthCheck{"test": healthCheck},
				JWTRedirectLocations: []JWTRedirectLocation{
					{
						Name:     "@login_url-default-cafe-ingress",
						LoginURL: "https://test.example.com/login",
					},
				},
			},
		},
		Upstreams: []Upstream{testUpstream},
		Keepalive: "16",
		Ingress: Ingress{
			Name:        "cafe-ingress",
			Namespace:   "default",
			Annotations: map[string]string{"nginx.org/path-regex": ""},
		},
	}

	mainCfg = MainConfig{
		DefaultHTTPListenerPort:  80,
		DefaultHTTPSListenerPort: 443,
		ServerNamesHashMaxSize:   "512",
		ServerTokens:             "off",
		WorkerProcesses:          "auto",
		WorkerCPUAffinity:        "auto",
		WorkerShutdownTimeout:    "1m",
		WorkerConnections:        "1024",
		WorkerRlimitNofile:       "65536",
		LogFormat:                []string{"$remote_addr", "$remote_user"},
		LogFormatEscaping:        "default",
		StreamSnippets:           []string{"# comment"},
		StreamLogFormat:          []string{"$remote_addr", "$remote_user"},
		StreamLogFormatEscaping:  "none",
		ResolverAddresses:        []string{"example.com", "127.0.0.1"},
		ResolverIPV6:             false,
		ResolverValid:            "10s",
		ResolverTimeout:          "15s",
		KeepaliveTimeout:         "65s",
		KeepaliveRequests:        100,
		VariablesHashBucketSize:  256,
		VariablesHashMaxSize:     1024,
	}

	mainCfgHTTP2On = MainConfig{
		DefaultHTTPListenerPort:  80,
		DefaultHTTPSListenerPort: 443,
		HTTP2:                    true,
		ServerNamesHashMaxSize:   "512",
		ServerTokens:             "off",
		WorkerProcesses:          "auto",
		WorkerCPUAffinity:        "auto",
		WorkerShutdownTimeout:    "1m",
		WorkerConnections:        "1024",
		WorkerRlimitNofile:       "65536",
		LogFormat:                []string{"$remote_addr", "$remote_user"},
		LogFormatEscaping:        "default",
		StreamSnippets:           []string{"# comment"},
		StreamLogFormat:          []string{"$remote_addr", "$remote_user"},
		StreamLogFormatEscaping:  "none",
		ResolverAddresses:        []string{"example.com", "127.0.0.1"},
		ResolverIPV6:             false,
		ResolverValid:            "10s",
		ResolverTimeout:          "15s",
		KeepaliveTimeout:         "65s",
		KeepaliveRequests:        100,
		VariablesHashBucketSize:  256,
		VariablesHashMaxSize:     1024,
	}

	mainCfgCustomTLSPassthroughPort = MainConfig{
		ServerNamesHashMaxSize:  "512",
		ServerTokens:            "off",
		WorkerProcesses:         "auto",
		WorkerCPUAffinity:       "auto",
		WorkerShutdownTimeout:   "1m",
		WorkerConnections:       "1024",
		WorkerRlimitNofile:      "65536",
		LogFormat:               []string{"$remote_addr", "$remote_user"},
		LogFormatEscaping:       "default",
		StreamSnippets:          []string{"# comment"},
		StreamLogFormat:         []string{"$remote_addr", "$remote_user"},
		StreamLogFormatEscaping: "none",
		ResolverAddresses:       []string{"example.com", "127.0.0.1"},
		ResolverIPV6:            false,
		ResolverValid:           "10s",
		ResolverTimeout:         "15s",
		KeepaliveTimeout:        "65s",
		KeepaliveRequests:       100,
		VariablesHashBucketSize: 256,
		VariablesHashMaxSize:    1024,
		TLSPassthrough:          true,
		TLSPassthroughPort:      8443,
	}

	mainCfgWithoutTLSPassthrough = MainConfig{
		ServerNamesHashMaxSize:  "512",
		ServerTokens:            "off",
		WorkerProcesses:         "auto",
		WorkerCPUAffinity:       "auto",
		WorkerShutdownTimeout:   "1m",
		WorkerConnections:       "1024",
		WorkerRlimitNofile:      "65536",
		LogFormat:               []string{"$remote_addr", "$remote_user"},
		LogFormatEscaping:       "default",
		StreamSnippets:          []string{"# comment"},
		StreamLogFormat:         []string{"$remote_addr", "$remote_user"},
		StreamLogFormatEscaping: "none",
		ResolverAddresses:       []string{"example.com", "127.0.0.1"},
		ResolverIPV6:            false,
		ResolverValid:           "10s",
		ResolverTimeout:         "15s",
		KeepaliveTimeout:        "65s",
		KeepaliveRequests:       100,
		VariablesHashBucketSize: 256,
		VariablesHashMaxSize:    1024,
		TLSPassthrough:          false,
		TLSPassthroughPort:      8443,
	}

	mainCfgDefaultTLSPassthroughPort = MainConfig{
		ServerNamesHashMaxSize:  "512",
		ServerTokens:            "off",
		WorkerProcesses:         "auto",
		WorkerCPUAffinity:       "auto",
		WorkerShutdownTimeout:   "1m",
		WorkerConnections:       "1024",
		WorkerRlimitNofile:      "65536",
		LogFormat:               []string{"$remote_addr", "$remote_user"},
		LogFormatEscaping:       "default",
		StreamSnippets:          []string{"# comment"},
		StreamLogFormat:         []string{"$remote_addr", "$remote_user"},
		StreamLogFormatEscaping: "none",
		ResolverAddresses:       []string{"example.com", "127.0.0.1"},
		ResolverIPV6:            false,
		ResolverValid:           "10s",
		ResolverTimeout:         "15s",
		KeepaliveTimeout:        "65s",
		KeepaliveRequests:       100,
		VariablesHashBucketSize: 256,
		VariablesHashMaxSize:    1024,
		TLSPassthrough:          true,
		TLSPassthroughPort:      443,
	}

	mainCfgCustomDefaultHTTPAndHTTPSListenerPorts = MainConfig{
		DefaultHTTPListenerPort:  8083,
		DefaultHTTPSListenerPort: 8443,
		ServerNamesHashMaxSize:   "512",
		ServerTokens:             "off",
		WorkerProcesses:          "auto",
		WorkerCPUAffinity:        "auto",
		WorkerShutdownTimeout:    "1m",
		WorkerConnections:        "1024",
		WorkerRlimitNofile:       "65536",
		LogFormat:                []string{"$remote_addr", "$remote_user"},
		LogFormatEscaping:        "default",
		StreamSnippets:           []string{"# comment"},
		StreamLogFormat:          []string{"$remote_addr", "$remote_user"},
		StreamLogFormatEscaping:  "none",
		ResolverAddresses:        []string{"example.com", "127.0.0.1"},
		ResolverIPV6:             false,
		ResolverValid:            "10s",
		ResolverTimeout:          "15s",
		KeepaliveTimeout:         "65s",
		KeepaliveRequests:        100,
		VariablesHashBucketSize:  256,
		VariablesHashMaxSize:     1024,
	}

	mainCfgCustomDefaultHTTPListenerPort = MainConfig{
		DefaultHTTPListenerPort:  8083,
		DefaultHTTPSListenerPort: 443,
		ServerNamesHashMaxSize:   "512",
		ServerTokens:             "off",
		WorkerProcesses:          "auto",
		WorkerCPUAffinity:        "auto",
		WorkerShutdownTimeout:    "1m",
		WorkerConnections:        "1024",
		WorkerRlimitNofile:       "65536",
		LogFormat:                []string{"$remote_addr", "$remote_user"},
		LogFormatEscaping:        "default",
		StreamSnippets:           []string{"# comment"},
		StreamLogFormat:          []string{"$remote_addr", "$remote_user"},
		StreamLogFormatEscaping:  "none",
		ResolverAddresses:        []string{"example.com", "127.0.0.1"},
		ResolverIPV6:             false,
		ResolverValid:            "10s",
		ResolverTimeout:          "15s",
		KeepaliveTimeout:         "65s",
		KeepaliveRequests:        100,
		VariablesHashBucketSize:  256,
		VariablesHashMaxSize:     1024,
	}

	mainCfgCustomDefaultHTTPSListenerPort = MainConfig{
		DefaultHTTPListenerPort:  80,
		DefaultHTTPSListenerPort: 8443,
		ServerNamesHashMaxSize:   "512",
		ServerTokens:             "off",
		WorkerProcesses:          "auto",
		WorkerCPUAffinity:        "auto",
		WorkerShutdownTimeout:    "1m",
		WorkerConnections:        "1024",
		WorkerRlimitNofile:       "65536",
		LogFormat:                []string{"$remote_addr", "$remote_user"},
		LogFormatEscaping:        "default",
		StreamSnippets:           []string{"# comment"},
		StreamLogFormat:          []string{"$remote_addr", "$remote_user"},
		StreamLogFormatEscaping:  "none",
		ResolverAddresses:        []string{"example.com", "127.0.0.1"},
		ResolverIPV6:             false,
		ResolverValid:            "10s",
		ResolverTimeout:          "15s",
		KeepaliveTimeout:         "65s",
		KeepaliveRequests:        100,
		VariablesHashBucketSize:  256,
		VariablesHashMaxSize:     1024,
	}

	// Vars for Mergable Ingress Master - Minion tests

	coffeeUpstreamNginxPlus = Upstream{
		Name:             "default-cafe-ingress-coffee-minion-cafe.example.com-coffee-svc-80",
		LBMethod:         "random two least_conn",
		UpstreamZoneSize: "512k",
		UpstreamServers: []UpstreamServer{
			{
				Address:     "10.0.0.1:80",
				MaxFails:    1,
				MaxConns:    0,
				FailTimeout: "10s",
			},
		},
		UpstreamLabels: UpstreamLabels{
			Service:           "coffee-svc",
			ResourceType:      "ingress",
			ResourceName:      "cafe-ingress-coffee-minion",
			ResourceNamespace: "default",
		},
	}

	teaUpstreamNGINXPlus = Upstream{
		Name:             "default-cafe-ingress-tea-minion-cafe.example.com-tea-svc-80",
		LBMethod:         "random two least_conn",
		UpstreamZoneSize: "512k",
		UpstreamServers: []UpstreamServer{
			{
				Address:     "10.0.0.2:80",
				MaxFails:    1,
				MaxConns:    0,
				FailTimeout: "10s",
			},
		},
		UpstreamLabels: UpstreamLabels{
			Service:           "tea-svc",
			ResourceType:      "ingress",
			ResourceName:      "cafe-ingress-tea-minion",
			ResourceNamespace: "default",
		},
	}

	ingressCfgMasterMinionNGINXPlus = IngressNginxConfig{
		Upstreams: []Upstream{
			coffeeUpstreamNginxPlus,
			teaUpstreamNGINXPlus,
		},
		Servers: []Server{
			{
				Name:         "cafe.example.com",
				ServerTokens: "on",
				Locations: []Location{
					{
						Path:                "/coffee",
						ServiceName:         "coffee-svc",
						Upstream:            coffeeUpstreamNginxPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-coffee-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
						ProxySSLName: "coffee-svc.default.svc",
					},
					{
						Path:                "/tea",
						ServiceName:         "tea-svc",
						Upstream:            teaUpstreamNGINXPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-tea-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
						ProxySSLName: "tea-svc.default.svc",
					},
				},
				SSL:               true,
				SSLCertificate:    "/etc/nginx/secrets/default-cafe-secret",
				SSLCertificateKey: "/etc/nginx/secrets/default-cafe-secret",
				StatusZone:        "cafe.example.com",
				HSTSMaxAge:        2592000,
				Ports:             []int{80},
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				HealthChecks:      make(map[string]HealthCheck),
			},
		},
		Ingress: Ingress{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
	}

	// ingressCfgMasterMinionNGINXPlusMasterMinions holds data to test the following scenario:
	//
	// Ingress Master - Minion
	//  - Master: with `path-regex` annotation
	//  - Minion 1 (cafe-ingress-coffee-minion): without `path-regex` annotation
	//  - Minion 2 (cafe-ingress-tea-minion): without `path-regex` annotation
	ingressCfgMasterMinionNGINXPlusMasterMinions = IngressNginxConfig{
		Upstreams: []Upstream{
			coffeeUpstreamNginxPlus,
			teaUpstreamNGINXPlus,
		},
		Servers: []Server{
			{
				Name:         "cafe.example.com",
				ServerTokens: "on",
				Locations: []Location{
					{
						Path:                "/coffee",
						ServiceName:         "coffee-svc",
						Upstream:            coffeeUpstreamNginxPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-coffee-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
						ProxySSLName: "coffee-svc.default.svc",
					},
					{
						Path:                "/tea",
						ServiceName:         "tea-svc",
						Upstream:            teaUpstreamNGINXPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-tea-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
						ProxySSLName: "tea-svc.default.svc",
					},
				},
				SSL:               true,
				SSLCertificate:    "/etc/nginx/secrets/default-cafe-secret",
				SSLCertificateKey: "/etc/nginx/secrets/default-cafe-secret",
				StatusZone:        "cafe.example.com",
				HSTSMaxAge:        2592000,
				Ports:             []int{80},
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				HealthChecks:      make(map[string]HealthCheck),
			},
		},
		Ingress: Ingress{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "master",
				"nginx.org/path-regex":             "case_sensitive",
			},
		},
	}

	// ingressCfgMasterMinionNGINXPlusMinionWithPathRegexAnnotation holds data to test the following scenario:
	//
	// Ingress Master - Minion
	//  - Master: without `path-regex` annotation
	//  - Minion 1 (cafe-ingress-coffee-minion): with `path-regex` annotation
	//  - Minion 2 (cafe-ingress-tea-minion): without `path-regex` annotation
	ingressCfgMasterMinionNGINXPlusMinionWithPathRegexAnnotation = IngressNginxConfig{
		Upstreams: []Upstream{
			coffeeUpstreamNginxPlus,
			teaUpstreamNGINXPlus,
		},
		Servers: []Server{
			{
				Name:         "cafe.example.com",
				ServerTokens: "on",
				Locations: []Location{
					{
						Path:                "/coffee",
						ServiceName:         "coffee-svc",
						Upstream:            coffeeUpstreamNginxPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-coffee-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
								"nginx.org/path-regex":             "case_insensitive",
							},
						},
						ProxySSLName: "coffee-svc.default.svc",
					},
					{
						Path:                "/tea",
						ServiceName:         "tea-svc",
						Upstream:            teaUpstreamNGINXPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-tea-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
						ProxySSLName: "tea-svc.default.svc",
					},
				},
				SSL:               true,
				SSLCertificate:    "/etc/nginx/secrets/default-cafe-secret",
				SSLCertificateKey: "/etc/nginx/secrets/default-cafe-secret",
				StatusZone:        "cafe.example.com",
				HSTSMaxAge:        2592000,
				Ports:             []int{80},
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				HealthChecks:      make(map[string]HealthCheck),
			},
		},
		Ingress: Ingress{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
	}

	// ingressCfgMasterMinionNGINXPlusSecondMinionWithPathRegexAnnotation holds data to test the following scenario:
	//
	// Ingress Master - Minion
	//  - Master: without `path-regex` annotation
	//  - Minion 1 (cafe-ingress-coffee-minion): without `path-regex` annotation
	//  - Minion 2 (cafe-ingress-tea-minion): with `path-regex` annotation
	ingressCfgMasterMinionNGINXPlusSecondMinionWithPathRegexAnnotation = IngressNginxConfig{
		Upstreams: []Upstream{
			coffeeUpstreamNginxPlus,
			teaUpstreamNGINXPlus,
		},
		Servers: []Server{
			{
				Name:         "cafe.example.com",
				ServerTokens: "on",
				Locations: []Location{
					{
						Path:                "/coffee",
						ServiceName:         "coffee-svc",
						Upstream:            coffeeUpstreamNginxPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-coffee-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
						ProxySSLName: "coffee-svc.default.svc",
					},
					{
						Path:                "/tea",
						ServiceName:         "tea-svc",
						Upstream:            teaUpstreamNGINXPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-tea-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
								"nginx.org/path-regex":             "case_sensitive",
							},
						},
						ProxySSLName: "tea-svc.default.svc",
					},
				},
				SSL:               true,
				SSLCertificate:    "/etc/nginx/secrets/default-cafe-secret",
				SSLCertificateKey: "/etc/nginx/secrets/default-cafe-secret",
				StatusZone:        "cafe.example.com",
				HSTSMaxAge:        2592000,
				Ports:             []int{80},
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				HealthChecks:      make(map[string]HealthCheck),
			},
		},
		Ingress: Ingress{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
	}

	// ingressCfgMasterMinionNGINXPlusMasterWithPathRegexAnnotation holds data to test the following scenario:
	//
	// Ingress Master - Minion
	//
	//  - Master: with `path-regex` annotation
	//  - Minion 1 (cafe-ingress-coffee-minion): without `path-regex` annotation
	//  - Minion 2 (cafe-ingress-tea-minion): without `path-regex` annotation
	ingressCfgMasterMinionNGINXPlusMasterWithPathRegexAnnotation = IngressNginxConfig{
		Upstreams: []Upstream{
			coffeeUpstreamNginxPlus,
			teaUpstreamNGINXPlus,
		},
		Servers: []Server{
			{
				Name:         "cafe.example.com",
				ServerTokens: "on",
				Locations: []Location{
					{
						Path:                "/coffee",
						ServiceName:         "coffee-svc",
						Upstream:            coffeeUpstreamNginxPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-coffee-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
						ProxySSLName: "coffee-svc.default.svc",
					},
					{
						Path:                "/tea",
						ServiceName:         "tea-svc",
						Upstream:            teaUpstreamNGINXPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-tea-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
						ProxySSLName: "tea-svc.default.svc",
					},
				},
				SSL:               true,
				SSLCertificate:    "/etc/nginx/secrets/default-cafe-secret",
				SSLCertificateKey: "/etc/nginx/secrets/default-cafe-secret",
				StatusZone:        "cafe.example.com",
				HSTSMaxAge:        2592000,
				Ports:             []int{80},
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				HealthChecks:      make(map[string]HealthCheck),
			},
		},
		Ingress: Ingress{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "master",
				"nginx.org/path-regex":             "case_sensitive",
			},
		},
	}

	// ingressCfgMasterMinionNGINXPlusMasterAndAllMinionsWithPathRegexAnnotation holds data to test the following scenario:
	//
	// Ingress Master - Minion
	//
	//  - Master: with `path-regex` annotation
	//  - Minion 1 (cafe-ingress-coffee-minion): with `path-regex` annotation
	//  - Minion 2 (cafe-ingress-tea-minion): with `path-regex` annotation
	ingressCfgMasterMinionNGINXPlusMasterAndAllMinionsWithPathRegexAnnotation = IngressNginxConfig{
		Upstreams: []Upstream{
			coffeeUpstreamNginxPlus,
			teaUpstreamNGINXPlus,
		},
		Servers: []Server{
			{
				Name:         "cafe.example.com",
				ServerTokens: "on",
				Locations: []Location{
					{
						Path:                "/coffee",
						ServiceName:         "coffee-svc",
						Upstream:            coffeeUpstreamNginxPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-coffee-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
								"nginx.org/path-regex":             "case_insensitive",
							},
						},
						ProxySSLName: "coffee-svc.default.svc",
					},
					{
						Path:                "/tea",
						ServiceName:         "tea-svc",
						Upstream:            teaUpstreamNGINXPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-tea-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
								"nginx.org/path-regex":             "case_insensitive",
							},
						},
						ProxySSLName: "tea-svc.default.svc",
					},
				},
				SSL:               true,
				SSLCertificate:    "/etc/nginx/secrets/default-cafe-secret",
				SSLCertificateKey: "/etc/nginx/secrets/default-cafe-secret",
				StatusZone:        "cafe.example.com",
				HSTSMaxAge:        2592000,
				Ports:             []int{80},
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				HealthChecks:      make(map[string]HealthCheck),
			},
		},
		Ingress: Ingress{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "master",
				"nginx.org/path-regex":             "case_sensitive",
			},
		},
	}

	// ingressCfgMasterMinionNGINXPlusMasterWithoutPathRegexMinionsWithPathRegexAnnotation holds data to test the following scenario:
	//
	// Ingress Master - Minion
	//  - Master: without `path-regex` annotation
	//  - Minion 1 (cafe-ingress-coffee-minion): with `path-regex` annotation
	//  - Minion 2 (cafe-ingress-tea-minion): with `path-regex` annotation
	ingressCfgMasterMinionNGINXPlusMasterWithoutPathRegexMinionsWithPathRegexAnnotation = IngressNginxConfig{
		Upstreams: []Upstream{
			coffeeUpstreamNginxPlus,
			teaUpstreamNGINXPlus,
		},
		Servers: []Server{
			{
				Name:         "cafe.example.com",
				ServerTokens: "on",
				Locations: []Location{
					{
						Path:                "/coffee",
						ServiceName:         "coffee-svc",
						Upstream:            coffeeUpstreamNginxPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-coffee-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
								"nginx.org/path-regex":             "case_insensitive",
							},
						},
						ProxySSLName: "coffee-svc.default.svc",
					},
					{
						Path:                "/tea",
						ServiceName:         "tea-svc",
						Upstream:            teaUpstreamNGINXPlus,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &Ingress{
							Name:      "cafe-ingress-tea-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.org/mergeable-ingress-type": "minion",
								"nginx.org/path-regex":             "case_sensitive",
							},
						},
						ProxySSLName: "tea-svc.default.svc",
					},
				},
				SSL:               true,
				SSLCertificate:    "/etc/nginx/secrets/default-cafe-secret",
				SSLCertificateKey: "/etc/nginx/secrets/default-cafe-secret",
				StatusZone:        "cafe.example.com",
				HSTSMaxAge:        2592000,
				Ports:             []int{80},
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				HealthChecks:      make(map[string]HealthCheck),
			},
		},
		Ingress: Ingress{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
	}

	// Ingress Config example without added annotations
	ingressCfgHTTP2On = IngressNginxConfig{
		Servers: []Server{
			{
				Name:              "test.example.com",
				ServerTokens:      "off",
				StatusZone:        "test.example.com",
				SSL:               true,
				HTTP2:             true,
				SSLCertificate:    "secret.pem",
				SSLCertificateKey: "secret.pem",
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				Locations: []Location{
					{
						Path:                "/tea",
						Upstream:            testUpstream,
						ProxyConnectTimeout: "10s",
						ProxyReadTimeout:    "10s",
						ProxySendTimeout:    "10s",
						ClientMaxBodySize:   "2m",
						MinionIngress: &Ingress{
							Name:      "tea-minion",
							Namespace: "default",
						},
					},
				},
				HealthChecks: map[string]HealthCheck{"test": healthCheck},
				JWTRedirectLocations: []JWTRedirectLocation{
					{
						Name:     "@login_url-default-cafe-ingress",
						LoginURL: "https://test.example.com/login",
					},
				},
			},
		},
		Upstreams: []Upstream{testUpstream},
		Keepalive: "16",
		Ingress: Ingress{
			Name:      "cafe-ingress",
			Namespace: "default",
		},
	}
)

var testUpstream = Upstream{
	Name:             "test",
	UpstreamZoneSize: "256k",
	UpstreamServers: []UpstreamServer{
		{
			Address:     "127.0.0.1:8181",
			MaxFails:    0,
			MaxConns:    0,
			FailTimeout: "1s",
			SlowStart:   "5s",
		},
	},
}

var (
	headers     = map[string]string{"Test-Header": "test-header-value"}
	healthCheck = HealthCheck{
		UpstreamName: "test",
		Fails:        1,
		Interval:     1,
		Passes:       1,
		Headers:      headers,
	}
)
