dashboard = service(
    "dashboard",
    count=2,
    env={},
    containers=[
        container(
            "dashboard",
            image="hashicorpnomad/counter-dashboard:v1",
            cpu=500,
            memory=256,
            ports=[9002],
            connect_to=["counter.counter:9001"],
            env={"COUNTIG_SERVICE_URL": "counter.counter:9001"},
            command="sleep 10000",
        ),
        container("foo", image="asdfa", command=["sleep", "10000"],),
    ],
)
