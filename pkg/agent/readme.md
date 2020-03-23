

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


# TODO
