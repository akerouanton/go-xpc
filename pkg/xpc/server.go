package xpc

/*
#import <xpc/xpc.h>

bool is_code_signing_requirement_available() {
	// See https://developer.apple.com/library/archive/releasenotes/AppKit/RN-AppKit/index.html
	if (@available(macOS 14.4, *)) {
		return true;
	}
	return false;
}

#cgo CFLAGS: -x objective-c
*/
import "C"
import (
	"errors"
	"sync"
	"unsafe"
)

// IsCodeSigningRequirementAvailable returns true if the current system is
// recent enough to support specifying code signing requirements on an XPC
// session.
func IsCodeSigningRequirementAvailable() bool {
	return bool(C.is_code_signing_requirement_available())
}

// Server is a high-level wrapper around multiple listeners. It will start
// processing messages for all listeners when calling [Server.Run]. You must
// call [Server.Close] to stop processing messages and free resources.
type Server struct {
	listeners []listener
	wg        sync.WaitGroup
}

// Listener represents a listener for an XPC service. The service Name must be
// a Mach service name -- that is, it should be a service managed by launchd.
// The Requirement string is optional and can be used to specify a code signing
// requirement. You can check whether the running system supports putting code
// signing requirements on XPC services by calling [IsCodeSigningRequirementAvailable].
// The listener will call the Handler function whenever a new message is
// received.
//
// To know more about code signing requirements, see the following documentation:
//
// [Code Signing Requirement Language](https://developer.apple.com/library/archive/documentation/Security/Conceptual/CodeSigningGuide/RequirementLang/RequirementLang.html)
// [TN3127: Inside Code Signing: Requirements](https://developer.apple.com/documentation/technotes/tn3127-inside-code-signing-requirements)
type Listener struct {
	Name        string
	Requirement string
	Handler     Handler
}

type Handler func(session *Session, msg unsafe.Pointer)

func NewServer(listeners ...Listener) (_ *Server, retErr error) {
	ls := make([]listener, len(listeners))
	for i, listener := range listeners {
		var err error
		ls[i], err = newListener(listener.Name, listener.Requirement, listener.Handler)
		if err != nil {
			return nil, err
		}

		defer func() {
			if retErr != nil {
				ls[i].Close()
			}
		}()
	}

	return &Server{
		listeners: ls,
		wg:        sync.WaitGroup{},
	}, nil
}

func (s *Server) Run() {
	for _, listener := range s.listeners {
		listener := listener
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			listener.run()
		}()
	}
	s.wg.Wait()
}

func (s *Server) Close() error {
	errs := make([]error, 0, len(s.listeners))
	for _, listener := range s.listeners {
		errs = append(errs, listener.Close())
	}
	return errors.Join(errs...)
}
