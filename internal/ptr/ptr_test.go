package ptr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClone(t *testing.T) {
	val := 10
	tcs := []struct {
		name   string
		input  *int
		assert func(t *testing.T, output *int)
	}{
		{
			name:  "nil input",
			input: nil,
			assert: func(t *testing.T, output *int) {
				assert.Nil(t, output)
			},
		},
		{
			name:  "non-nil input",
			input: &val,
			assert: func(t *testing.T, output *int) {
				assert.NotNil(t, output)
				assert.Equal(t, val, *output)
				assert.NotSame(t, &val, output)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := Clone(tc.input)
			tc.assert(t, output)
		})
	}
}

func TestCloneOr(t *testing.T) {
	val := 10
	fallbackVal := 5

	tcs := []struct {
		name     string
		input    *int
		fallback *int
		assert   func(t *testing.T, output *int)
	}{
		{
			name:     "nil input with fallback",
			input:    nil,
			fallback: &fallbackVal,
			assert: func(t *testing.T, output *int) {
				assert.NotNil(t, output)
				assert.Equal(t, fallbackVal, *output)
				assert.NotSame(t, &fallbackVal, output)
			},
		},
		{
			name:     "non-nil input",
			input:    &val,
			fallback: &fallbackVal,
			assert: func(t *testing.T, output *int) {
				assert.Equal(t, val, *output)
				assert.NotSame(t, &val, output)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := CloneOr(tc.input, tc.fallback)
			tc.assert(t, output)
		})
	}
}

func TestCloneSlice(t *testing.T) {
	tcs := []struct {
		name   string
		input  []int
		assert func(t *testing.T, output []int)
	}{
		{
			name:  "nil input",
			input: nil,
			assert: func(t *testing.T, output []int) {
				assert.Nil(t, output)
			},
		},
		{
			name:  "non-nil input",
			input: []int{1, 2, 3},
			assert: func(t *testing.T, output []int) {
				assert.Equal(t, []int{1, 2, 3}, output)
				assert.NotSame(t, &[]int{1, 2, 3}[0], &output[0])
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := CloneSlice(tc.input)
			tc.assert(t, output)
		})
	}
}

func TestFromValue(t *testing.T) {
	tcs := []struct {
		name   string
		input  int
		assert func(t *testing.T, output *int)
	}{
		{
			name:  "value",
			input: 10,
			assert: func(t *testing.T, output *int) {
				assert.NotNil(t, output)
				assert.Equal(t, 10, *output)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := FromValue(tc.input)
			tc.assert(t, output)
		})
	}
}

func TestFromPtr(t *testing.T) {
	val := 10
	tcs := []struct {
		name   string
		input  *int
		assert func(t *testing.T, output int)
	}{
		{
			name:  "nil input",
			input: nil,
			assert: func(t *testing.T, output int) {
				assert.Equal(t, 0, output)
			},
		},
		{
			name:  "non-nil input",
			input: &val,
			assert: func(t *testing.T, output int) {
				assert.Equal(t, val, output)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := FromPtr(tc.input)
			tc.assert(t, output)
		})
	}
}

func TestFromPtrOr(t *testing.T) {
	val := 10
	tcs := []struct {
		name     string
		input    *int
		fallback int
		assert   func(t *testing.T, output int)
	}{
		{
			name:     "nil input",
			input:    nil,
			fallback: 100,
			assert: func(t *testing.T, output int) {
				assert.Equal(t, 100, output)
			},
		},
		{
			name:     "non-nil input",
			input:    &val,
			fallback: 100,
			assert: func(t *testing.T, output int) {
				assert.Equal(t, val, output)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := FromPtrOr(tc.input, tc.fallback)
			tc.assert(t, output)
		})
	}
}
