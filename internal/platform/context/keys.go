package context

type contextKey string

func (c contextKey) String() string {
	return "pipeline context key " + string(c)
}

const (
	contextKeyCorrelationId = contextKey("correlation-id")
)
