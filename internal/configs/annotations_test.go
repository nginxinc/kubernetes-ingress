package configs

import (
	"errors"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseRewrites(t *testing.T) {
	t.Parallel()
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

func TestParseRewritesWithLeadingAndTrailingWhitespace(t *testing.T) {
	t.Parallel()
	serviceName := "coffee-svc"
	serviceNamePart := "serviceName=" + serviceName
	rewritePath := "/beans/"
	rewritePathPart := "rewrite=" + rewritePath
	rewriteService := "\t\n " + serviceNamePart + " " + rewritePathPart + " \t\n"

	serviceNameActual, rewritePathActual, err := parseRewrites(rewriteService)
	if serviceName != serviceNameActual || rewritePath != rewritePathActual || err != nil {
		t.Errorf("parseRewrites(%s) should return %q, %q, nil; got %q, %q, %v", rewriteService, serviceName, rewritePath, serviceNameActual, rewritePathActual, err)
	}
}

func TestParseRewritesInvalidFormat(t *testing.T) {
	t.Parallel()
	rewriteService := "serviceNamecoffee-svc rewrite=/"

	_, _, err := parseRewrites(rewriteService)
	if err == nil {
		t.Errorf("parseRewrites(%s) should return error, got nil", rewriteService)
	}
}

func TestParseStickyService(t *testing.T) {
	t.Parallel()
	serviceName := "coffee-svc"
	serviceNamePart := "serviceName=" + serviceName
	stickyCookie := "srv_id expires=1h domain=.example.com path=/"
	stickyService := serviceNamePart + " " + stickyCookie

	serviceNameActual, stickyCookieActual, err := parseStickyService(stickyService)
	if serviceName != serviceNameActual || stickyCookie != stickyCookieActual || err != nil {
		t.Errorf("parseStickyService(%s) should return %q, %q, nil; got %q, %q, %v", stickyService, serviceName, stickyCookie, serviceNameActual, stickyCookieActual, err)
	}
}

func TestParseStickyServiceInvalidFormat(t *testing.T) {
	t.Parallel()
	stickyService := "serviceNamecoffee-svc srv_id expires=1h domain=.example.com path=/"

	_, _, err := parseStickyService(stickyService)
	if err == nil {
		t.Errorf("parseStickyService(%s) should return error, got nil", stickyService)
	}
}

func TestFilterMasterAnnotations(t *testing.T) {
	t.Parallel()
	masterAnnotations := map[string]string{
		"nginx.org/rewrites":                "serviceName=service1 rewrite=rewrite1",
		"nginx.org/ssl-services":            "service1",
		"nginx.org/hsts":                    "True",
		"nginx.org/hsts-max-age":            "2700000",
		"nginx.org/hsts-include-subdomains": "True",
	}
	removedAnnotations := filterMasterAnnotations(masterAnnotations)

	expectedfilteredMasterAnnotations := map[string]string{
		"nginx.org/hsts":                    "True",
		"nginx.org/hsts-max-age":            "2700000",
		"nginx.org/hsts-include-subdomains": "True",
	}
	expectedRemovedAnnotations := []string{
		"nginx.org/rewrites",
		"nginx.org/ssl-services",
	}

	sort.Strings(removedAnnotations)
	sort.Strings(expectedRemovedAnnotations)

	if !reflect.DeepEqual(expectedfilteredMasterAnnotations, masterAnnotations) {
		t.Errorf("filterMasterAnnotations returned %v, but expected %v", masterAnnotations, expectedfilteredMasterAnnotations)
	}
	if !reflect.DeepEqual(expectedRemovedAnnotations, removedAnnotations) {
		t.Errorf("filterMasterAnnotations returned %v, but expected %v", removedAnnotations, expectedRemovedAnnotations)
	}
}

func TestFilterMinionAnnotations(t *testing.T) {
	t.Parallel()
	minionAnnotations := map[string]string{
		"nginx.org/rewrites":                "serviceName=service1 rewrite=rewrite1",
		"nginx.org/ssl-services":            "service1",
		"nginx.org/hsts":                    "True",
		"nginx.org/hsts-max-age":            "2700000",
		"nginx.org/hsts-include-subdomains": "True",
	}
	removedAnnotations := filterMinionAnnotations(minionAnnotations)

	expectedfilteredMinionAnnotations := map[string]string{
		"nginx.org/rewrites":     "serviceName=service1 rewrite=rewrite1",
		"nginx.org/ssl-services": "service1",
	}
	expectedRemovedAnnotations := []string{
		"nginx.org/hsts",
		"nginx.org/hsts-max-age",
		"nginx.org/hsts-include-subdomains",
	}

	sort.Strings(removedAnnotations)
	sort.Strings(expectedRemovedAnnotations)

	if !reflect.DeepEqual(expectedfilteredMinionAnnotations, minionAnnotations) {
		t.Errorf("filterMinionAnnotations returned %v, but expected %v", minionAnnotations, expectedfilteredMinionAnnotations)
	}
	if !reflect.DeepEqual(expectedRemovedAnnotations, removedAnnotations) {
		t.Errorf("filterMinionAnnotations returned %v, but expected %v", removedAnnotations, expectedRemovedAnnotations)
	}
}

func TestMergeMasterAnnotationsIntoMinion(t *testing.T) {
	t.Parallel()
	masterAnnotations := map[string]string{
		"nginx.org/proxy-buffering":       "True",
		"nginx.org/proxy-buffers":         "2",
		"nginx.org/proxy-buffer-size":     "8k",
		"nginx.org/hsts":                  "True",
		"nginx.org/hsts-max-age":          "2700000",
		"nginx.org/proxy-connect-timeout": "50s",
		"nginx.com/jwt-token":             "$cookie_auth_token",
	}
	minionAnnotations := map[string]string{
		"nginx.org/client-max-body-size":  "2m",
		"nginx.org/proxy-connect-timeout": "20s",
	}
	mergeMasterAnnotationsIntoMinion(minionAnnotations, masterAnnotations)

	expectedMergedAnnotations := map[string]string{
		"nginx.org/proxy-buffering":       "True",
		"nginx.org/proxy-buffers":         "2",
		"nginx.org/proxy-buffer-size":     "8k",
		"nginx.org/client-max-body-size":  "2m",
		"nginx.org/proxy-connect-timeout": "20s",
	}
	if !reflect.DeepEqual(expectedMergedAnnotations, minionAnnotations) {
		t.Errorf("mergeMasterAnnotationsIntoMinion returned %v, but expected %v", minionAnnotations, expectedMergedAnnotations)
	}
}

func TestParseProxySetHeaderInputString(t *testing.T) {
	t.Parallel()
	headers1 := []string{"X-Forwarded-For"}
	headers2 := []string{"ABC"}
	headers3 := []string{"Test"}

	headers := [][]string{headers1, headers2, headers3}
	for _, header := range headers {
		err := ParseProxySetHeader(header)
		if err != nil {
			t.Errorf("want nil, got error on valid input: %+v", header)
		}
	}
}

func TestParseProxySetHeaderInvalidInputString(t *testing.T) {
	t.Parallel()
	headers1 := []string{"X-Forwarded-For1"}
	headers2 := []string{"ABC!"}
	headers3 := []string{""}
	headers4 := []string{" "}

	headers := [][]string{headers1, headers2, headers3, headers4}
	for _, header := range headers {
		err := ParseProxySetHeader(header)
		if err == nil {
			t.Errorf("want error on input %+v, got nil", header)
		}
	}
}

// ParseProxySetHeader ensures that the string value contains only letters
func ParseProxySetHeader(headers []string) error {
	for _, header := range headers {
		if err := ValidateProxySetHeader(header); err != nil {
			return err
		}
	}
	return nil
}

var setHeader = regexp.MustCompile("[a-zA-Z]+$")

func ValidateProxySetHeader(header string) error {
	header = strings.TrimSpace(header)

	if !setHeader.MatchString(header) {
		return errors.New("error: invalid header syntax")
	}
	return nil
}

func TestParseRateLimitAnnotations(t *testing.T) {
	context := &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "context",
		},
	}

	if errors := parseRateLimitAnnotations(map[string]string{
		"nginx.org/limit-req-rate":        "200r/s",
		"nginx.org/limit-req-key":         "${request_uri}",
		"nginx.org/limit-req-burst":       "100",
		"nginx.org/limit-req-delay":       "80",
		"nginx.org/limit-req-no-delay":    "true",
		"nginx.org/limit-req-reject-code": "429",
		"nginx.org/limit-req-zone-size":   "11m",
		"nginx.org/limit-req-dry-run":     "true",
		"nginx.org/limit-req-log-level":   "info",
	}, NewDefaultConfigParams(false), context); len(errors) > 0 {
		t.Error("Errors when parsing valid limit-req annotations")
	}

	if errors := parseRateLimitAnnotations(map[string]string{
		"nginx.org/limit-req-rate": "200",
	}, NewDefaultConfigParams(false), context); len(errors) == 0 {
		t.Error("No Errors when parsing invalid request rate")
	}

	if errors := parseRateLimitAnnotations(map[string]string{
		"nginx.org/limit-req-rate": "200r/h",
	}, NewDefaultConfigParams(false), context); len(errors) == 0 {
		t.Error("No Errors when parsing invalid request rate")
	}

	if errors := parseRateLimitAnnotations(map[string]string{
		"nginx.org/limit-req-rate": "0r/s",
	}, NewDefaultConfigParams(false), context); len(errors) == 0 {
		t.Error("No Errors when parsing invalid request rate")
	}

	if errors := parseRateLimitAnnotations(map[string]string{
		"nginx.org/limit-req-zone-size": "10abc",
	}, NewDefaultConfigParams(false), context); len(errors) == 0 {
		t.Error("No Errors when parsing invalid zone size")
	}

	if errors := parseRateLimitAnnotations(map[string]string{
		"nginx.org/limit-req-log-level": "foobar",
	}, NewDefaultConfigParams(false), context); len(errors) == 0 {
		t.Error("No Errors when parsing invalid log level")
	}
}
