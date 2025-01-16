package xpc

/*
#import "session.h"

#cgo CFLAGS: -x objective-c
*/
import "C"
import (
	"errors"
	"reflect"
	"unsafe"
)

type Session struct {
	sess unsafe.Pointer
	q    unsafe.Pointer
}

// NewSession opens a new session with the given XPC service. This service
// must be a Mach service name -- that is, it should be a service managed by
// launchd. You need to [Close] the session when you're done with it.
func NewSession(service string) (*Session, error) {
	cname := C.CString(service)
	defer C.free(unsafe.Pointer(cname))

	res := C.new_session(cname)
	if res.err != nil {
		defer C.xpc_release((C.xpc_object_t)(res.err))
		return nil, newRichError(unsafe.Pointer(res.err))
	}
	return &Session{
		sess: unsafe.Pointer(res.session),
		q:    unsafe.Pointer(res.queue),
	}, nil
}

// Send sends a message to the session without waiting for a reply. This should
// be used by clients exclusively. See [Reply] for the server side.
func Send[In any](s *Session, msg In) error {
	// Despite xpc_session_send_message's 2nd argument being an xpc_object_t,
	// it actually expect an XPC dictionary. [marshal] doesn't support maps, so
	// just check if msg is a struct.
	// See here: https://developer.apple.com/documentation/xpc/xpc_session_send_message?language=objc
	if !isStruct(msg) {
		return errors.New("msg must be a struct")
	}

	payload, err := Marshal(msg)
	if err != nil {
		return err
	}
	// TODO(aker): do we need to walk the payload to free all the objects?
	defer C.xpc_release(payload)

	xpcErr := C.xpc_session_send_message((C.xpc_session_t)(s.sess), payload)
	if xpcErr != nil {
		defer C.xpc_release(xpcErr)
		return newRichError(unsafe.Pointer(xpcErr))
	}

	return nil
}

func Reply[Out any](s *Session, original unsafe.Pointer, msg Out) error {
	if !isStruct(msg) {
		return errors.New("msg must be a struct")
	}

	payload := C.xpc_dictionary_create_reply((C.xpc_object_t)(original))
	if err := marshalIntoDict(payload, msg); err != nil {
		return err
	}

	xpcErr := C.xpc_session_send_message((C.xpc_session_t)(s.sess), payload)
	if xpcErr != nil {
		defer C.xpc_release(xpcErr)
		return newRichError(unsafe.Pointer(xpcErr))
	}

	return nil
}

// SendWaitReply sends a message to the session and waits for a reply. This
// should be used by clients exclusively. See [Reply] for the server side.
func SendWaitReply[In any, Out any](s *Session, msg In) (Out, error) {
	var out Out
	// Despite xpc_session_send_message_with_reply_sync's 2nd argument being
	// an xpc_object_t, it actually expect an XPC dictionary. [marshal] doesn't
	// support maps, so just check if msg is a struct.
	// See here: https://developer.apple.com/documentation/xpc/xpc_session_send_message_with_reply_sync?language=objc
	if !isStruct(msg) {
		return out, errors.New("msg must be a struct")
	}

	payload, err := Marshal(msg)
	if err != nil {
		return out, err
	}
	defer C.xpc_release(payload)

	res := C.send_message_with_reply((C.xpc_session_t)(s.sess), payload)
	if res.err != nil {
		defer C.xpc_release(res.err)
		return out, newRichError(unsafe.Pointer(res.err))
	}
	defer C.xpc_release(res.reply)

	if err := Unmarshal(unsafe.Pointer(res.reply), &out); err != nil {
		return out, err
	}

	return out, nil
}

func isStruct(v any) bool {
	return reflect.TypeOf(v).Kind() == reflect.Struct
}

// Close closes the session and releases all associated resources. You must
// call this when you're done with the session.
func (s *Session) Close() {
	if s.sess == nil {
		return
	}

	C.xpc_session_cancel((C.xpc_session_t)(s.sess))
	C.xpc_release((C.xpc_object_t)(s.sess))
	C.dispatch_release((C.dispatch_queue_t)(s.q))
}
