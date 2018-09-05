package nginx

import (
	"reflect"
	"testing"

	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var configMap = api_v1.ConfigMap{
	ObjectMeta: meta_v1.ObjectMeta{
		Name:      "test",
		Namespace: "default",
	},
	TypeMeta: meta_v1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	},
}
var ingress = extensions.Ingress{
	ObjectMeta: meta_v1.ObjectMeta{
		Name:      "test",
		Namespace: "kube-system",
	},
	TypeMeta: meta_v1.TypeMeta{
		Kind:       "Ingress",
		APIVersion: "extensions/v1beta1",
	},
}

//
// GetMapKeyAsBool
//
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

//
// GetMapKeyAsInt
//
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
	var expected int64 = 123456789
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
	expected := `ConfigMap default/test 'key' contains invalid integer: strconv.ParseInt: parsing "string": invalid syntax, ignoring`
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
	expected = `Ingress kube-system/test 'key' contains invalid integer: strconv.ParseInt: parsing "other_string": invalid syntax, ignoring`
	if err.Error() != expected {
		t.Errorf("The error message does not match expectations:\nGot: %v\nExpected: %v", err, expected)
	}
}

//
// GetMapKeyAsStringSlice
//
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
