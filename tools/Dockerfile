FROM golang:1.14 as builder
WORKDIR /app
COPY . .
ENV GO111MODULE=on
ENV CGO_ENABLED=0
RUN make build

FROM busybox:1.31
WORKDIR /app
COPY --from=builder /app/bin/k3d-tools .
ENTRYPOINT [ "/app/k3d-tools"]