package rest_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ciphermountain/exchange-client/go/pkg/rest"
)

func TestPaymentOrderChainRequestValidateWithoutTransfer(t *testing.T) {
	t.Parallel()

	requestJSON := `{"orders":[{"action":"BUY","base":"USDT","quote":"BTC","type":{"name":"MARKET","base":"USDT","quantity":"100"}}]}`

	var request rest.PaymentOrderChainRequest
	require.NoError(t, json.Unmarshal([]byte(requestJSON), &request))
	require.NoError(t, rest.ValidatePaymentOrderChainRequest(request))

	require.Len(t, request.Orders, 1)
	require.Nil(t, request.Transfer)
}

func TestPaymentOrderChainRequestValidateWithTransfer(t *testing.T) {
	t.Parallel()

	requestJSON := `{"orders":[{"action":"BUY","base":"USDT","quote":"BTC","type":{"name":"MARKET","base":"USDT","quantity":"100"}}],"transfer":{"type":"TRANSFER","recipientType":"REMOTE","recipient":"abc123","symbol":"BTC","quantity":"1"}}`

	var request rest.PaymentOrderChainRequest
	require.NoError(t, json.Unmarshal([]byte(requestJSON), &request))
	require.NoError(t, rest.ValidatePaymentOrderChainRequest(request))

	require.Len(t, request.Orders, 1)
	require.NotNil(t, request.Transfer)
}

func TestPaymentOrderChainRequestValidateIgnoresLegacyPolicyFields(t *testing.T) {
	t.Parallel()

	requestJSON := `{"orders":[{"action":"BUY","base":"USDT","quote":"BTC","type":{"name":"MARKET","base":"USDT","quantity":"100"}}],"maxSlippageBps":120,"maxExecutionDelaySec":300,"allowReprice":true,"maxRepriceAttempts":2}`

	var request rest.PaymentOrderChainRequest
	require.NoError(t, json.Unmarshal([]byte(requestJSON), &request))
	require.NoError(t, rest.ValidatePaymentOrderChainRequest(request))

	require.Len(t, request.Orders, 1)
}
