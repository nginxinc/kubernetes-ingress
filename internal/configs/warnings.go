package configs

import "k8s.io/apimachinery/pkg/runtime"

// Warnings stores a list of warnings for a given runtime k8s object in a map
type Warnings map[runtime.Object][]string

func newWarnings() Warnings {
	return make(map[runtime.Object][]string)
}

// Add adds new Warnings to the map
func (w Warnings) Add(warnings Warnings) {
	for k, v := range warnings {
		w[k] = v
	}
}
