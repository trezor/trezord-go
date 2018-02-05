PLATFORM  = linux 
BITS      = 32
NAME32 = trezord-go-windows-4.0-386.exe
VOL_MOUNT = -v $(shell pwd):/release
IMAGETAG  = trezord-go-build-env-$(PLATFORM)

all: .package

clean:
	$(info Cleaning...)
	rm -rf build

.binary:
	$(info Building with xgo ...)
	mkdir -p build
	xgo -targets=windows/386 ../..
	mv -f $(NAME32) build/trezord.exe
	cp ../../VERSION build

.package: .binary .docker-image
	$(info Packaging ...)
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /release/release.sh win32

.docker-image:
	$(info Preparing docker image ...)
	docker build -t $(IMAGETAG) .

shell: .docker-image
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /bin/bash