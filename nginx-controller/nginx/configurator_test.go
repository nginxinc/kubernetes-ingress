package nginx

import "testing"

func TestPathOrDefaultReturnDefault(t *testing.T) {
	path := ""
	expected := "/"
	if pathOrDefault(path) != expected {
		t.Errorf("pathOrDefault(%q) should return %q", path, expected)
	}
}

func TestPathOrDefaultReturnActual(t *testing.T) {
	path := "/path/to/resource"
	if pathOrDefault(path) != path {
		t.Errorf("pathOrDefault(%q) should return %q", path, path)
	}
}

func TestParseRewrites(t *testing.T) {
	serviceName := "coffee-svc"
	serviceNamePart := "serviceName=" + serviceName
	rewritePath := "/beans/"
	rewritePathPart := "rewrite=" + rewritePath
	rewriteService := serviceNamePart + " " + rewritePathPart

	serviceNameActual, rewritePathActual, err := parseRewrites(rewriteService)
	if serviceName != serviceNameActual || rewritePath != rewritePathActual || err != nil {
		t.Errorf("parseRewrites(%s) should return %q, %q, nil; got %q, %q, %v", rewriteService, serviceName, rewritePath, serviceNameActual, rewritePathActual, err)
	}
}

func TestParseRewritesInvalidFormat(t *testing.T) {
	rewriteService := "serviceNamecoffee-svc rewrite=/"

	_, _, err := parseRewrites(rewriteService)
	if err == nil {
		t.Errorf("parseRewrites(%s) should return error, got nil", rewriteService)
	}
}
