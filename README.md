# DockerFluent

DockerFluent is a Go library that provides a fluent interface for interacting with Docker. It offers a chainable API for common Docker operations and includes advanced features like sorting, partitioning, and grouping containers, as well as AI-powered analysis of Docker environments.

## Installation

```bash
go get github.com/vinodhalaharvi/dockerfluent
```

## Usage

Here are some examples of how to use DockerFluent:

### Deploying an Nginx Container

```go
func ExampleDockerFluent_DeployNginx() {
    ctx := context.Background()
    dockerF, _ := New()
    
    err := dockerF.PullImage(ctx, "nginx:latest").
        CreateContainer(ctx, &container.Config{
            Image:        "nginx:latest",
            ExposedPorts: nat.PortSet{"80/tcp": struct{}{}},
        }, &container.HostConfig{
            PortBindings: nat.PortMap{"80/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}}},
        }, "example-nginx").
        StartContainer(ctx, "example-nginx").
        WaitForHealthy(ctx, "example-nginx", 30*time.Second).
        AIChain(ctx, "Analyze the Nginx container configuration and suggest optimizations.", processAIResponse).
        Error()

    if err != nil {
        log.Printf("Error in ExampleDeployNginx: %v", err)
    } else {
        fmt.Println("Nginx container deployed and analyzed successfully!")
    }

    // Cleanup
    dockerF.StopContainer(ctx, "example-nginx").RemoveContainer(ctx, "example-nginx")
}
```

### Building a Custom Image

```go
func ExampleDockerFluent_BuildCustomImage() {
    ctx := context.Background()
    dockerF, _ := New()

    dockerfile := `
FROM golang:1.16-alpine
WORKDIR /app
COPY . .
RUN go build -o main .
CMD ["./main"]
`

    err := dockerF.
        BuildImage(ctx, dockerfile, "custom-go-app:latest").
        CreateContainer(ctx, &container.Config{
            Image: "custom-go-app:latest",
        }, &container.HostConfig{}, "custom-go-app").
        StartContainer(ctx, "custom-go-app").
        WaitForHealthy(ctx, "custom-go-app", 30*time.Second).
        BulkInspectAndAnalyze(ctx).
        AIChain(ctx, "Analyze the custom Go app container and suggest performance improvements.", processAIResponse).
        Error()

    if err != nil {
        log.Printf("Error in ExampleDockerFluent_BuildCustomImage: %v", err)
    } else {
        fmt.Println("Custom image built, container deployed and analyzed successfully!")
    }

    // Cleanup
    dockerF.StopContainer(ctx, "custom-go-app").RemoveContainer(ctx, "custom-go-app")
}
```

### Deploying a Multi-Container Application

```go
func ExampleDockerFluent_DeployMultiContainerApp() {
    ctx := context.Background()
    dockerF, _ := New()

    err := dockerF.
        PullImage(ctx, "nginx:latest").
        PullImage(ctx, "redis:latest").
        CreateNetwork(ctx, "example-network").
        CreateContainer(ctx, &container.Config{
            Image:        "nginx:latest",
            ExposedPorts: nat.PortSet{"80/tcp": struct{}{}},
        }, &container.HostConfig{
            PortBindings: nat.PortMap{"80/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}}},
            NetworkMode:  "example-network",
        }, "example-nginx").
        CreateContainer(ctx, &container.Config{
            Image: "redis:latest",
        }, &container.HostConfig{
            NetworkMode: "example-network",
        }, "example-redis").
        StartContainer(ctx, "example-nginx").
        StartContainer(ctx, "example-redis").
        WaitForHealthy(ctx, "example-nginx", 30*time.Second).
        WaitForHealthy(ctx, "example-redis", 30*time.Second).
        BulkInspectAndAnalyze(ctx).
        AIChain(ctx, "Analyze the multi-container setup and suggest improvements.", processAIResponse).
        Error()

    if err != nil {
        log.Printf("Error in ExampleDockerFluent_DeployMultiContainerApp: %v", err)
    } else {
        fmt.Println("Multi-container application deployed and analyzed successfully!")
    }

    // Cleanup
    dockerF.
        StopContainer(ctx, "example-nginx").
        StopContainer(ctx, "example-redis").
        RemoveContainer(ctx, "example-nginx").
        RemoveContainer(ctx, "example-redis").
        RemoveNetwork(ctx, "example-network")
}
```

### Managing Microservices

```go
func ExampleDockerFluent_ManageMicroservices() {
    ctx := context.Background()
    dockerF, _ := New()

    services := []struct {
        name  string
        image string
        port  string
    }{
        {"auth-service", "auth-service:v1", "8081"},
        {"user-service", "user-service:v1", "8082"},
        {"product-service", "product-service:v1", "8083"},
    }

    for _, svc := range services {
        err := dockerF.
            PullImage(ctx, svc.image).
            CreateContainer(ctx, &container.Config{
                Image: svc.image,
                ExposedPorts: nat.PortSet{
                    nat.Port(svc.port + "/tcp"): struct{}{},
                },
            }, &container.HostConfig{
                PortBindings: nat.PortMap{
                    nat.Port(svc.port + "/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: svc.port}},
                },
            }, svc.name).
            StartContainer(ctx, svc.name).
            WaitForHealthy(ctx, svc.name, 30*time.Second).
            Error()

        if err != nil {
            log.Printf("Error deploying %s: %v", svc.name, err)
            return
        }
    }

    // Use the new Sort method to sort containers by creation time
    dockerF.Sort(ctx, func(c1, c2 types.Container) bool {
        return c1.Created > c2.Created
    })

    // Use the new Partition method to separate running and stopped containers
    result, _ := dockerF.Partition(ctx, func(c types.Container) bool {
        return c.State == "running"
    })

    fmt.Printf("Running containers: %d, Stopped containers: %d\n",
        len(result.Matching), len(result.NonMatching))

    // Cleanup
    for _, svc := range services {
        dockerF.StopContainer(ctx, svc.name).RemoveContainer(ctx, svc.name)
    }
}
```

### Monitoring Containers

```go
func ExampleDockerFluent_MonitorContainers() {
    ctx := context.Background()
    dockerF, _ := New()

    containers, _ := dockerF.ListContainers(ctx)

    groups, _ := dockerF.GroupBy(ctx, func(c types.Container) string {
        return c.Image
    })

    for image, containers := range groups {
        fmt.Printf("Image %s has %d containers\n", image, len(containers))
    }

    for _, container := range containers {
        stats, _ := dockerF.Client.ContainerStats(ctx, container.ID, false)
        defer stats.Body.Close()

        var statsJSON types.StatsJSON
        json.NewDecoder(stats.Body).Decode(&statsJSON)

        fmt.Printf("Container %s (Image: %s):\n", container.ID[:12], container.Image)
        fmt.Printf("  CPU: %.2f%%\n", calculateCPUPercentUnix(statsJSON.CPUStats, statsJSON.PreCPUStats))
        fmt.Printf("  Memory: %.2f%%\n", calculateMemoryUsageUnixNoCache(statsJSON.MemoryStats))
    }

    dockerF.AIChain(ctx, "Analyze the container metrics and suggest optimizations for resource usage.", processAIResponse)
}
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
