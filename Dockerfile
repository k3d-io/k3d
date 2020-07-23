FROM golang:1.14 as builder
WORKDIR /app
COPY . .
RUN make build && bin/k3d version

FROM docker:19.03-dind as dind
RUN apk add bash curl sudo jq git make
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl
COPY --from=builder /app/bin/k3d /bin/k3d
