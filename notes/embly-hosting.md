Embly Hosting

Nomad, but it uses a minimal yaml config. Idea here being twofold:
 - Many of the things needed to set up embly+wasm involve becoming a complete hosting provider
 - k8s for hobbyists is just silly

General:
 - everything is in nomad
 - you could even use this to trial run the github product thing, git urls that map to libs that can be used. kind of taints the wasm vision, but hopefully not a big deal.
 - yeah, I mean if we can define the build script in this config as well, then we can offer free CI like we've wanted to
 - these are sentences not bullet points
 - it is pretty sick though, the same "punch" can be provided, http request with a git repo url that you can spin up in an instant.
 - so slow though, but it respects the patterns we're building
Pieces:

Docker:
 - gvisor
 - mem and cpu hard limits, billed on those values
 - metered networking, billed on that value
 - disk, not sure...
 - blob storage, given an account prefix
 - git integration for continuous delivery, but we're not a git host
 - can list and exec nodes
 - fast deploys, pull docker layers in parallel like a madman
 - easy for ci use? quite a rabbit hole though
 - provide serverless primitives, but definitely be shady about it, like, be all mad that it's on a server, talk louder than people that disagree

Secrets
 - UI for secret management, use with account login, everyone has access in the beginning (punt on permissions)

Load Balancing
 - configurable in the ui, http vs tcp

Persistent Disk
 - standard migration stuff, maybe write a disk driver, but kind of ignore this for now

Monitoring:
 - m3db, prometheus, look into namespacing, billed per item
 - http stats and billing stats come for free

Health checks
 - per-service health checks
 - global health checks

Nice things:
 - distributed locks
 - message queue, billed on message (this is the embly problem, start small, listen to users?)
 - key value store
 - blob storage
 - provide all of this stuff locally, local dev is prod

Logs:
 - free and provided

Databases:
 - oof
 - foundation db all the way?
 - gvisor is going to make this hard, redis/postgres performance would be meh
 - vpc peering is hard for this, can we do vpc peering with a shared host? maybe proxy through a t4.micro or something...

https:
 - let's encrypt


### A Functionality description:

Go server that sits in front of nomad/consul. Also ships with a web interface.
Used to deploy servers, roll back to previous deploys.
Namespaces nomad, allows for isolated resources.

### Starlark?

It's a docker-compose replacement with more features. Just make `service()` in starlark and allow the population of service methods. Use it to build locally.
```python


shared = volume("shared")

metrics = service(
    "metrics",
    image="maxmcd/metrics",
    ports=[8080],
    volumes={"/opt":shared})

flask = service(
    "flask",
    build=None,
    environment={"FOO":secrets.flask.database_url,
    "STATS_HOST": metrics.address},
)
```


You can deploy a service or a cron job (serverless stuff comes later). This is all configured in the
configuration file.

There is a load balancer sitting on every node. The load balancer
