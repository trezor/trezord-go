PLATFORM  = win
BITS      = 32
NAME32 = trezord-go-windows-4.0-386.exe
VOL_MOUNT = -v $(shell pwd):/release
IMAGETAG  = trezord-go-build-env-$(PLATFORM)

WDI_VOL_MOUNT = -v $(shell pwd):/release
WDI_IMAGETAG  = trezord-go-wdi-build

ifeq ($(LOCAL_BUILD),1)
IMPORT_PATH = ../..
else
IMPORT_PATH = github.com/trezor/trezord-go
endif

all: clean .package

clean:
	$(info Cleaning...)
	rm -rf build

.binary:
	$(info Building with xgo ...)
	mkdir -p build
	xgo -ldflags="-H=windowsgui" -targets=windows/386 $(IMPORT_PATH)
	mv -f $(NAME32) build/trezord.exe
	cp ../../VERSION build

.package: .binary .docker-image .wdi
	$(info Packaging ...)
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /release/release.sh $(PLATFORM)$(BITS)

.docker-image: Dockerfile
	$(info Preparing docker image ...)
	docker build -t $(IMAGETAG) .

shell: .docker-image
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /bin/bash

.wdi: Dockerfile.wdi_driver
	$(info Building WDI driver installer in docker)
	docker build -t $(WDI_IMAGETAG) -f Dockerfile.wdi_driver .
	docker run -i -t $(WDI_VOL_MOUNT) $(WDI_IMAGETAG) /bin/bash -c 'cp wdi-simple-* /release/build'
