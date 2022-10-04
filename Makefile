native:
	CGO_ENABLED=1 go build $(GOFLAGS)

build-release:
	make -C release clean all
