


down:
	cd nomad && docker-compose kill
	cd nomad && docker-compose down

build:
	cd nomad && docker build -t host_nomad:latest -f nomad.Dockerfile .

run:
	make -j down build
	cd nomad && docker-compose up

test:
	go test -v -count=1 -cover .

exec_nomad:
	cd nomad && docker-compose exec nomad bash -c "/opt/nomad_entrypoint.sh bash"

nomad_job_run:
	cd nomad && docker-compose exec \
		nomad bash -c "/opt/nomad_entrypoint.sh bash -c \"nomad job run job.hcl\""

build_hello:
	cd cmd/hello && docker build -t maxmcd/hello:latest .
	docker push maxmcd/hello:latest

generate_api_ast:
	cd python && docker build -t embly-host-ast .
	docker run -it embly-host-ast

run_run:
	cd cmd/run && go run . ../../nomad/counter.cfg
