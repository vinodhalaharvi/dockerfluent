package dockerfluent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"log"
	"os"
	"strings"
	"time"
)

// ExampleDeployNginx demonstrates deploying an Nginx container with custom configuration
func ExampleDockerFluent_DeployNginx() {
	ctx := context.Background()

	if err := os.Setenv(
		"OPENAI_API_KEY",
		"YOUR_OPENAI_API_KEY",
	); err != nil {
		panic(err)
	}

	dockerF, err := New("unix:///Users/vinodhalaharvi/.docker/run/docker.sock") // Or specify your Docker socket path if known
	if err != nil {
		log.Fatalf("Error creating dockerfluent.DockerFluent client: %v", err)
	}
	fmt.Println("Deploying Nginx container...")

	processAIResponse := func(aiResponse string) func(*DockerFluent) *DockerFluent {
		return func(d *DockerFluent) *DockerFluent {
			fmt.Printf("AI Analysis of Container:\n%s\n", aiResponse)

			// Here you could implement logic to take actions based on the AI's analysis
			// For example, if the AI identifies a resource constraint, you could increase the container's resources
			if strings.Contains(strings.ToLower(aiResponse), "memory limit") {
				fmt.Println("AI suggested increasing memory limit. Implementing this suggestion...")
				// Implement logic to increase memory limit
				// This is just a placeholder - actual implementation would depend on your specific needs
				// return d.UpdateContainerResources(context.Background(), "my-nginx", updatedResources)
			}

			// For now, we'll just continue the chain without any specific action
			return d
		}
	}
	err = dockerF.PullImage(ctx, "nginx:latest").
		CreateContainer(ctx, &container.Config{
			Image:        "nginx:latest",
			ExposedPorts: nat.PortSet{"80/tcp": struct{}{}},
		}, &container.HostConfig{
			PortBindings: nat.PortMap{"80/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "8080"}}},
		}, "example-nginx").
		StartContainer(ctx, "example-nginx").
		WaitForHealthy(ctx, "example-nginx", 30*time.Second). // Assuming we've implemented this method
		//EnhancedInspect(ctx, "example-nginx").
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

// ExampleDockerFluent_BuildCustomImage demonstrates building a custom image and running a container from it
func ExampleDockerFluent_BuildCustomImage() {
	ctx := context.Background()

	if err := os.Setenv(
		"OPENAI_API_KEY",
		"YOUR_OPENAI_API_KEY",
	); err != nil {
		panic(err)
	}

	dockerF, err := New() // Or specify your Docker socket path if known
	if err != nil {
		log.Fatalf("Error creating dockerfluent.DockerFluent client: %v", err)
	}
	fmt.Println("Building custom image and deploying container...")

	dockerfile := `
FROM golang:1.16-alpine
WORKDIR /app
COPY . .
RUN go build -o main .
CMD ["./main"]
`
	processAIResponse := func(aiResponse string) func(*DockerFluent) *DockerFluent {
		return func(d *DockerFluent) *DockerFluent {
			fmt.Printf("AI Analysis of Container:\n%s\n", aiResponse)

			// Here you could implement logic to take actions based on the AI's analysis
			// For example, if the AI identifies a resource constraint, you could increase the container's resources
			if strings.Contains(strings.ToLower(aiResponse), "memory limit") {
				fmt.Println("AI suggested increasing memory limit. Implementing this suggestion...")
				// Implement logic to increase memory limit
				// This is just a placeholder - actual implementation would depend on your specific needs
				// return d.UpdateContainerResources(context.Background(), "my-nginx", updatedResources)
			}

			// For now, we'll just continue the chain without any specific action
			return d
		}
	}

	err = dockerF.
		BuildImage(ctx, dockerfile, "custom-go-app:latest"). // Assuming we've implemented this method
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

func ExampleDockerFluent_DeployMultiContainerApp() {
	ctx := context.Background()

	if err := os.Setenv(
		"OPENAI_API_KEY",
		"YOUR_OPENAI_API_KEY",
	); err != nil {
		panic(err)
	}

	dockerF, err := New() // Or specify your Docker socket path if known
	if err != nil {
		log.Fatalf("Error creating dockerfluent.DockerFluent client: %v", err)
	}
	fmt.Println("Deploying multi-container application...")
	processAIResponse := func(aiResponse string) func(*DockerFluent) *DockerFluent {
		return func(d *DockerFluent) *DockerFluent {
			fmt.Printf("AI Analysis of Container:\n%s\n", aiResponse)

			// Here you could implement logic to take actions based on the AI's analysis
			// For example, if the AI identifies a resource constraint, you could increase the container's resources
			if strings.Contains(strings.ToLower(aiResponse), "memory limit") {
				fmt.Println("AI suggested increasing memory limit. Implementing this suggestion...")
				// Implement logic to increase memory limit
				// This is just a placeholder - actual implementation would depend on your specific needs
				// return d.UpdateContainerResources(context.Background(), "my-nginx", updatedResources)
			}

			// For now, we'll just continue the chain without any specific action
			return d
		}
	}

	err = dockerF.
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
		AIChain(
			ctx,
			"Analyze the multi-container setup and suggest improvements for container communication and resource allocation.",
			processAIResponse,
		).
		Error()

	if err != nil {
		log.Printf("Error in ExampleDockerFluent_DeployMultiContainerApp: %v", err)
	} else {
		fmt.Println("Multi-container application deployed and analyzed successfully!")

		// Generate SVGs for both containers
		containers := []string{"example-nginx", "example-redis"}
		for _, containerName := range containers {
			containerJSON, err := dockerF.Client.ContainerInspect(ctx, containerName)
			if err != nil {
				log.Printf("Error inspecting container %s: %v", containerName, err)
				continue
			}

			svg := dockerF.GenerateContainerSVG(&containerJSON)
			filename := fmt.Sprintf("%s_visualization.svg", containerName)
			if err := os.WriteFile(filename, []byte(svg), 0644); err != nil {
				log.Printf("Error writing SVG file for %s: %v", containerName, err)
			} else {
				fmt.Printf("SVG visualization for %s saved to %s\n", containerName, filename)
			}
		}
	}

	// Cleanup
	dockerF.
		StopContainer(ctx, "example-nginx").
		StopContainer(ctx, "example-redis").
		RemoveContainer(ctx, "example-nginx").
		RemoveContainer(ctx, "example-redis").
		RemoveNetwork(ctx, "example-network")
}

// ExampleDockerFluent_DeployDatabaseCluster demonstrates deploying a cluster of database containers
func ExampleDockerFluent_DeployDatabaseCluster() {
	ctx := context.Background()

	if err := os.Setenv(
		"OPENAI_API_KEY",
		"YOUR_OPENAI_API_KEY",
	); err != nil {
		panic(err)
	}

	dockerF, err := New() // Or specify your Docker socket path if known
	if err != nil {
		log.Fatalf("Error creating dockerfluent.DockerFluent client: %v", err)
	}
	fmt.Println("Deploying database cluster...")

	// Create a network for the cluster
	err = dockerF.CreateNetwork(ctx, "db-cluster-network").Error()
	if err != nil {
		log.Printf("Error creating network: %v", err)
		return
	}

	// Deploy multiple database containers
	for i := 1; i <= 3; i++ {
		containerName := fmt.Sprintf("db-node-%d", i)
		err := dockerF.
			PullImage(ctx, "postgres:13").
			CreateContainer(ctx, &container.Config{
				Image: "postgres:13",
				Env:   []string{"POSTGRES_PASSWORD=secretpassword"},
			}, &container.HostConfig{
				NetworkMode: "db-cluster-network",
			}, containerName).
			StartContainer(ctx, containerName).
			WaitForHealthy(ctx, containerName, 30*time.Second).
			Error()

		if err != nil {
			log.Printf("Error deploying %s: %v", containerName, err)
			return
		}
	}

	// Use the new GroupBy method to group containers by their status
	groups, err := dockerF.GroupBy(ctx, func(c types.Container) string {
		return c.State
	})
	if err != nil {
		log.Printf("Error grouping containers: %v", err)
	} else {
		for state, containers := range groups {
			fmt.Printf("Containers in state %s: %d\n", state, len(containers))
		}
	}
	processAIResponse := func(aiResponse string) func(*DockerFluent) *DockerFluent {
		return func(d *DockerFluent) *DockerFluent {
			fmt.Printf("AI Analysis of Container:\n%s\n", aiResponse)

			// Here you could implement logic to take actions based on the AI's analysis
			// For example, if the AI identifies a resource constraint, you could increase the container's resources
			if strings.Contains(strings.ToLower(aiResponse), "memory limit") {
				fmt.Println("AI suggested increasing memory limit. Implementing this suggestion...")
				// Implement logic to increase memory limit
				// This is just a placeholder - actual implementation would depend on your specific needs
				// return d.UpdateContainerResources(context.Background(), "my-nginx", updatedResources)
			}

			// For now, we'll just continue the chain without any specific action
			return d
		}
	}

	// Use the AI chain to analyze the cluster setup
	dockerF.AIChain(ctx, "Analyze the database cluster configuration and suggest optimizations.", processAIResponse)

	fmt.Println("Database cluster deployed and analyzed successfully!")

	// Cleanup
	for i := 1; i <= 3; i++ {
		containerName := fmt.Sprintf("db-node-%d", i)
		dockerF.StopContainer(ctx, containerName).RemoveContainer(ctx, containerName)
	}
	dockerF.RemoveNetwork(ctx, "db-cluster-network")
}

// ExampleDockerFluent_ManageMicroservices demonstrates managing a set of microservices
func ExampleDockerFluent_ManageMicroservices() {
	ctx := context.Background()

	if err := os.Setenv(
		"OPENAI_API_KEY",
		"YOUR_OPENAI_API_KEY",
	); err != nil {
		panic(err)
	}

	dockerF, err := New() // Or specify your Docker socket path if known
	if err != nil {
		log.Fatalf("Error creating dockerfluent.DockerFluent client: %v", err)
	}
	fmt.Println("Managing microservices...")

	// Deploy multiple microservices
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
		err = dockerF.
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
	result, err := dockerF.Partition(ctx, func(c types.Container) bool {
		return c.State == "running"
	})
	if err != nil {
		log.Printf("Error partitioning containers: %v", err)
	} else {
		fmt.Printf("Running containers: %d, Stopped containers: %d\n",
			len(result.Matching), len(result.NonMatching))
	}
	processAIResponse := func(aiResponse string) func(*DockerFluent) *DockerFluent {
		return func(d *DockerFluent) *DockerFluent {
			fmt.Printf("AI Analysis of Container:\n%s\n", aiResponse)

			// Here you could implement logic to take actions based on the AI's analysis
			// For example, if the AI identifies a resource constraint, you could increase the container's resources
			if strings.Contains(strings.ToLower(aiResponse), "memory limit") {
				fmt.Println("AI suggested increasing memory limit. Implementing this suggestion...")
				// Implement logic to increase memory limit
				// This is just a placeholder - actual implementation would depend on your specific needs
				// return d.UpdateContainerResources(context.Background(), "my-nginx", updatedResources)
			}

			// For now, we'll just continue the chain without any specific action
			return d
		}
	}

	// Use the AI chain to analyze the microservices setup
	dockerF.AIChain(ctx, "Analyze the microservices configuration and suggest improvements for scalability.", processAIResponse)

	fmt.Println("Microservices deployed and analyzed successfully!")

	// Cleanup
	for _, svc := range services {
		dockerF.StopContainer(ctx, svc.name).RemoveContainer(ctx, svc.name)
	}
}

// ExampleDockerFluent_MonitorContainers demonstrates monitoring and analyzing container metrics
func ExampleDockerFluent_MonitorContainers() {
	ctx := context.Background()

	if err := os.Setenv(
		"OPENAI_API_KEY",
		"YOUR_OPENAI_API_KEY",
	); err != nil {
		panic(err)
	}

	dockerF, err := New() // Or specify your Docker socket path if known
	if err != nil {
		log.Fatalf("Error creating dockerfluent.DockerFluent client: %v", err)
	}
	fmt.Println("Monitoring containers...")

	// List all running containers
	containers, err := dockerF.ListContainers(ctx)
	if err != nil {
		log.Printf("Error listing containers: %v", err)
		return
	}

	// Use the new GroupBy method to group containers by their image
	groups, err := dockerF.GroupBy(ctx, func(c types.Container) string {
		return c.Image
	})
	if err != nil {
		log.Printf("Error grouping containers: %v", err)
	} else {
		for image, containers := range groups {
			fmt.Printf("Image %s has %d containers\n", image, len(containers))
		}
	}

	// Collect metrics for each container
	for _, container := range containers {
		stats, err := dockerF.Client.ContainerStats(ctx, container.ID, false)
		if err != nil {
			log.Printf("Error getting stats for container %s: %v", container.ID, err)
			continue
		}
		defer stats.Body.Close()

		var statsJSON types.StatsJSON
		if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
			log.Printf("Error decoding stats for container %s: %v", container.ID, err)
			continue
		}
		calculateCPUPercentUnix := func(v types.CPUStats, pre types.CPUStats) float64 {
			cpuPercent := 0.0
			// calculate the change for the cpu usage of the container in between readings
			cpuDelta := float64(v.CPUUsage.TotalUsage) - float64(pre.CPUUsage.TotalUsage)
			// calculate the change for the entire system between readings
			systemDelta := float64(v.SystemUsage) - float64(pre.SystemUsage)

			if systemDelta > 0.0 && cpuDelta > 0.0 {
				cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUUsage.PercpuUsage)) * 100.0
			}
			return cpuPercent
		}
		calculateMemoryUsageUnixNoCache := func(memStats types.MemoryStats) float64 {
			used := float64(memStats.Usage - memStats.Stats["cache"])
			limit := float64(memStats.Limit)
			return (used / limit) * 100.0
		}

		fmt.Printf("Container %s (Image: %s):\n", container.ID[:12], container.Image)
		fmt.Printf("  CPU: %.2f%%\n", calculateCPUPercentUnix(statsJSON.CPUStats, statsJSON.PreCPUStats))
		fmt.Printf("  Memory: %.2f%%\n", calculateMemoryUsageUnixNoCache(statsJSON.MemoryStats))
	}

	processAIResponse := func(aiResponse string) func(*DockerFluent) *DockerFluent {
		return func(d *DockerFluent) *DockerFluent {
			fmt.Printf("AI Analysis of Container:\n%s\n", aiResponse)

			// Here you could implement logic to take actions based on the AI's analysis
			// For example, if the AI identifies a resource constraint, you could increase the container's resources
			if strings.Contains(strings.ToLower(aiResponse), "memory limit") {
				fmt.Println("AI suggested increasing memory limit. Implementing this suggestion...")
				// Implement logic to increase memory limit
				// This is just a placeholder - actual implementation would depend on your specific needs
				// return d.UpdateContainerResources(context.Background(), "my-nginx", updatedResources)
			}

			// For now, we'll just continue the chain without any specific action
			return d
		}
	}
	// Use the AI chain to analyze the container metrics
	dockerF.AIChain(ctx, "Analyze the container metrics and suggest optimizations for resource usage.", processAIResponse)

	fmt.Println("Container monitoring and analysis completed successfully!")
}
