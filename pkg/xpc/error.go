package xpc

import (
	"errors"
	"fmt"
	"unsafe"
)

/*
#include <xpc/xpc.h>
#cgo CFLAGS: -x objective-c
*/
import "C"

type RichError struct {
	Err      error
	CanRetry bool
}

func newRichError(richErr unsafe.Pointer) error {
	if richErr == nil {
		return nil
	}

	canRetry := C.xpc_rich_error_can_retry((C.xpc_rich_error_t)(richErr))
	desc := C.xpc_rich_error_copy_description((C.xpc_rich_error_t)(richErr))
	defer C.free(unsafe.Pointer(desc))

	return RichError{
		Err:      errors.New(C.GoString(desc)),
		CanRetry: bool(canRetry),
	}
}

func (r RichError) Error() string {
	return fmt.Sprintf("%s (canRetry=%t)", r.Err.Error(), r.CanRetry)
}

func (r RichError) Unwrap() error {
	return r.Err
}
