Let's say we have a service like so:
```py

# flask.flask:8080
# redis.flask:8080

flask = service(
    "flask",
    count=2,  # asdfs
    containers=[
        container(
            "flask",
            image="crccheck/hello-world",
            ports=["8080/udp", 8080],
            cpu=500,
            memory=256,
            command="bash",
            connect_to=["apple.orange:8080"],
        ),
        container("redis", image="redis", ports=[8080]),
    ],
)
```

each of these containers will now be available at the following hosts:

 - flask.flask:8080
 - redis.flask:8080

We use the service name as the tld because it makes more sense from a hierarchy standpoint.

When a different service makes a request for that hostname we'll need to be sure to respond with
an ip address of a known proxy ip. Once we have responded with that ip, we'll need to add iptables
rules so that requests to that ip travel to the appropriate proxy address and port.

Since we're using a proxy for a component of the chain we can do isolation there. We'll also need to
make sure to lock down firewall rules for access to other containers.

could populate extra_hosts in dev, just to get things working, hmm, but
we still need the port mapping stuff to work, so likely still need iptables

## Deploy

A deploy could look like:

 - add hostnames to consul
 - open up ports for services on the proxy
 -

there is a proxy
the proxy takes requests on various ports
the dns server maps a hostname to the proxy ip address
so there is a proxy per machine and it's receiving inbound traffic for the services on that machine?
it allocates various ports on itself, when traffic hits those ports it is round-robined (random) to
the services it belongs to.

so when the docker driver is set up

for broadcasting:
    - consul gets the proxy ip and the name of the service
    - dns gets that ip and that name
for addressing:
    - iptables route is written in the docker container to map the outgoing port
      and proxy ip with the correct destination



## Resources

 - https://github.com/kubernetes/kubernetes/tree/master/pkg/proxy/ipvs
