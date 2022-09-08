PLATFORM  = win
BITS      = 32
VOL_MOUNT = -v $(shell pwd):/release
IMAGETAG  = trezord-go-build-env-$(PLATFORM)

WDI_VOL_MOUNT = -v $(shell pwd):/release
WDI_IMAGETAG  = trezord-go-wdi-build

DEVCON_VOL_MOUNT = -v $(shell pwd):/release
DEVCON_IMAGETAG  = trezord-go-devcon-build

IMPORT_PATH = $(shell realpath ../..)

all: clean .package

clean:
	$(info Cleaning...)
	rm -rf build

.binary:
	$(info Building with crossbuild ...)
	mkdir -p build
	docker run -it --rm \
		-v $(IMPORT_PATH):/trezord \
		-w /trezord \
		-e CGO_ENABLED=1 \
		docker.elastic.co/beats-dev/golang-crossbuild:1.19-main-debian10 \
		--build-cmd "go build -o release/windows/build/trezord-64b.exe -ldflags=\"-H=windowsgui\"" \
		-p "windows/amd64"
	docker run -it --rm \
		-v $(IMPORT_PATH):/trezord \
		-w /trezord \
		-e CGO_ENABLED=1 \
		-e CGO_CFLAGS="-D_WIN32_WINNT=0x0600" \
		docker.elastic.co/beats-dev/golang-crossbuild:1.19-main-debian10 \
		--build-cmd "go build -o release/windows/build/trezord-32b.exe -ldflags=\"-H=windowsgui\"" \
		-p "windows/386"
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
