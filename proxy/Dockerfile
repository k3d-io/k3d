FROM nginx:1.19-alpine
# TODO:_ consider switching to https://github.com/abtreece/confd to not maintain a custom fork anymore
ARG CONFD_REPO=iwilltry42/confd
ARG CONFD_VERSION=0.17.0-rc.0
ARG OS=linux
ARG ARCH=amd64
RUN echo "Building for '${OS}/${ARCH}'..." \
    && mkdir -p /etc/confd \
    && wget "https://github.com/${CONFD_REPO}/releases/download/v${CONFD_VERSION}/confd-${CONFD_VERSION}-${OS}-${ARCH}" -O /usr/bin/confd \
    && chmod +x /usr/bin/confd

COPY templates /etc/confd/templates/
COPY conf.d /etc/confd/conf.d/
COPY nginx-proxy /usr/bin/

ENTRYPOINT nginx-proxy