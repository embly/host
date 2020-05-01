# counter = service(
#     "counter",
#     containers=[
#         container(
#             "counter",
#             image="hashicorpnomad/counter-api:v1",
#             cpu=500,
#             memory=256,
#             ports=[9001],
#         )
#     ],
# )

# dashboard = service(
#     "dashboard",
#     count=2,
#     containers=[
#         container(
#             "dashboard",
#             image="hashicorpnomad/counter-dashboard:v1",
#             cpu=500,
#             memory=256,
#             ports=[9002],
#             connect_to=["counter.counter:9001"],
#             environment={"COUNTING_SERVICE_URL": "counter.counter:9001"},
#         )
#     ],
# )


resources = dict(cpu=500, memory=256)
counter = container(
    "counter", image="hashicorpnomad/counter-api:v1", ports=[9001], **resources
)
standalone = service(
    "standalone",
    containers=[
        counter,
        container(
            "dashboard",
            image="hashicorpnomad/counter-dashboard:v1",
            ports=[9002],
            environment={"COUNTING_SERVICE_URL": "http://%s:9001" % counter.name},
            **resources
        ),
    ],
)
# 172.17.0.1


load_balancer(
    "all",
    {
        "localhost:8080": "dashboard.dashboard:9002",
        "localhost:8081": "dashboard.standalone:9002",
    },
)
