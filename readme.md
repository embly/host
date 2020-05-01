# Embly Hosting

Embly hosting provides simple tools to run share and develop web services.

Define a service you would like to run. All services have a name and are made up of containers that also have a name:

```python
counter = service(
    "counter",
    containers=[
        container(
            "counter",
            image="hashicorpnomad/counter-api:v1",
            ports=[9001],
        )
    ],
)
```

You can connect services together. This dashboard service would like to talk to the counter service. We ask embly to connect dasboard to counter. A services hostname is constructed using "{container_name}.{service_name}:{port}". You can't connect to another service without setting the `connect_to` value.

```python
dashboard = service(
    "dashboard",
    count=2,
    containers=[
        container(
            "dashboard",
            image="hashicorpnomad/counter-dashboard:v1",
            ports=[9002],
            connect_to=["counter.counter:9001"],
            environment={"COUNTING_SERVICE_URL": "counter.counter:9001"},
        )
    ],
)
```

You could also group counter and dashboard into one service. Now, the counting api can just be accessed with its container name and port "counter:9001".

```python
standalone = service(
    "standalone2",
    containers=[
        container(
            "counter",
            image="hashicorpnomad/counter-api:v1",
            ports=[9001],
        ),
        container(
            "dashboard",
            image="hashicorpnomad/counter-dashboard:v1",
            ports=[9002],
            environment={"COUNTING_SERVICE_URL": "http://counter:9001"},
        ),
    ],
)
```

Embly services are configured with starlark, so you can make use of its python language features:
```python
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
```
