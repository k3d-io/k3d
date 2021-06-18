.PHONY: test

test:
	docker build . -t rancher/k3d-proxy:dev
	docker run --rm -v $(shell pwd)/test/portmap.yaml:/etc/confd/values.yaml rancher/k3d-proxy:dev