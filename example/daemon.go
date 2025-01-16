package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"unsafe"

	"github.com/akerouanton/go-xpc/pkg/xpc"
)

func runDaemon() error {
	var requirement string
	flag.StringVar(&requirement, "requirement", "", "code signing requirement that the daemon should enforce")
	flag.Parse()

	srv, err := xpc.NewServer(
		xpc.Listener{Name: "com.foobar.daemon.ping", Requirement: requirement, Handler: handlePing},
		xpc.Listener{Name: "com.foobar.daemon.add", Requirement: requirement, Handler: handleAddRequest},
		// This listener will simulate a panicking daemon, to test how the
		// client handles that error mode.
		xpc.Listener{Name: "com.foobar.daemon.panic", Requirement: requirement, Handler: handlePanic},
	)
	if err != nil {
		panic(fmt.Errorf("error creating server: %w", err))
	}

	go func() {
		fmt.Println("Starting XPC server...")
		srv.Run()
		fmt.Println("XPC server stopped")
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	fmt.Println("Waiting for interrupt signal")
	<-sigCh

	fmt.Println("Received interrupt signal -- stopping XPC server")
	srv.Close()

	return nil
}

type greetings struct {
	Message string
}

func handlePing(session *xpc.Session, msg unsafe.Pointer) {
	var req greetings
	if err := xpc.Unmarshal(msg, &req); err != nil {
		fmt.Printf("com.foobar.daemon.ping: unmarshal err: %+v\n", err)
		return
	}

	fmt.Printf("[ping] Received message: %s\n", req.Message)
	xpc.Reply(session, msg, greetings{Message: "pong"})
}

func handleAddRequest(session *xpc.Session, msg unsafe.Pointer) {
	var req AddRequest
	if err := xpc.Unmarshal(msg, &req); err != nil {
		fmt.Printf("com.foobar.daemon.add: unmarshal err: %+v\n", err)
		return
	}
	fmt.Printf("New message received: %+v\n", req)

	resp := req.FirstNumber + req.SecondNumber
	fmt.Printf("Sending response: %+v\n", resp)

	xpc.Reply(session, msg, AddResponse{Result: resp})
}

func handlePanic(session *xpc.Session, msg unsafe.Pointer) {
	var req PanicRequest
	if err := xpc.Unmarshal(msg, &req); err != nil {
		fmt.Printf("com.foobar.daemon.panic: unmarshal err: %+v\n", err)
		return
	}

	if req.Panic {
		panic("panic")
	}

	xpc.Reply(session, msg, PanicResponse{Message: "didn't panic"})
}
