# flask_build = docker_build(
#     sources=["./Dockerfile", "./go.*", "./main.go"], file="./Dockerfile"
# )
# nomad_build = docker_build(
#     sources=["./nomad.Dockerfile"], build_args={"FLASK_IMG": flask_build.tag}
# )

flask = service(
    "flask",
    count=2,  # asdfs
    image="crccheck/hello-world",
    ports=[port("http", 8000)],
    cpu=500,
    memory=256,
)

load_balancer("flask-tcp", port=8000, type="tcp", target=flask.ports.http)
load_balancer("flask-http", port=8080, type="http", target=flask.ports.http)
