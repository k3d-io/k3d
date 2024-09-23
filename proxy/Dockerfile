FROM nginx:1.27.0-alpine3.19
# TODO:_ consider switching to https://github.com/abtreece/confd to not maintain a custom fork anymore

ARG OS
ARG ARCH

ENV OS=${OS}
ENV ARCH=${ARCH}
COPY install-confd.sh /scripts/install-confd.sh
RUN mkdir -p /etc/confd \
    && /scripts/install-confd.sh

COPY templates /etc/confd/templates/
COPY conf.d /etc/confd/conf.d/
COPY nginx-proxy /usr/bin/

ENTRYPOINT nginx-proxy
