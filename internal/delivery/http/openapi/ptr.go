package openapi

// Ptr returns a pointer to the given value. Use for optional fields in generated types.
func Ptr[T any](v T) *T { return &v }
