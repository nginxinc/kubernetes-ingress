package version1

import (
	"bytes"
	"testing"
	"text/template"
)

func TestWithPathRegex_MatchesCaseSensitiveModifier(t *testing.T) {
	t.Parallel()

	want := "~ \"^/coffee/[A-Z0-9]{3}\""
	got := makePathRegex("/coffee/[A-Z0-9]{3}", map[string]string{"nginx.org/path-regex": "case_sensitive"})
	if got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestWithPathRegex_MatchesCaseInsensitiveModifier(t *testing.T) {
	t.Parallel()

	want := "~* \"^/coffee/[A-Z0-9]{3}\""
	got := makePathRegex("/coffee/[A-Z0-9]{3}", map[string]string{"nginx.org/path-regex": "case_insensitive"})
	if got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestWithPathReqex_MatchesExactModifier(t *testing.T) {
	t.Parallel()

	want := "= \"/coffee\""
	got := makePathRegex("/coffee", map[string]string{"nginx.org/path-regex": "exact"})
	if got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestWithPathReqex_DoesNotMatchModifier(t *testing.T) {
	t.Parallel()

	want := "/coffee"
	got := makePathRegex("/coffee", map[string]string{"nginx.org/path-regex": "bogus"})
	if got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestWithPathReqex_DoesNotMatchEmptyModifier(t *testing.T) {
	t.Parallel()

	want := "/coffee"
	got := makePathRegex("/coffee", map[string]string{"nginx.org/path-regex": ""})
	if got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestWithPathReqex_DoesNotMatchBogusAnnotationName(t *testing.T) {
	t.Parallel()

	want := "/coffee"
	got := makePathRegex("/coffee", map[string]string{"nginx.org/bogus-annotation": ""})
	if got != want {
		t.Errorf("got: %s, want: %s", got, want)
	}
}

func TestSplitHelperFunction(t *testing.T) {
	t.Parallel()
	const tpl = `{{range $n := split . ","}}{{$n}} {{end}}`

	tmpl, err := template.New("testTemplate").Funcs(helperFunctions).Parse(tpl)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	var buf bytes.Buffer

	input := "foo,bar"
	expected := "foo bar "

	err = tmpl.Execute(&buf, input)
	if err != nil {
		t.Fatalf("Failed to execute the template %v", err)
	}

	if buf.String() != expected {
		t.Fatalf("Template generated wrong config, got %v but expected %v.", buf.String(), expected)
	}
}

func TestTrimHelperFunction(t *testing.T) {
	t.Parallel()
	const tpl = `{{trim .}}`

	tmpl, err := template.New("testTemplate").Funcs(helperFunctions).Parse(tpl)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	var buf bytes.Buffer

	input := "  foobar     "
	expected := "foobar"

	err = tmpl.Execute(&buf, input)
	if err != nil {
		t.Fatalf("Failed to execute the template %v", err)
	}

	if buf.String() != expected {
		t.Fatalf("Template generated wrong config, got %v but expected %v.", buf.String(), expected)
	}
}
