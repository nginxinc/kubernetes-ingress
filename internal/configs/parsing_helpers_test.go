package configs

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var configMap = v1.ConfigMap{
	ObjectMeta: meta_v1.ObjectMeta{
		Name:      "test",
		Namespace: "default",
	},
	TypeMeta: meta_v1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	},
}
var ingress = v1beta1.Ingress{
	ObjectMeta: meta_v1.ObjectMeta{
		Name:      "test",
		Namespace: "kube-system",
	},
	TypeMeta: meta_v1.TypeMeta{
		Kind:       "Ingress",
		APIVersion: "extensions/v1beta1",
	},
}

func TestGetMapKeyAsBool(t *testing.T) {
	configMap := configMap
	configMap.Data = map[string]string{
		"key": "True",
	}

	b, exists, err := GetMapKeyAsBool(configMap.Data, "key", &configMap)
	if !exists {
		t.Errorf("The key 'key' must exist in the configMap")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if b != true {
		t.Errorf("Result should be true")
	}
}

func TestGetMapKeyAsBoolNotFound(t *testing.T) {
	configMap := configMap
	configMap.Data = map[string]string{}

	_, exists, _ := GetMapKeyAsBool(configMap.Data, "key", &configMap)
	if exists {
		t.Errorf("The key 'key' must not exist in the configMap")
	}
}

func TestGetMapKeyAsBoolErrorMessage(t *testing.T) {
	cfgm := configMap
	cfgm.Data = map[string]string{
		"key": "string",
	}

	// Test with configmap
	_, _, err := GetMapKeyAsBool(cfgm.Data, "key", &cfgm)
	if err == nil {
		t.Error("An error was expected")
	}
	expected := `ConfigMap default/test 'key' contains invalid bool: strconv.ParseBool: parsing "string": invalid syntax, ignoring`
	if err.Error() != expected {
		t.Errorf("The error message does not match expectations:\nGot: %v\nExpected: %v", err, expected)
	}

	// Test with ingress object
	ingress := ingress
	ingress.Annotations = map[string]string{
		"key": "other_string",
	}

	_, _, err = GetMapKeyAsBool(ingress.Annotations, "key", &ingress)
	if err == nil {
		t.Error("An error was expected")
	}
	expected = `Ingress kube-system/test 'key' contains invalid bool: strconv.ParseBool: parsing "other_string": invalid syntax, ignoring`
	if err.Error() != expected {
		t.Errorf("The error message does not match expectations:\nGot: %v\nExpected: %v", err, expected)
	}
}

func TestGetMapKeyAsInt(t *testing.T) {
	configMap := configMap
	configMap.Data = map[string]string{
		"key": "123456789",
	}

	i, exists, err := GetMapKeyAsInt(configMap.Data, "key", &configMap)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !exists {
		t.Errorf("The key 'key' must exist in the configMap")
	}
	expected := 123456789
	if i != expected {
		t.Errorf("Unexpected return value:\nGot: %v\nExpected: %v", i, expected)
	}
}

func TestGetMapKeyAsIntNotFound(t *testing.T) {
	configMap := configMap
	configMap.Data = map[string]string{}

	_, exists, _ := GetMapKeyAsInt(configMap.Data, "key", &configMap)
	if exists {
		t.Errorf("The key 'key' must not exist in the configMap")
	}
}

func TestGetMapKeyAsIntErrorMessage(t *testing.T) {
	cfgm := configMap
	cfgm.Data = map[string]string{
		"key": "string",
	}

	// Test with configmap
	_, _, err := GetMapKeyAsInt(cfgm.Data, "key", &cfgm)
	if err == nil {
		t.Error("An error was expected")
	}
	expected := `ConfigMap default/test 'key' contains invalid integer: strconv.Atoi: parsing "string": invalid syntax, ignoring`
	if err.Error() != expected {
		t.Errorf("The error message does not match expectations:\nGot: %v\nExpected: %v", err, expected)
	}

	// Test with ingress object
	ingress := ingress
	ingress.Annotations = map[string]string{
		"key": "other_string",
	}

	_, _, err = GetMapKeyAsInt(ingress.Annotations, "key", &ingress)
	if err == nil {
		t.Error("An error was expected")
	}
	expected = `Ingress kube-system/test 'key' contains invalid integer: strconv.Atoi: parsing "other_string": invalid syntax, ignoring`
	if err.Error() != expected {
		t.Errorf("The error message does not match expectations:\nGot: %v\nExpected: %v", err, expected)
	}
}

func TestGetMapKeyAsInt64(t *testing.T) {
	configMap := configMap
	configMap.Data = map[string]string{
		"key": "123456789",
	}

	i, exists, err := GetMapKeyAsInt64(configMap.Data, "key", &configMap)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !exists {
		t.Errorf("The key 'key' must exist in the configMap")
	}
	var expected int64 = 123456789
	if i != expected {
		t.Errorf("Unexpected return value:\nGot: %v\nExpected: %v", i, expected)
	}
}

func TestGetMapKeyAsInt64NotFound(t *testing.T) {
	configMap := configMap
	configMap.Data = map[string]string{}

	_, exists, _ := GetMapKeyAsInt64(configMap.Data, "key", &configMap)
	if exists {
		t.Errorf("The key 'key' must not exist in the configMap")
	}
}

func TestGetMapKeyAsInt64ErrorMessage(t *testing.T) {
	cfgm := configMap
	cfgm.Data = map[string]string{
		"key": "string",
	}

	// Test with configmap
	_, _, err := GetMapKeyAsInt64(cfgm.Data, "key", &cfgm)
	if err == nil {
		t.Error("An error was expected")
	}
	expected := `ConfigMap default/test 'key' contains invalid integer: strconv.ParseInt: parsing "string": invalid syntax, ignoring`
	if err.Error() != expected {
		t.Errorf("The error message does not match expectations:\nGot: %v\nExpected: %v", err, expected)
	}

	// Test with ingress object
	ingress := ingress
	ingress.Annotations = map[string]string{
		"key": "other_string",
	}

	_, _, err = GetMapKeyAsInt64(ingress.Annotations, "key", &ingress)
	if err == nil {
		t.Error("An error was expected")
	}
	expected = `Ingress kube-system/test 'key' contains invalid integer: strconv.ParseInt: parsing "other_string": invalid syntax, ignoring`
	if err.Error() != expected {
		t.Errorf("The error message does not match expectations:\nGot: %v\nExpected: %v", err, expected)
	}
}

func TestGetMapKeyAsStringSlice(t *testing.T) {
	configMap := configMap
	configMap.Data = map[string]string{
		"key": "1.String,2.String,3.String",
	}

	slice, exists, err := GetMapKeyAsStringSlice(configMap.Data, "key", &configMap, ",")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !exists {
		t.Errorf("The key 'key' must exist in the configMap")
	}
	expected := []string{"1.String", "2.String", "3.String"}
	t.Log(expected)
	if !reflect.DeepEqual(expected, slice) {
		t.Errorf("Unexpected return value:\nGot: %#v\nExpected: %#v", slice, expected)
	}

}

func TestGetMapKeyAsStringSliceMultilineSnippets(t *testing.T) {
	configMap := configMap
	configMap.Data = map[string]string{
		"server-snippets": `
			if ($new_uri) {
				rewrite ^ $new_uri permanent;
			}`,
	}

	slice, exists, err := GetMapKeyAsStringSlice(configMap.Data, "server-snippets", &configMap, "\n")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !exists {
		t.Errorf("The key 'server-snippets' must exist in the configMap")
	}
	expected := []string{"", "\t\t\tif ($new_uri) {", "\t\t\t\trewrite ^ $new_uri permanent;", "\t\t\t}"}
	t.Log(expected)
	if !reflect.DeepEqual(expected, slice) {
		t.Errorf("Unexpected return value:\nGot: %#v\nExpected: %#v", slice, expected)
	}
}

func TestGetMapKeyAsStringSliceNotFound(t *testing.T) {
	configMap := configMap
	configMap.Data = map[string]string{}

	_, exists, _ := GetMapKeyAsStringSlice(configMap.Data, "key", &configMap, ",")
	if exists {
		t.Errorf("The key 'key' must not exist in the configMap")
	}
}

func TestParseLBMethod(t *testing.T) {
	var testsWithValidInput = []struct {
		input    string
		expected string
	}{
		{"least_conn", "least_conn"},
		{"round_robin", ""},
		{"ip_hash", "ip_hash"},
		{"random", "random"},
		{"random two", "random two"},
		{"random two least_conn", "random two least_conn"},
		{"hash $request_id", "hash $request_id"},
		{"hash $request_id consistent", "hash $request_id consistent"},
	}

	var invalidInput = []string{
		"",
		"blabla",
		"least_time header",
		"hash123",
		"hash $request_id conwrongspelling",
		"random one",
		"random two least_time=header",
		"random two least_time=last_byte",
		"random two ip_hash",
	}

	for _, test := range testsWithValidInput {
		result, err := ParseLBMethod(test.input)
		if err != nil {
			t.Errorf("TestParseLBMethod(%q) returned an error for valid input", test.input)
		}

		if result != test.expected {
			t.Errorf("TestParseLBMethod(%q) returned %q expected %q", test.input, result, test.expected)
		}
	}

	for _, input := range invalidInput {
		_, err := ParseLBMethod(input)
		if err == nil {
			t.Errorf("TestParseLBMethod(%q) does not return an error for invalid input", input)
		}
	}
}

func TestParseLBMethodForPlus(t *testing.T) {
	var testsWithValidInput = []struct {
		input    string
		expected string
	}{
		{"least_conn", "least_conn"},
		{"round_robin", ""},
		{"ip_hash", "ip_hash"},
		{"random", "random"},
		{"random two", "random two"},
		{"random two least_conn", "random two least_conn"},
		{"random two least_time=header", "random two least_time=header"},
		{"random two least_time=last_byte", "random two least_time=last_byte"},
		{"hash $request_id", "hash $request_id"},
		{"least_time header", "least_time header"},
		{"least_time last_byte", "least_time last_byte"},
		{"least_time header inflight", "least_time header inflight"},
		{"least_time last_byte inflight", "least_time last_byte inflight"},
	}

	var invalidInput = []string{
		"",
		"blabla",
		"hash123",
		"least_time",
		"last_byte",
		"least_time inflight header",
		"random one",
		"random two ip_hash",
		"random two least_time",
	}

	for _, test := range testsWithValidInput {
		result, err := ParseLBMethodForPlus(test.input)
		if err != nil {
			t.Errorf("TestParseLBMethod(%q) returned an error for valid input", test.input)
		}

		if result != test.expected {
			t.Errorf("TestParseLBMethod(%q) returned %q expected %q", test.input, result, test.expected)
		}
	}

	for _, input := range invalidInput {
		_, err := ParseLBMethodForPlus(input)
		if err == nil {
			t.Errorf("TestParseLBMethod(%q) does not return an error for invalid input", input)
		}
	}
}

func TestParseTime(t *testing.T) {
	var testsWithValidInput = []string{"1", "1m10s", "11 11", "5m 30s", "1s", "100m", "5w", "15m", "11M", "3h", "100y", "600"}
	var invalidInput = []string{"ss", "rM", "m0m", "s1s", "-5s", "", "1L"}
	for _, test := range testsWithValidInput {
		result, err := ParseTime(test)
		if err != nil {
			t.Errorf("TestparseTime(%q) returned an error for valid input", test)
		}
		if test != result {
			t.Errorf("TestparseTime(%q) returned %q expected %q", test, result, test)
		}
	}
	for _, test := range invalidInput {
		result, err := ParseTime(test)
		if err == nil {
			t.Errorf("TestparseTime(%q) didn't return error. Returned: %q", test, result)
		}
	}
}
