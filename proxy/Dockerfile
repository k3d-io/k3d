FROM nginx:1.16.0-alpine
ARG CONFD_REPO=iwilltry42/confd
ARG CONFD_VERSION=0.16.1
ARG OS=linux
ARG ARCH=amd64
RUN echo "Building for '${OS}/${ARCH}'..." \
    && apk -U --no-cache add curl ca-certificates\
    && mkdir -p /etc/confd \
    && curl -sLf "https://github.com/${CONFD_REPO}/releases/download/v${CONFD_VERSION}/confd-${CONFD_VERSION}-${OS}-${ARCH}" > /usr/bin/confd \
    && chmod +x /usr/bin/confd \
    && apk del curl

COPY templates /etc/confd/templates/
COPY conf.d /etc/confd/conf.d/
COPY nginx-proxy /usr/bin/

ENTRYPOINT nginx-proxy