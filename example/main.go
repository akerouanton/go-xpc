package main

import (
	"os"
	"path/filepath"
)

func main() {
	binaryName := filepath.Base(os.Args[0])
	switch binaryName {
	case "client":
		if err := runClient(); err != nil {
			panic(err)
		}
	case "com.foobar.daemon":
		if err := runDaemon(); err != nil {
			panic(err)
		}
	default:
		panic("unknown mode")
	}
}
