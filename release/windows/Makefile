PLATFORM  = win
BITS      = 32
NAME32 = onekey-bridge-windows-4.0-386.exe
NAME64 = onekey-bridge-windows-4.0-amd64.exe
VOL_MOUNT = -v $(shell pwd):/release
IMAGETAG  = onekey-bridge-build-env-$(PLATFORM)

WDI_VOL_MOUNT = -v $(shell pwd):/release
WDI_IMAGETAG  = onekey-bridge-wdi-build

DEVCON_VOL_MOUNT = -v $(shell pwd):/release
DEVCON_IMAGETAG  = onekey-bridge-devcon-build

IMPORT_PATH = ../..

all: clean .package

clean:
	$(info Cleaning...)
	rm -rf build

.binary:
	$(info Building with xgo ...)
	mkdir -p build
	xgo -ldflags="-H=windowsgui" -targets=windows/386 $(IMPORT_PATH)
	mv -f $(NAME32) build/onekeyd-32b.exe
	xgo -ldflags="-H=windowsgui" -targets=windows/amd64 $(IMPORT_PATH)
	mv -f $(NAME64) build/onekeyd-64b.exe
	cp ../../VERSION build

.package: .binary .docker-image .wdi .devcon
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

.devcon:
	$(info Building devcon in docker)
	docker build -t $(DEVCON_IMAGETAG) -f Dockerfile.devcon .
	docker run -i -t $(DEVCON_VOL_MOUNT) $(DEVCON_IMAGETAG) /bin/bash -c 'cp /devcon/build-i686/devcon.exe /release/build/devcon-32b.exe'
	docker run -i -t $(DEVCON_VOL_MOUNT) $(DEVCON_IMAGETAG) /bin/bash -c 'cp /devcon/build-x86_64/devcon.exe /release/build/devcon-64b.exe'
