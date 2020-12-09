FROM golang:1.15 as builder
ARG GIT_TAG_OVERRIDE
WORKDIR /app
COPY . .
RUN make build -e GIT_TAG_OVERRIDE=${GIT_TAG_OVERRIDE} && bin/k3d version

FROM docker:19.03-dind as dind
RUN apk update && apk add bash curl sudo jq git make netcat-openbsd
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl
COPY --from=builder /app/bin/k3d /bin/k3d

FROM scratch as binary-only
COPY --from=builder /app/bin/k3d /bin/k3d
ENTRYPOINT ["/bin/k3d"]