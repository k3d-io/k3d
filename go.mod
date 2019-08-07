module github.com/rancher/k3d

go 1.12

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.12
	github.com/containerd/containerd v1.2.7 // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.3.3
	github.com/gogo/protobuf v1.2.1
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/mattn/go-runewidth v0.0.4
	github.com/mitchellh/go-homedir v1.1.0
	github.com/morikuni/aec v0.0.0-20170113033406-39771216ff4c // indirect
	github.com/olekukonko/tablewriter v0.0.1
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/stretchr/testify v1.3.0 // indirect
	github.com/urfave/cli v1.20.0
	golang.org/x/net v0.0.0-20190403144856-b630fd6fe46b
	golang.org/x/sys v0.0.0-20190422165155-953cdadca894
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/grpc v1.22.0 // indirect
	gotest.tools v2.2.0+incompatible // indirect
)

replace github.com/docker/docker v1.13.1 => github.com/docker/docker v0.7.3-0.20190723064612-a9dc697fd2a5
