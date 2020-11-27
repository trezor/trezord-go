PLATFORM  = linux
VOL_MOUNT = -v $(shell pwd):/release
IMAGETAG  = trezord-go-build-env-$(PLATFORM)

GITHASH  := $(shell git rev-parse --short HEAD)

IMPORT_PATH = ../..

all: clean .package

clean:
	$(info Cleaning...)
	rm -rf build

.binary:
	$(info Building with xgo ...)
	mkdir -p build
	xgo -ldflags="-X 'main.githash=$(GITHASH)'" -targets=linux/386,linux/amd64,linux/arm64 $(IMPORT_PATH)
	mv -f trezord-go-linux-386 build/trezord-linux-386
	mv -f trezord-go-linux-amd64 build/trezord-linux-amd64
	mv -f trezord-go-linux-arm64 build/trezord-linux-arm64
	cp ../../VERSION build

.package: .binary .docker-image
	$(info Packaging ...)
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /release/release.sh linux-386
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /release/release.sh linux-amd64
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /release/release.sh linux-arm64

.docker-image:
	$(info Preparing docker image ...)
	docker build -t $(IMAGETAG) .

shell: .docker-image
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /bin/bash

privkey:
	gpg --armor --export-secret-key > privkey.asc
