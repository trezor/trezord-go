native:
	CGO_ENABLED=1 go build -buildvcs=false $(GOFLAGS)

build-release:
	make -C release clean all
