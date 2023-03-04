FROM golang:1.20-alpine3.17 as builder
ARG GIT_TAG
WORKDIR /app
COPY . .
RUN apk update && apk add make bash git
ENV GIT_TAG=${GIT_TAG}
ENV GO111MODULE=on
ENV CGO_ENABLED=0
RUN make build

FROM alpine:3.17
RUN apk update && apk add bash
WORKDIR /app
COPY --from=builder /app/bin/k3d-tools .
ENTRYPOINT [ "/app/k3d-tools"]

