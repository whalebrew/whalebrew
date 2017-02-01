default: build
build:
	go get github.com/mitchellh/gox
	go get github.com/bfirsh/whalebrew/cmd
	go get github.com/inconshreveable/mousetrap
	go get golang.org/x/sys/unix
	mkdir -p build
	cd build; gox -osarch="linux/amd64" -osarch="darwin/amd64" -osarch="windows/amd64" ../; \
	mv whalebrew_linux_amd64 whalebrew-Linux-x86_64; \
	mv whalebrew_darwin_amd64 whalebrew-Darwin-x86_64; \
	mv whalebrew_windows_amd64.exe whalebrew-Windows-x86_64.exe;
test:
	docker build -t whalebrew .;
	docker run --rm whalebrew go test ./...
clean:
	git clean -dxf	
