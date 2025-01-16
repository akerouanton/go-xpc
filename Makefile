.PHONY: build-example
build-example:
	./example/build.sh

.PHONY: test
test: build-example
	sudo launchctl start com.foobar.daemon
	go test ./...

.PHONY: clean
clean:
	sudo launchctl stop com.foobar.daemon || true
	sudo launchctl remove com.foobar.daemon || true
	rm -rf example/ExampleDaemon.app
