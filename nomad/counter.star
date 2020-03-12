counter = service(
    "counter",
    containers=[
        container(
            "counter",
            image="hashicorpnomad/counter-api:v1",
            cpu=500,
            memory=256,
            ports=[9002],
        )
    ],
)

dashboard = service(
    "dashboard",
    containers=[
        container(
            "dashboard",
            image="hashicorpnomad/counter-dashboard:v1",
            cpu=500,
            memory=256,
            ports=[9002],
            connect_to=["counter.counter:9002"],
            environment={"COUNTING_SERVICE_URL": "counter.counter:9002"},
        )
    ],
)


standalone = service(
    "standalone",
    containers=[
        container(
            "counter",
            image="hashicorpnomad/counter-api:v1",
            cpu=500,
            memory=256,
            ports=[9002],
        ),
        container(
            "dashboard",
            image="hashicorpnomad/counter-dashboard:v1",
            cpu=500,
            memory=256,
            ports=[9002],
            environment={"COUNTING_SERVICE_URL": "counter:9002"},
        ),
    ],
)


load_balancer(
    "all",
    {
        "localhost:8080": "dashboard.dashboard:9002",
        "localhost:8081": "dashboard.standalone:9002",
    },
)
