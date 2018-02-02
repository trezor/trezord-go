PLATFORM  = linux
NAME64 = trezord-go-linux-amd64
VOL_MOUNT = -v $(shell pwd):/release
IMAGETAG  = trezord-go-build-env-$(PLATFORM)

.binary64:
	mkdir -p build
	xgo -targets=linux/amd64 ../..
	mv -f $(NAME64) build/trezord
	cp ../../VERSION build

.docker-image:
	$(info Preparing docker image ...)
	docker build -t $(IMAGETAG) .

shell: .docker-image
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /bin/bash

.package64:
	$(info Packaging ...)
	docker run -i -t $(VOL_MOUNT) $(IMAGETAG) /release/release.sh lin64
