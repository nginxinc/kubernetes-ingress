package version2

import (
	"strings"
	"text/template"
)

func headerListToCIMap(headers []Header) map[string]string {
	ret := make(map[string]string)

	for _, header := range headers {
		ret[strings.ToLower(header.Name)] = header.Value
	}

	return ret
}

func hasCIKey(key string, d map[string]string) bool {
	_, ok := d[strings.ToLower(key)]
	return ok
}

// toLower takes a string and make it lowercase.
//
// Example:
//
//	{{ if .SameSite}} samesite={{.SameSite | toLower }}{{ end }}
func toLower(s string) string {
	return strings.ToLower(s)
}

var helperFunctions = template.FuncMap{
	"headerListToCIMap": headerListToCIMap,
	"hasCIKey":          hasCIKey,
	"toLower":           toLower,
}
