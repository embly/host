

build:
	docker build -t nomad-host-iptables .

run: build
	docker run -it --cap-add=NET_ADMIN nomad-host-iptables bash
