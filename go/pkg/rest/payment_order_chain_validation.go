package rest

// Validate validates payment order chain request shape.
// Policy fields are no longer part of this request model.
func (r PaymentOrderChainRequest) Validate() error {
	return nil
}

// ValidatePaymentOrderChainRequest validates payment order chain requests.
func ValidatePaymentOrderChainRequest(req PaymentOrderChainRequest) error {
	return req.Validate()
}
