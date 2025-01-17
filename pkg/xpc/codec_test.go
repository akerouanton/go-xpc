package xpc

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "nil value",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "bool true",
			input:   true,
			wantErr: false,
		},
		{
			name:    "bool false",
			input:   false,
			wantErr: false,
		},
		{
			name:    "int",
			input:   42,
			wantErr: false,
		},
		{
			name:    "int64",
			input:   int64(9223372036854775807),
			wantErr: false,
		},
		{
			name:    "uint",
			input:   uint(42),
			wantErr: false,
		},
		{
			name:    "float64",
			input:   3.14159,
			wantErr: false,
		},
		{
			name:    "string",
			input:   "hello world",
			wantErr: false,
		},
		{
			name:    "array",
			input:   [3]int{1, 2, 3},
			wantErr: false,
		},
		{
			name: "struct",
			input: struct {
				Name    string  `xpc:"name"`
				Age     int     `xpc:"age"`
				Balance float64 `xpc:"balance"`
			}{
				Name:    "John Doe",
				Age:     30,
				Balance: 100.50,
			},
			wantErr: false,
		},
		{
			name: "nested struct",
			input: struct {
				User struct {
					Name string `xpc:"name"`
					Age  int    `xpc:"age"`
				} `xpc:"user"`
				Active bool `xpc:"active"`
			}{
				User: struct {
					Name string `xpc:"name"`
					Age  int    `xpc:"age"`
				}{
					Name: "John Doe",
					Age:  30,
				},
				Active: true,
			},
			wantErr: false,
		},
		{
			name:    "unsupported type",
			input:   make(chan int),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tt.input != nil {
					assert.NotNil(t, result)
				}
			}

			// For non-error cases, verify the type of the XPC object
			if !tt.wantErr && tt.input != nil {
				switch tt.input.(type) {
				case bool:
					assert.Equal(t, "bool", getXPCType(result))
				case int, int8, int16, int32, int64:
					assert.Equal(t, "int64", getXPCType(result))
				case uint, uint8, uint16, uint32, uint64:
					assert.Equal(t, "uint64", getXPCType(result))
				case float32, float64:
					assert.Equal(t, "double", getXPCType(result))
				case string:
					assert.Equal(t, "string", getXPCType(result))
				case [3]int:
					assert.Equal(t, "array", getXPCType(result))
				case struct{}:
					// Structs are marshaled as dictionaries
					assert.Equal(t, "dictionary", getXPCType(result))
				}
			}
		})
	}
}

