package client

import (
	"net/http"
	"testing"

	"github.com/glynternet/go-accounting-storage"
	"github.com/stretchr/testify/assert"
)

// ensure that a Client can be used as a storage.Storage
var _ storage.Storage = Client("")

func Test_getBodyFromEndpoint(t *testing.T) {
	t.Run("get error", func(t *testing.T) {
		c := Client("bloopybloop")
		as, err := c.getBodyFromEndpoint("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "getting from endpoint")
		assert.Nil(t, as)
	})

	t.Run("unexpected status", func(t *testing.T) {
		srv := newJSONTestServer(nil, http.StatusTeapot)
		defer srv.Close()
		c := Client(srv.URL)
		as, err := c.getBalancesFromEndpoint("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server returned unexpected code")
		assert.Nil(t, as)
	})
}
