PLATFORM  = linux
VOL_MOUNT = -v $(shell pwd):/release

IMAGETAG_PACKAGE  = trezord-go-package-env-$(PLATFORM)
IMAGETAG_BUILD    = trezord-go-build-env-$(PLATFORM)

IMPORT_PATH =  $(shell realpath ../..)

all: clean .package

clean:
	$(info Cleaning...)
	rm -rf build

.binary: .docker-image-build
	$(info Building with musl crosbuild ...)
	mkdir -p build
	docker run --rm -v $(IMPORT_PATH):/trezord -w /trezord $(IMAGETAG_BUILD)-arm64 /trezord/release/linux/build.sh
	docker run --rm -v $(IMPORT_PATH):/trezord -w /trezord $(IMAGETAG_BUILD)-386 /trezord/release/linux/build.sh
	docker run --rm -v $(IMPORT_PATH):/trezord -w /trezord $(IMAGETAG_BUILD)-amd64 /trezord/release/linux/build.sh
	cp ../../VERSION build

.package: .binary .docker-image-package
	$(info Packaging ...)
	docker run --rm $(VOL_MOUNT) $(IMAGETAG_PACKAGE) /release/release.sh linux-386
	docker run --rm $(VOL_MOUNT) $(IMAGETAG_PACKAGE) /release/release.sh linux-amd64
	docker run --rm $(VOL_MOUNT) $(IMAGETAG_PACKAGE) /release/release.sh linux-arm64

.docker-image-build:
	$(info Preparing docker images for build ...)
	docker build -t $(IMAGETAG_BUILD)-arm64 . -f Dockerfile.build --target go-arm64-musl
	docker build -t $(IMAGETAG_BUILD)-amd64 . -f Dockerfile.build --target go-amd64-musl
	docker build -t $(IMAGETAG_BUILD)-386 . -f Dockerfile.build --target go-386-musl

.docker-image-package:
	$(info Preparing docker image ...)
	docker build -t $(IMAGETAG_PACKAGE) . -f Dockerfile.package

shell: .docker-image
	docker run $(VOL_MOUNT) $(IMAGETAG) /bin/bash

privkey:
	gpg --armor --export-secret-key > privkey.asc
