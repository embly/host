FROM docker-driver-embly as docker
FROM docker:19.03 as docker
FROM debian

COPY --from=docker /usr/local/bin/docker /usr/local/bin/docker
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    unzip \
    iptables
ENV NOMADVERSION=0.10.4
ENV NOMADDOWNLOAD=https://releases.hashicorp.com/nomad/${NOMADVERSION}/nomad_${NOMADVERSION}_linux_amd64.zip
RUN curl -L $NOMADDOWNLOAD > nomad.zip \
    && unzip nomad.zip -d /usr/local/bin \
    && rm nomad.zip

RUN curl -L -o cni-plugins.tgz https://github.com/containernetworking/plugins/releases/download/v0.8.4/cni-plugins-linux-amd64-v0.8.4.tgz \
    && mkdir -p /opt/cni/bin \
    && tar -C /opt/cni/bin -xzf cni-plugins.tgz
WORKDIR /app

RUN mkdir -p /tmp

COPY nomad.hcl nomad_entrypoint.sh /opt/
COPY --from=docker-driver-embly /go/src/github.com/hashicorp/nomad/drivers/docker/cmd/custom-docker /opt/plugins/
ENV NOMAD_RUN_ROOT=1

ENTRYPOINT ["/opt/nomad_entrypoint.sh"]

CMD nomad agent -dev -config=/opt/nomad.hcl -plugin-dir=/opt/plugins
