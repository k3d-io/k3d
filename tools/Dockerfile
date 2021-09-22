FROM golang:1.17-alpine as builder
ARG GIT_TAG
WORKDIR /app
COPY . .
RUN apk update && apk add make bash git
ENV GIT_TAG=${GIT_TAG}
ENV GO111MODULE=on
ENV CGO_ENABLED=0
RUN make build

FROM alpine:3.14
RUN apk update && apk add bash
WORKDIR /app
COPY --from=builder /app/bin/k3d-tools .
ENTRYPOINT [ "/app/k3d-tools"]

<IP>\s+STREAM\s+host.docker.internal
