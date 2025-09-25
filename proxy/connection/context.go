package connection

import (
	"context"
	"errors"
)

// contextKey is a type for context keys to avoid collisions
type contextKey struct{ name string }

var (
	// EnhancedConnContextKey is the context key for EnhancedConn
	EnhancedConnContextKey = &contextKey{"enhanced-conn"}

	// TLSConnContextKey is the context key for TLSConn forward
	TLSConnContextKey = &contextKey{"tls-conn"}
)

// GetEnhancedConnFromContext retrieves the EnhancedConn from the context
func GetEnhancedConnFromContext(ctx context.Context) (*EnhancedConn, error) {
	val := ctx.Value(EnhancedConnContextKey)
	if val == nil {
		return nil, errors.New("context not found")
	}

	proxyConn, ok := val.(*EnhancedConn)
	if !ok {
		return nil, errors.New("val must be enhanced conn")
	}

	return proxyConn, nil
}

// MustGetEnhancedConnFromContext retrieves the EnhancedConn from the context and panics if not found
func MustGetEnhancedConnFromContext(ctx context.Context) *EnhancedConn {
	proxyConn, err := GetEnhancedConnFromContext(ctx)
	if err != nil {
		panic(err)
	}

	return proxyConn
}
