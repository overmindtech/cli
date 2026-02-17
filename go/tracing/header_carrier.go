package tracing

import "github.com/nats-io/nats.go"

// HeaderCarrier is a custom wrapper on top of nats.Headers for otel's TextMapCarrier.
type HeaderCarrier struct {
	headers nats.Header
}

// NewNatsHeaderCarrier creates a new HeaderCarrier.
func NewNatsHeaderCarrier(h nats.Header) *HeaderCarrier {
	return &HeaderCarrier{
		headers: h,
	}
}

func (c *HeaderCarrier) Get(key string) string {
	return c.headers.Get(key)
}

func (c *HeaderCarrier) Set(key, value string) {
	c.headers.Set(key, value)
}

func (c *HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for key := range c.headers {
		keys = append(keys, key)
	}
	return keys
}
