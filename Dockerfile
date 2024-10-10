ARG DOCKER_VERSION=27.3.1
############################################################
# builder                                                  #
# -> golang image used solely for building the k3d binary  #
# -> built executable can then be copied into other stages #
############################################################
FROM golang:1.22.4 as builder
ARG GIT_TAG_OVERRIDE
WORKDIR /app
RUN mkdir /tmp/empty
COPY . .
RUN make build -e GIT_TAG_OVERRIDE=${GIT_TAG_OVERRIDE} && bin/k3d version

#######################################################
# dind                                                #
# -> k3d + some tools in a docker-in-docker container #
# -> used e.g. in our CI pipelines for testing        #
#######################################################
FROM docker:$DOCKER_VERSION-dind as dind
ARG OS
ARG ARCH

ENV OS=${OS}
ENV ARCH=${ARCH}

# Helper script to install some tooling
COPY scripts/install-tools.sh /scripts/install-tools.sh

# install some basic packages needed for testing, etc.
RUN apk update && \
    apk add bash curl sudo jq git make netcat-openbsd iptables

# install kubectl to interact with the k3d cluster
# install yq (yaml processor) from source, as the busybox yq had some issues
RUN /scripts/install-tools.sh kubectl yq

COPY --from=builder /app/bin/k3d /bin/k3d

#########################################
# binary-only                           #
# -> only the k3d binary.. nothing else #
#########################################
FROM scratch as binary-only
# empty tmp directory to avoid errors when transforming the config file
COPY --from=builder /tmp/empty /tmp
COPY --from=builder /app/bin/k3d /bin/k3d
ENTRYPOINT ["/bin/k3d"]
