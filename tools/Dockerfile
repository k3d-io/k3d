FROM golang:1.20.7-alpine3.18 as builder
ARG GIT_TAG
WORKDIR /app
COPY . .
RUN apk update && apk add make bash git
ENV GIT_TAG=${GIT_TAG}
ENV GO111MODULE=on
ENV CGO_ENABLED=0
RUN make build

FROM alpine:3.18
WORKDIR /app
RUN apk update && apk add bash
COPY --from=builder /app/bin/k3d-tools .
ENTRYPOINT [ "/app/k3d-tools"]

