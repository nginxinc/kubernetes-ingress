package nginx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDenyCollisionHandler(t *testing.T) {
	ch := NewDenyCollisionHandler()
	t.Run("Add first ingress", func(t *testing.T) {
		assert := assert.New(t)
		result, err := ch.AddConfigs(&ingress1, []Server{ingress1Server1})
		if assert.NoError(err) && assert.NotNil(result) && assert.Len(result, 1) {
			assert.Equal(result[0], ingress1Server1)
		}
	})

	t.Run("Add 2nd ingress", func(t *testing.T) {
		assert := assert.New(t)
		result, err := ch.AddConfigs(&ingress2, []Server{ingress2Server1, ingress2Server2})
		if assert.NoError(err) && assert.Len(result, 2, "Unexpected number of configs returned") {
			// 1. servername
			assert.Equal(ingress2Server1.Name, result[0].Name, "Server names do not match")
			assert.Contains(result[0].Locations, ingress2Location1)
			assert.Contains(result[0].Locations, ingress2Location2)
			if assert.Len(result[0].Upstreams, 1, "Unexpected number of upstreams") {
				assert.Contains(result[0].Upstreams, ingress2Upstream1)
			}

			// 2. Server
			assert.Equal(ingress2Server2.Name, result[1].Name, "Server names do not match")
			assert.Contains(result[1].Locations, ingress2Location3)
			if assert.Len(result[1].Upstreams, 1, "Unexpected number of upstreams") {
				assert.Contains(result[1].Upstreams, ingress2Upstream2)
			}
		}
	})

	t.Run("Remove 2nd ingress", func(t *testing.T) {
		assert := assert.New(t)
		changed, deleted, err := ch.RemoveConfigs("default/ing2")
		if assert.NoError(err) {
			if assert.Len(changed, 1, "Unexpected number of changed servers") {
				assert.Contains(changed, ingress1Server1)
			}
			if assert.Len(deleted, 1) {
				assert.Contains(deleted, ingress2Server2)
			}
		}
	})

	t.Run("Add 3rd ingress", func(t *testing.T) {
		assert := assert.New(t)
		changed, err := ch.AddConfigs(&ingress3, []Server{ingress3Server1})
		if assert.NoError(err) {
			assert.Len(changed, 0, "Unexpected number of changed servers")
		}
	})

	t.Run("Try to remove 2nd ingress again", func(t *testing.T) {
		assert := assert.New(t)
		_, _, err := ch.RemoveConfigs("default/ing2")
		if assert.Error(err) {
			assert.Equal("Ingress 'default/ing2' cannot be removed, because it was not found in the mapping", err.Error())
		}
	})
}
