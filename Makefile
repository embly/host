


nomad_down:
	cd nomad && docker-compose kill
	cd nomad && docker-compose down

nomad_build:
	cd $$GOPATH/src/github.com/hashicorp/nomad/ \
		&& docker build -f ./Dockerfile -t nomad-embly .
	cd nomad && docker build -t nomad_nomad:latest -f nomad.Dockerfile .

nomad_run:
	make -j nomad_down nomad_build agent_build
	cd nomad && docker-compose up

nomad_exec:
	cd nomad && docker-compose exec nomad bash -c "/opt/nomad_entrypoint.sh bash"

nomad_job_run:
	cd nomad && docker-compose exec \
		nomad bash -c "/opt/nomad_entrypoint.sh bash -c \"nomad job run job.hcl\""

test:
	go test -v -count=1 -cover ./...

docker_scratch_run:
	cd pkg/docker-scratch/ \
		&& go run .

hello_exec:
	docker exec -it $$(docker ps --filter ancestor=maxmcd/hello:latest -q) bash

hello_run: hello_build
	docker run --cap-add=NET_ADMIN -p 8080:8080 maxmcd/hello:latest

hello_build:
	cd cmd/hello && docker build -t maxmcd/hello:latest .

hello_push: hello_build
	docker push maxmcd/hello:latest

agent_build:
	cd cmd/agent && docker build -f ./Dockerfile -t embly-host-agent:latest ../../

genapi_ast:
	cd python && docker build -t embly-host-ast .
	docker run -it embly-host-ast

cli_run:
	cd cmd/cli && go run . ../../nomad/counter.star

dns_run:
	cd cmd/dns && go run .


tmux_cli_run: nomad_down
	tmux \
    	new-session  'make nomad_run' \; \
    	split-window 'sleep 12 && make cli_run && bash' \; \
		split-window 'sleep 12 && cd nomad && docker-compose logs -f agent' \;
