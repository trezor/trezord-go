PLATFORM  = linux
NAME64 = trezord-go-linux-amd64
NAME32 = trezord-go-linux-386
VOL_MOUNT = -v $(shell pwd):/release
IMAGETAG  = trezord-go-build-env-$(PLATFORM)

all: clean .package64 .package32

clean:
	$(info Cleaning...)
	rm -rf build

.binary64:
	$(info Building with xgo ...)
	mkdir -p build
	xgo -targets=linux/amd64 ../..
	mv -f $(NAME64) build/trezord
	cp ../../VERSION build

.package64: .binary64 .docker-image
	$(info Packaging ...)
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /release/release.sh lin64

.binary32:
	$(info Building with xgo ...)
	mkdir -p build
	xgo -targets=linux/386 ../..
	mv -f $(NAME32) build/trezord
	cp ../../VERSION build

.package32: .binary32 .docker-image
	$(info Packaging ...)
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /release/release.sh lin32

.docker-image:
	$(info Preparing docker image ...)
	docker build -t $(IMAGETAG) .

shell: .docker-image
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /bin/bash

privkey:
	gpg --armor --export-secret-key > privkey.asc
