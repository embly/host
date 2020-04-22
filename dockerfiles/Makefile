

simple_server_build:
	cd host-simple-server && docker build -t maxmcd/host-simple-server:latest .

simple_server_push: simple_server_build
	docker push maxmcd/host-simple-server:latest

build_image_build:
	cd build-image && docker build \
		-t maxmcd/host-build-image:latest \
		-f ./Dockerfile ../..

build_image_push: build_image_build
	docker push maxmcd/host-build-image:latest
