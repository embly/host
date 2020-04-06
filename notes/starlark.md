

use reflection to just pass the go values back into starlark, makes for easy extensibility


```py


embly.services.flask.ports.http
embly.services.flask.


load("embly", "job", "task")

flask = service(
    "flask",
    count=2,  # asdfs
    image="crccheck/hello-world",
    ports=[port("http", 8000)],
    cpu=500,
    memory=256,
)


flask = job(
    "flask",
    count=2,  # asdfs
    tasks=[
        container(
            "flask",
            image="crccheck/hello-world",
            ports=[port("http", 8000)],
            cpu=500,
            memory=256,
        ),
        container(
            "redis",
            image="crccheck/hello-world",
            ports=[port("http", 8000)],
            cpu=500,
            memory=256,
        ),
    ],
)



load_balancer("flask-tcp", port=8000, type="tcp", target=flask.ports.http)
load_balancer("flask-http", port=8080, type="http", target=flask.ports.http)
```

We need:
 - individual task definition
 - group tasks into groups that are colocated
 - talk to another task on another machine
 - talk to a specific port of another task in my group
 - set up a tcp/udp/http load balancer that I can access from outside of the cluster



```py



```
