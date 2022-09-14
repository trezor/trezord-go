PLATFORM  = win
BITS      = 32
VOL_MOUNT = -v $(shell pwd):/release
IMAGETAG  = trezord-go-build-env-$(PLATFORM)

IMPORT_PATH = $(shell realpath ../..)

all: clean .package

clean:
	$(info Cleaning...)
	rm -rf build

.binary:
	$(info Building with crossbuild ...)
	mkdir -p build
	docker run --rm \
		-v $(IMPORT_PATH):/trezord \
		-w /trezord \
		-e CGO_ENABLED=1 \
		docker.elastic.co/beats-dev/golang-crossbuild:1.19-main-debian10 \
		--build-cmd "go build -o release/windows/build/trezord-64b.exe -ldflags=\"-H=windowsgui\"" \
		-p "windows/amd64"
	docker run --rm \
		-v $(IMPORT_PATH):/trezord \
		-w /trezord \
		-e CGO_ENABLED=1 \
		-e CGO_CFLAGS="-D_WIN32_WINNT=0x0600" \
		docker.elastic.co/beats-dev/golang-crossbuild:1.19-main-debian10 \
		--build-cmd "go build -o release/windows/build/trezord-32b.exe -ldflags=\"-H=windowsgui\"" \
		-p "windows/386"
	cp ../../VERSION build

.package: .binary .docker-image
	$(info Packaging ...)
	docker run --rm  $(VOL_MOUNT) $(IMAGETAG) /release/release.sh $(PLATFORM)$(BITS)

.docker-image: Dockerfile
	$(info Preparing docker image ...)
	docker build -t $(IMAGETAG) .

shell: .docker-image
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /bin/bash
