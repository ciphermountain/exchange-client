package restclient_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ciphermountain/exchange-client/pkg/restclient"
)

func TestOrderRequest(t *testing.T) {
	t.Parallel()

	requestJSON := `{"action":"BUY","base":"USDT","quote":"BTC","type":{"name":"MARKET","base":"USDT","quantity":"12345"}}`

	var request restclient.OrderRequest
	require.NoError(t, json.Unmarshal([]byte(requestJSON), &request))

	mOrder, err := request.Type.AsMarketOrderRequest()
	require.NoError(t, err)

	assert.Equal(t, "12345", mOrder.Quantity)
}
