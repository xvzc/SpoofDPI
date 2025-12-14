package ptr

func Clone[T any](x *T) *T {
	if x == nil {
		return nil
	}
	// dereference and create a new pointer to the value
	v := *x
	return &v
}

func CloneOr[T any](x *T, fallback *T) *T {
	if x == nil {
		return Clone(fallback)
	}

	return Clone(x)
}

func CloneSlice[T any](x []T) []T {
	if x == nil {
		return nil
	}

	return append([]T(nil), x...)
}

func CloneSliceOr[T any](x []T, fallback []T) []T {
	if x == nil {
		return CloneSlice(fallback)
	}

	return CloneSlice(x)
}

func FromValue[T any](v T) *T {
	return &v
}

func Empty[T any]() T {
	var zero T
	return zero
}

func FromPtr[T any](x *T) T {
	if x == nil {
		return Empty[T]()
	}

	return *x
}

func FromPtrOr[T any](x *T, v T) T {
	if x == nil {
		return v
	}

	return *x
}
