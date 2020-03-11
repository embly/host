FROM docker:19.03 as docker
FROM debian

COPY --from=docker /usr/local/bin/docker /usr/local/bin/docker
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates unzip
ENV NOMADVERSION=0.10.4
ENV NOMADDOWNLOAD=https://releases.hashicorp.com/nomad/${NOMADVERSION}/nomad_${NOMADVERSION}_linux_amd64.zip
RUN curl -L $NOMADDOWNLOAD > nomad.zip \
    && unzip nomad.zip -d /usr/local/bin \
    && rm nomad.zip

WORKDIR /app

RUN mkdir -p /tmp

COPY nomad.hcl nomad_entrypoint.sh /opt/

ENV NOMAD_RUN_ROOT=1

ENTRYPOINT ["/opt/nomad_entrypoint.sh"]

CMD nomad agent -dev -config=/opt/nomad.hcl
