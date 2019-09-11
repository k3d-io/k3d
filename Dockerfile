FROM golang:1.13 as builder
WORKDIR /app
COPY . .
RUN make build && bin/k3d --version

FROM docker:19.03-dind
COPY --from=builder /app/bin/k3d /bin/k3d
