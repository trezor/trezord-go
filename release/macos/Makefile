PLATFORM  = macos
BITS      = 64
VOL_MOUNT = -v $(shell pwd):/release
IMAGETAG  = trezord-go-build-env-$(PLATFORM)

GITHASH  := $(shell git rev-parse --short HEAD)

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
		docker.elastic.co/beats-dev/golang-crossbuild:1.19-darwin-debian10 \
		--build-cmd "go build -o release/macos/build/trezord-amd64" \
		-p "darwin/amd64"
	docker run -it --rm \
		-v $(IMPORT_PATH):/trezord \
		-w /trezord \
		-e CGO_ENABLED=1 \
		docker.elastic.co/beats-dev/golang-crossbuild:1.19-darwin-arm64-debian10 \
		--build-cmd "go build -o release/macos/build/trezord-arm64" \
		-p "darwin/arm64"
	docker run -it --rm \
		-v $(IMPORT_PATH):/trezord \
		-w /trezord \
		-e CGO_ENABLED=1 \
		--entrypoint /trezord/release/macos/build-fat.sh \
		docker.elastic.co/beats-dev/golang-crossbuild:1.19-darwin-arm64-debian10
	cp ../../VERSION build

.package: .binary .docker-image
	$(info Packaging ...)
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /release/release.sh $(PLATFORM)$(BITS)

.docker-image:
	$(info Preparing docker image ...)
	docker build -t $(IMAGETAG) .

shell: .docker-image
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /bin/bash
