


down:
	docker-compose kill
	docker-compose down

build:
	docker build -t host_nomad:latest -f nomad.Dockerfile .

run:
	make -j down build
	docker-compose up


exec_nomad:
	docker-compose exec nomad bash -c "/opt/nomad_entrypoint.sh bash"
