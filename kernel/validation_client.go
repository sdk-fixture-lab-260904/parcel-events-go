package kernel

import "io"

// WithValidation controls structural request and response checking for a client.
func WithValidation(enabled bool) ClientOption { return func(c *Client) { c.validation = enabled } }

// WithRequestValidation overrides validation for one operation and its pages.
func WithRequestValidation(enabled bool) RequestOption {
	return func(r *requestConfig) { r.validation = &enabled }
}

// WithBodyContract binds a generated request-body schema.
func WithBodyContract(contract Contract) RequestOption {
	return func(r *requestConfig) { r.bodyContract = contract }
}

// WithResponseContract binds a generated response-body schema.
func WithResponseContract(contract Contract) RequestOption {
	return func(r *requestConfig) { r.responseContract = contract }
}

// WithEventContract binds a generated streamed-event schema.
func WithEventContract(contract Contract) RequestOption {
	return func(r *requestConfig) { r.eventContract = contract }
}

func (c *Client) validationEnabled(config *requestConfig) bool {
	if config.validation != nil {
		return *config.validation
	}
	return c.validation
}

// ValidateValue checks operation parameters before their wire serialization.
func (c *Client) ValidateValue(value any, contract Contract, options ...RequestOption) error {
	if !c.validationEnabled(newRequestConfig(options)) {
		return nil
	}
	return Validate(value, contract)
}

type validatedStreamBody struct {
	io.ReadCloser
	eventContract Contract
}

func (c *Client) streamBody(body io.ReadCloser, config *requestConfig) io.ReadCloser {
	if !c.validationEnabled(config) || config.eventContract.Schema == nil {
		return body
	}
	return &validatedStreamBody{body, config.eventContract}
}
