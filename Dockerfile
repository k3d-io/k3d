FROM golang:1.16 as builder
ARG GIT_TAG_OVERRIDE
WORKDIR /app
COPY . .
RUN make build -e GIT_TAG_OVERRIDE=${GIT_TAG_OVERRIDE} && bin/k3d version

FROM docker:20.10-dind as dind
ARG OS=linux
ARG ARCH=amd64

# install some basic packages needed for testing, etc.
RUN echo "building for ${OS}/${ARCH}" && \
    apk update && \
    apk add bash curl sudo jq git make netcat-openbsd

# install kubectl to interact with the k3d cluster
RUN curl -L https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/${OS}/${ARCH}/kubectl -o /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl

# install yq (yaml processor) from source, as the busybox yq had some issues
RUN curl -L https://github.com/mikefarah/yq/releases/download/v4.9.6/yq_${OS}_${ARCH} -o /usr/bin/yq &&\
    chmod +x /usr/bin/yq
COPY --from=builder /app/bin/k3d /bin/k3d

FROM scratch as binary-only
COPY --from=builder /app/bin/k3d /bin/k3d
ENTRYPOINT ["/bin/k3d"]
