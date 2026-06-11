// createContainer creates a new container
func createContainer(ctx context.Context, containerConfig *container.Config) (*container.Container, error) {
    // ...
    if containerConfig.NetworkMode == "host" {
        containerConfig.NetworkMode = "none"
    }
    // ...
    return dockerClient.CreateContainer(ctx, containerConfig)
}