// TestUnmarshalRoundTrip tests marshaling and unmarshaling of various types
func TestUnmarshalRoundTrip(t *testing.T) {
	type User struct {
		Name string `xpc:"name"`
		Age  int    `xpc:"age"`
	}

	type CustomError struct {
		Err error
		Val int
	}

	tests := []struct {
		name    string
		input   interface{}
		target  interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name:   "bool true",
			input:  true,
			target: new(bool),
			want:   true,
		},
		{
			name:   "bool false",
			input:  false,
			target: new(bool),
			want:   false,
		},
		{
			name:   "int",
			input:  42,
			target: new(int),
			want:   42,
		},
		{
			name:   "int64",
			input:  int64(9223372036854775807),
			target: new(int64),
			want:   int64(9223372036854775807),
		},
		{
			name:   "uint",
			input:  uint(42),
			target: new(uint),
			want:   uint(42),
		},
		{
			name:   "float64",
			input:  3.14159,
			target: new(float64),
			want:   3.14159,
		},
		{
			name:   "string",
			input:  "hello world",
			target: new(string),
			want:   "hello world",
		},
		{
			name:   "array",
			input:  [3]int{1, 2, 3},
			target: new([3]int),
			want:   [3]int{1, 2, 3},
		},
		{
			name: "struct",
			input: struct {
				Name    string  `xpc:"name"`
				Age     int     `xpc:"age"`
				Balance float64 `xpc:"balance"`
			}{
				Name:    "John Doe",
				Age:     30,
				Balance: 100.50,
			},
			target: new(struct {
				Name    string  `xpc:"name"`
				Age     int     `xpc:"age"`
				Balance float64 `xpc:"balance"`
			}),
			want: struct {
				Name    string  `xpc:"name"`
				Age     int     `xpc:"age"`
				Balance float64 `xpc:"balance"`
			}{
				Name:    "John Doe",
				Age:     30,
				Balance: 100.50,
			},
		},
		{
			name: "nested struct",
			input: struct {
				User struct {
					Name string `xpc:"name"`
					Age  int    `xpc:"age"`
				} `xpc:"user"`
				Active bool `xpc:"active"`
			}{
				User: User{
					Name: "John Doe",
					Age:  30,
				},
				Active: true,
			},
			target: new(struct {
				User   User `xpc:"user"`
				Active bool `xpc:"active"`
			}),
			want: struct {
				User   User `xpc:"user"`
				Active bool `xpc:"active"`
			}{
				User: User{
					Name: "John Doe",
					Age:  30,
				},
				Active: true,
			},
		},
		{
			name: "struct with nil value",
			input: struct {
				Name *string `xpc:"name"`
			}{},
			target: new(struct {
				Name *string `xpc:"name"`
			}),
			want: struct {
				Name *string `xpc:"name"`
			}{},
		},
		{
			name: "struct with pointers",
			input: struct {
				Name *string `xpc:"name"`
			}{
				Name: strPtr("John Doe"),
			},
			target: new(struct {
				Name *string `xpc:"name"`
			}),
			want: struct {
				Name *string `xpc:"name"`
			}{
				Name: strPtr("John Doe"),
			},
		},
		{
			name: "embedded struct",
			input: struct {
				User
			}{
				User: User{
					Name: "John Doe",
					Age:  30,
				},
			},
			target: new(struct {
				User
			}),
			want: struct {
				User
			}{
				User{
					Name: "John Doe",
					Age:  30,
				},
			},
		},
		{
			name: "struct with unexpected field",
			input: struct {
				name string
			}{},
			target: new(struct {
				name string
			}),
			want: struct {
				name string
			}{},
		},
		{
			name:   "net.IP",
			input:  net.ParseIP("192.168.1.1"),
			target: new(net.IP),
			want:   net.ParseIP("192.168.1.1"),
		},
		{
			name:   "net.IP field",
			input:  struct{ IP net.IP }{IP: net.ParseIP("192.168.1.1")},
			target: new(struct{ IP net.IP }),
			want:   struct{ IP net.IP }{IP: net.ParseIP("192.168.1.1")},
		},
		{
			name: "struct with error",
			input: struct {
				Error error
			}{
				Error: fmt.Errorf("test error"),
			},
			target: new(struct {
				Error error
			}),
			want: struct {
				Error error
			}{
				Error: fmt.Errorf("test error"),
			},
		},
		{
			name: "struct with custom error",
			input: struct {
				Error CustomError
			}{
				Error: CustomError{
					Err: fmt.Errorf("test error"),
					Val: 10,
				},
			},
			target: new(struct {
				Error CustomError
			}),
			want: struct {
				Error CustomError
			}{
				Error: CustomError{
					Err: fmt.Errorf("test error"),
					Val: 10,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// First marshal the input
			xpcObj, err := Marshal(tc.input)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Then unmarshal into the target
			err = Unmarshal(unsafe.Pointer(xpcObj), tc.target)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, reflect.ValueOf(tc.target).Elem().Interface())
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestUnmarshalFDRoundTrip(t *testing.T) {
	f, err := os.CreateTemp(os.TempDir(), "xpc-marshal-fd")
	require.NoError(t, err)
	defer f.Close()

	// First marshal the input
	xpcObj, err := Marshal(struct {
		FD FD
	}{
		FD: FD(f.Fd()),
	})
	require.NoError(t, err)

	// Then unmarshal into the target
	dst := struct {
		FD FD
	}{}
	err = Unmarshal(unsafe.Pointer(xpcObj), &dst)
	assert.NoError(t, err)

	dstFile := dst.FD.File()
	_, err = dstFile.WriteString("hello")
	assert.NoError(t, err)
	assert.NoError(t, dstFile.Close())

	f.Seek(0, 0)
	r := bufio.NewReader(f)
	line, _, err := r.ReadLine()
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(line))
}

// TestUnmarshalErrors tests error cases
func TestUnmarshalErrors(t *testing.T) {
	// Create a simple value to marshal
	xpcObj, _ := Marshal(true)

	tests := []struct {
		name    string
		target  interface{}
		wantErr string
	}{
		{
			name:    "nil target",
			target:  nil,
			wantErr: "unmarshal target must be a non-nil pointer",
		},
		{
			name:    "non-pointer target",
			target:  true,
			wantErr: "unmarshal target must be a non-nil pointer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal(unsafe.Pointer(xpcObj), tt.target)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
