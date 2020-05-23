

nomad_down:
	cd nomad && docker-compose kill
	cd nomad && docker-compose down

consul_build:
	docker pull nixery.dev/shell/consul
	docker tag nixery.dev/shell/consul embly/consul

nomad_build:
	cd $$GOPATH/src/github.com/hashicorp/nomad/ \
		&& docker build -f ./Dockerfile -t nomad-embly .
	cd nomad && docker build -t embly/nomad:latest -f nomad.Dockerfile .

nomad_run:
	make -j nomad_down nomad_build agent_docker_build
	cd nomad && docker-compose up

nomad_exec:
	cd nomad && docker-compose exec nomad bash -c "/opt/nomad_entrypoint.sh bash"

nomad_job_run:
	cd nomad && docker-compose exec \
		nomad bash -c "/opt/nomad_entrypoint.sh bash -c \"nomad job run job.hcl\""

test:
	go test -v -count=1 -cover ./...

agent_docker_build:
	cd cmd/twelve && docker build -f ./Dockerfile -t embly/twelve:latest ../../

agent_logs:
	cd nomad && docker-compose logs -f agent

agent_rebuild_and_restart: agent_docker_build
	cd nomad && docker-compose kill agent && docker-compose up -d agent
	make agent_logs

genapi_ast:
	cd python && docker build -t embly-host-ast .
	docker run -it embly-host-ast

CLI_RUN = cd cmd/twelve && go run -ldflags "-X main.version=`git rev-parse HEAD`" .
DEPLOY_ARG = run ../../nomad/counter.star
cli_run_deploy:
	$(CLI_RUN) $(DEPLOY_ARG)

cli_run:
	$(CLI_RUN) $(ARG)

cli_install:
	cd cmd/twelve && go install -ldflags "-X main.version=`git rev-parse HEAD`"

dns_run:
	cd cmd/dns && go run .


tmux_cli_run: nomad_down
	tmux \
    	new-session  'make nomad_run' \; \
    	split-window 'sleep 12 && make cli_run && bash' \; \
		split-window 'sleep 12 && make agent_logs' \;


generate:
	cd pkg/pb && go generate


build_image_push:
	cd dockerfiles && make build_image_push
