FROM nomad-embly as docker
FROM docker:19.03 as docker
FROM debian

COPY --from=docker /usr/local/bin/docker /usr/local/bin/docker
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    unzip \
    iptables

RUN curl -L -o cni-plugins.tgz https://github.com/containernetworking/plugins/releases/download/v0.8.4/cni-plugins-linux-amd64-v0.8.4.tgz \
    && mkdir -p /opt/cni/bin \
    && tar -C /opt/cni/bin -xzf cni-plugins.tgz
WORKDIR /app

RUN mkdir -p /tmp

COPY nomad.hcl nomad_entrypoint.sh /opt/
COPY --from=nomad-embly /bin/nomad /usr/local/bin/nomad
ENV NOMAD_RUN_ROOT=1

ENTRYPOINT ["/opt/nomad_entrypoint.sh"]

CMD nomad agent -dev -config=/opt/nomad.hcl -plugin-dir=/opt/plugins
