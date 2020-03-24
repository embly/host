counter = service(
    "counter",
    containers=[
        container(
            "counter",
            image="hashicorpnomad/counter-api:v1",
            cpu=500,
            memory=256,
            ports=[9001],
        )
    ],
)

dashboard = service(
    "dashboard",
    count=2,
    containers=[
        container(
            "dashboard",
            image="hashicorpnomad/counter-dashboard:v1",
            cpu=500,
            memory=255,
            ports=[9002],
            connect_to=["counter.counter:9001"],
            environment={"COUNTIG_SERVICE_URL": "counter.counter:9001"},
        )
    ],
)


standalone = service(
    "standalone2",
    containers=[
        container(
            "counter",
            image="hashicorpnomad/counter-api:v1",
            cpu=500,
            memory=256,
            ports=[9001],
        ),
        container(
            "dashboard",
            image="hashicorpnomad/counter-dashboard:v1",
            cpu=500,
            memory=256,
            ports=[9002],
            environment={"COUNTING_SERVICE_URL": "http://counter:9001"},
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
