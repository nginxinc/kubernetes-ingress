package nginx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeverMerger(t *testing.T) {
	n := NewNeverMerger()
	t.Run("First ingress", func(t *testing.T) {
		assert := assert.New(t)
		result := n.Merge(&ingress1, []IngressNginxConfig{ingress1Config1})
		if assert.NotNil(result) && assert.Len(result, 1) {
			assert.Equal(result[0], ingress1Config1)
		}
	})

	t.Run("Merge 2nd ingress", func(t *testing.T) {
		assert := assert.New(t)
		result := n.Merge(&ingress2, []IngressNginxConfig{ingress2Config1, ingress2Config2})
		if assert.Len(result, 1, "Unexpected number of configs returned") {
			// 1. IngressNginxConfig
			assert.Equal(ingress2Server2.Name, result[0].Server.Name, "Server names do not match")
			assert.Contains(result[0].Server.Locations, ingress2Location3)
			if assert.Len(result[0].Upstreams, 1, "Unexpected number of upstreams") {
				assert.Contains(result[0].Upstreams, ingress2Upstream2)
			}
		}
	})

	t.Run("Seperate 2nd ingress", func(t *testing.T) {
		assert := assert.New(t)
		result, deleted := n.Separate("default/ing2")
		if assert.Len(result, 0, "Unexpected number of configs returned") {
			if assert.Len(deleted, 1) {
				assert.Contains(deleted, "two.example.com")
			}
		}
	})
}
