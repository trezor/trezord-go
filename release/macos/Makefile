PLATFORM  = macos
BITS      = 64
NAME64 = trezord-go-darwin-10.6-amd64
VOL_MOUNT = -v $(shell pwd):/release
IMAGETAG  = trezord-go-build-env-$(PLATFORM)

all: clean .package

clean:
	$(info Cleaning...)
	rm -rf build

.binary:
	$(info Building with xgo ...)
	mkdir -p build
	xgo -targets=darwin/amd64 github.com/trezor/trezord-go
	mv -f $(NAME64) build/trezord
	cp ../../VERSION build

.package: .binary .docker-image
	$(info Packaging ...)
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /release/release.sh $(PLATFORM)$(BITS)

.docker-image:
	$(info Preparing docker image ...)
	docker build -t $(IMAGETAG) .

shell: .docker-image
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /bin/bash
