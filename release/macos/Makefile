PLATFORM  = macos
BITS      = 64
NAME64 = trezord-go-darwin-10.7-amd64
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
	xgo -ldflags="-X 'main.githash=$(GITHASH)'" -targets=darwin-10.7/amd64 $(IMPORT_PATH)
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
