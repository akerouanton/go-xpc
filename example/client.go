package main

import (
	"flag"
	"fmt"

	"github.com/akerouanton/go-xpc/pkg/xpc"
)

type AddRequest struct {
	FirstNumber  int64
	SecondNumber int64
}

type AddResponse struct {
	Result int64
}

func runClient() error {
	var method string
	flag.StringVar(&method, "method", "", "method: ping, add or panic")
	flag.Parse()

	switch method {
	case "ping":
		if err := callPing(); err != nil {
			return err
		}
	case "add":
		if err := callAdd(); err != nil {
			return err
		}
	case "panic":
		if err := callPanic(true); err == nil {
			return fmt.Errorf("expected callPanic to return an error")
		}
		if err := callPanic(false); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid -method %q", method)
	}
	return nil
}

func callPing() error {
	session, err := xpc.NewSession("com.foobar.daemon.ping")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	type greetings struct {
		Message string
	}

	reply, err := xpc.SendWaitReply[greetings, greetings](session, greetings{Message: "hello"})
	if err != nil {
		return err
	}

	if reply.Message != "pong" {
		return fmt.Errorf("expected pong, got %s", reply.Message)
	}
	fmt.Println("ping succeeded")

	return nil
}

func callAdd() error {
	session, err := xpc.NewSession("com.foobar.daemon.add")
	if err != nil {
		return err
	}
	defer session.Close()

	reply, err := xpc.SendWaitReply[AddRequest, AddResponse](session, AddRequest{
		FirstNumber:  1,
		SecondNumber: 2,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(reply.Result)

	return nil
}

type PanicRequest struct {
	Panic bool
}

type PanicResponse struct {
	Message string
}

func callPanic(panic bool) error {
	session, err := xpc.NewSession("com.foobar.daemon.panic")
	if err != nil {
		return err
	}
	defer session.Close()

	reply, err := xpc.SendWaitReply[PanicRequest, PanicResponse](session, PanicRequest{Panic: panic})
	if err != nil {
		return err
	}

	if reply.Message != "didn't panic" {
		return fmt.Errorf(`expected "didn't panic", got %s`, reply.Message)
	}
	fmt.Println(reply.Message)

	return nil
}
