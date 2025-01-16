package xpc

/*
#import "listener.h"

#cgo CFLAGS: -x objective-c
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime/cgo"
	"unsafe"
)

type listener struct {
	l        unsafe.Pointer
	q        unsafe.Pointer
	ch       chan Message
	chHandle cgo.Handle
	cb       Handler
}

type Message struct {
	Msg  unsafe.Pointer
	Peer unsafe.Pointer
}

func (m Message) Release() {
	C.xpc_release((C.xpc_object_t)(m.Msg))
	C.xpc_release((C.xpc_object_t)(m.Peer))
}

func newListener(name, requirement string, cb Handler) (listener, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	var crequirement *C.char
	if requirement != "" {
		crequirement = C.CString(requirement)
		defer C.free(unsafe.Pointer(crequirement))
	}

	ch := make(chan Message)
	chHandle := cgo.NewHandle(ch)

	res := C.new_listener(cname, crequirement, C.uintptr_t(chHandle))
	switch res.err_code {
	case 0:
		// No errors
	case C.XPC_LISTENER_CREATE_FAILED:
		chHandle.Delete()
		defer C.xpc_release((C.xpc_object_t)(res.err))
		return listener{}, newRichError(unsafe.Pointer(res.err))
	case C.XPC_LISTENER_SET_PEER_CODE_SIGNING_REQUIREMENT_FAILED:
		chHandle.Delete()
		return listener{}, errors.New("failed to set code signing requirement")
	case C.XPC_LISTENER_ACTIVATE_FAILED:
		chHandle.Delete()
		defer C.xpc_release((C.xpc_object_t)(res.err))
		return listener{}, fmt.Errorf("failed to activate listener: %w", newRichError(unsafe.Pointer(res.err)))
	default:
		chHandle.Delete()
		return listener{}, fmt.Errorf("unknown error code: %d", res.err_code)
	}

	return listener{
		l:        unsafe.Pointer(res.listener),
		q:        unsafe.Pointer(res.queue),
		ch:       ch,
		chHandle: chHandle,
		cb:       cb,
	}, nil
}

func (l *listener) run() {
	for msg := range l.ch {
		sess := Session{sess: unsafe.Pointer(msg.Peer)}
		l.cb(&sess, msg.Msg)
		msg.Release()
	}
}

func (l *listener) Close() error {
	C.xpc_listener_cancel((C.xpc_listener_t)(l.l))
	// According to [1], xpc_release must be called when it's no longer needed.
	//
	// [1]: https://developer.apple.com/documentation/xpc/xpc_listener_create?language=objc
	C.xpc_release((C.xpc_object_t)(l.l))
	if l.q != nil {
		C.dispatch_release((C.dispatch_queue_t)(l.q))
	}
	l.chHandle.Delete()
	close(l.ch)
	return nil
}

//export on_msg_recv
func on_msg_recv(h C.uintptr_t, peer C.xpc_session_t, msg C.xpc_object_t) {
	ch := cgo.Handle(h).Value().(chan Message)

	C.xpc_retain(msg)
	C.xpc_retain(peer)

	ch <- Message{
		Peer: unsafe.Pointer(peer),
		Msg:  unsafe.Pointer(msg),
	}
}
