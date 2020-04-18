


# Summary

 - connect containers to other services



# Host Agent

Local:

Listen for new allocations. Allocations have multiple "tasks". Set up dns records for these containers.
When they request for siblings route them through iptables.

Also listen to connectTo requests from the nomad allocations. When a connectTo request is received, set up the correct iptables rule and proxy connection.

When a nomad allocation comes in, we key it by name+allocation_id.
Docker containers are keyed with the same key


# Proxy

reference: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies

This proxy server should be running on every nomad host. When a job is deployed, every proxy instance adds proxy rules and iptables rules to allow local jobs to address those services. When a job makes a request it hits the proxy and then a request is round-robin'd to a service that is listed. The proxy must listen on consul for job lifecycles to remove addresses from the pool and deregister the entire listener when the job no longer exists.

With that in mind, this proxy will:
 - accept requests on a port and proxy them to services elsewhere
 - the proxy must know:
   - the port it expects to listen on
   - the addr it expects to listen on
   - the jobs it is going to balance requests between
   - to deregister the listener if the service no longer exists
 - the iptables rules must:
   - know the ip of the container that is allowed to request a given service
   - know the ip and addr of the proxy responsible for that service

## Local IPTables rules docker listener rewrite

Listen for connect_to consul tags and add an ipt rule for traffic from that docker container to the local proxy running for the related service.

Needs to do three things:
 - listen for a docker container start and attach the rule if we have a proxy running
 - listen for a new service to connect to in consul and add the rule if we're just starting up a new proxy service (and we have knowledge of a running container)
 - listen for a new service to connect to another service in consul, and add the rule if we have an existing proxy service running
