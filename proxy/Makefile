.PHONY: test

test:
	docker build . -t ghcr.io/k3d-io/k3d-proxy:dev --no-cache
	docker run --rm -v $(shell pwd)/test/portmap.yaml:/etc/confd/values.yaml ghcr.io/k3d-io/k3d-proxy:dev
