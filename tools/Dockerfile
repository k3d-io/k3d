FROM golang:1.14 as builder
ARG GIT_TAG
WORKDIR /app
COPY . .
ENV GIT_TAG=${GIT_TAG}
ENV GO111MODULE=on
ENV CGO_ENABLED=0
RUN make build

FROM busybox:1.31
WORKDIR /app
COPY --from=builder /app/bin/k3d-tools .
ENTRYPOINT [ "/app/k3d-tools"]