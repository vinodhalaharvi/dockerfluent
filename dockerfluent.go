package dockerfluent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/sashabaranov/go-openai"
)

// DockerFluent represents a monadic operation on the Docker client
type DockerFluent struct {
	Client *client.Client
	err    error
}

// New creates a new DockerFluent instance with optional Docker host
func New(dockerHost ...string) (*DockerFluent, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	if len(dockerHost) > 0 && dockerHost[0] != "" {
		opts = append(opts, client.WithHost(dockerHost[0]))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}
	return &DockerFluent{Client: cli}, nil
}

// Bind chains an operation to the DockerFluent
func (d *DockerFluent) Bind(operation func(*client.Client) error) *DockerFluent {
	if d.err != nil {
		return d
	}
	d.err = operation(d.Client)
	return d
}

// ConcurrentBind executes multiple operations concurrently
func (d *DockerFluent) ConcurrentBind(operations ...func(*client.Client) error) *DockerFluent {
	if d.err != nil {
		return d
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(operations))

	for _, op := range operations {
		wg.Add(1)
		go func(operation func(*client.Client) error) {
			defer wg.Done()
			if err := operation(d.Client); err != nil {
				errs <- err
			}
		}(op)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			d.err = err
			return d
		}
	}

	return d
}

// CreateContainer creates a new container
func (d *DockerFluent) CreateContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, name string) *DockerFluent {
	return d.Bind(func(cli *client.Client) error {
		_, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, name)
		return err
	})
}

// StartContainer starts a container
func (d *DockerFluent) StartContainer(ctx context.Context, containerID string) *DockerFluent {
	return d.Bind(func(cli *client.Client) error {
		return cli.ContainerStart(ctx, containerID, container.StartOptions{})
	})
}

// StopContainer stops a container
func (d *DockerFluent) StopContainer(ctx context.Context, containerID string) *DockerFluent {
	return d.Bind(func(cli *client.Client) error {
		return cli.ContainerStop(ctx, containerID, container.StopOptions{})
	})
}

// RemoveContainer removes a container
func (d *DockerFluent) RemoveContainer(ctx context.Context, containerID string) *DockerFluent {
	return d.Bind(func(cli *client.Client) error {
		return cli.ContainerRemove(ctx, containerID, container.RemoveOptions{})
	})
}

// ListContainers lists all containers
func (d *DockerFluent) ListContainers(ctx context.Context) ([]types.Container, error) {
	var containers []types.Container
	err := d.Bind(func(cli *client.Client) error {
		var err error
		containers, err = cli.ContainerList(ctx, container.ListOptions{})
		return err
	}).err
	return containers, err
}

// GetLogs retrieves logs from a container
func (d *DockerFluent) GetLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	var logs io.ReadCloser
	err := d.Bind(func(cli *client.Client) error {
		var err error
		logs, err = cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
		return err
	}).err
	return logs, err
}

// Error returns any error that occurred during the chain of operations
func (d *DockerFluent) Error() error {
	return d.err
}

// ExecResult represents the result of an exec operation
type ExecResult struct {
	ContainerID string
	Output      string
	Error       error
}

// ConcurrentExec executes commands in multiple containers concurrently
func (d *DockerFluent) ConcurrentExec(ctx context.Context, execConfigs map[string]types.ExecConfig) []ExecResult {
	results := make([]ExecResult, 0, len(execConfigs))
	var wg sync.WaitGroup
	resultChan := make(chan ExecResult, len(execConfigs))

	for containerID, config := range execConfigs {
		wg.Add(1)
		go func(cID string, cfg types.ExecConfig) {
			defer wg.Done()
			result := ExecResult{ContainerID: cID}

			execID, err := d.Client.ContainerExecCreate(ctx, cID, cfg)
			if err != nil {
				result.Error = err
				resultChan <- result
				return
			}

			resp, err := d.Client.ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{})
			if err != nil {
				result.Error = err
				resultChan <- result
				return
			}
			defer resp.Close()

			output, err := io.ReadAll(resp.Reader)
			if err != nil {
				result.Error = err
			} else {
				result.Output = string(output)
			}

			resultChan <- result
		}(containerID, config)
	}

	wg.Wait()
	close(resultChan)

	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// ConcurrentStartContainers starts multiple containers concurrently
func (d *DockerFluent) ConcurrentStartContainers(ctx context.Context, containerIDs ...string) *DockerFluent {
	operations := make([]func(*client.Client) error, len(containerIDs))
	for i, containerID := range containerIDs {
		containerID := containerID // Create a new variable to avoid closure issues
		operations[i] = func(cli *client.Client) error {
			return cli.ContainerStart(ctx, containerID, container.StartOptions{})
		}
	}
	return d.ConcurrentBind(operations...)
}

// ConcurrentStopContainers stops multiple containers concurrently
func (d *DockerFluent) ConcurrentStopContainers(ctx context.Context, containerIDs ...string) *DockerFluent {
	operations := make([]func(*client.Client) error, len(containerIDs))
	for i, containerID := range containerIDs {
		containerID := containerID // Create a new variable to avoid closure issues
		operations[i] = func(cli *client.Client) error {
			return cli.ContainerStop(ctx, containerID, container.StopOptions{})
		}
	}
	return d.ConcurrentBind(operations...)
}

// PullImage pulls a Docker image with detailed progress output
func (d *DockerFluent) PullImage(ctx context.Context, img string) *DockerFluent {
	return d.Bind(func(cli *client.Client) error {
		out, err := cli.ImagePull(ctx, img, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("failed to pull img %s: %w", img, err)
		}
		defer out.Close()

		decoder := json.NewDecoder(out)
		for {
			var message map[string]interface{}
			if err := decoder.Decode(&message); err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("error decoding pull message: %w", err)
			}
			log.Printf("Pull progress: %v", message)
		}

		log.Printf("Successfully pulled img: %s", img)
		return nil
	})
}

// ConcurrentPullImages Update ConcurrentPullImages similarly
func (d *DockerFluent) ConcurrentPullImages(ctx context.Context, images ...string) *DockerFluent {
	operations := make([]func(*client.Client) error, len(images))
	for i, img := range images {
		im := img // Create a new variable to avoid closure issues
		operations[i] = func(cli *client.Client) error {
			return d.PullImage(ctx, im).Error()
		}
	}
	return d.ConcurrentBind(operations...)
}

// InspectResult represents the comprehensive inspection result
type InspectResult struct {
	Container *types.ContainerJSON  `json:"container,omitempty"`
	Image     types.ImageInspect    `json:"image,omitempty"`
	Network   types.NetworkResource `json:"network,omitempty"`
	Volume    volume.Volume         `json:"volume,omitempty"`
	Logs      string                `json:"logs,omitempty"`
	Stats     types.Stats           `json:"stats,omitempty"`
	Processes [][]string            `json:"processes,omitempty"`
	//Ports       []types.Port          `json:"ports,omitempty"`
	Ports       nat.PortMap        `json:"ports,omitempty"`
	Mounts      []types.MountPoint `json:"mounts,omitempty"`
	Environment []string           `json:"environment,omitempty"`
}

// EnhancedInspect provides a comprehensive inspection of a Docker object
func (d *DockerFluent) EnhancedInspect(ctx context.Context, id string) (*InspectResult, error) {
	var result InspectResult

	// Inspect Container
	ctr, err := d.Client.ContainerInspect(ctx, id)
	if err == nil {
		result.Container = &ctr
		result.Mounts = ctr.Mounts
		result.Ports = ctr.NetworkSettings.Ports
		result.Environment = ctr.Config.Env
	}

	// Inspect Image
	image, _, err := d.Client.ImageInspectWithRaw(ctx, id)
	if err == nil {
		result.Image = image
	}

	// Inspect Network
	network, err := d.Client.NetworkInspect(ctx, id, types.NetworkInspectOptions{})
	if err == nil {
		result.Network = network
	}

	// Inspect Volume
	volume, err := d.Client.VolumeInspect(ctx, id)
	if err == nil {
		result.Volume = volume
	}

	// Get Container Logs
	if result.Container != nil {
		logs, err := d.Client.ContainerLogs(ctx, id, container.LogsOptions{ShowStdout: true, ShowStderr: true})
		if err == nil {
			defer logs.Close()
			logContent, _ := io.ReadAll(logs)
			result.Logs = string(logContent)
		}
	}

	// Get Container Stats
	if result.Container != nil {
		stats, err := d.Client.ContainerStats(ctx, id, false)
		if err == nil {
			defer stats.Body.Close()
			json.NewDecoder(stats.Body).Decode(&result.Stats)
		}
	}

	// Get Container Processes
	if result.Container != nil {
		processes, err := d.Client.ContainerTop(ctx, id, []string{})
		if err == nil {
			result.Processes = processes.Processes
		}
	}

	return &result, nil
}

// PrettyPrint outputs the InspectResult in a formatted JSON
func (r *InspectResult) PrettyPrint() string {
	jsonData, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting inspect result: %v", err)
	}
	return string(jsonData)
}

// AIChainFunc is a function type that processes AI responses and returns a DockerFluent operation
type AIChainFunc func(aiResponse string) func(*DockerFluent) *DockerFluent

// AIChain sends the current state to OpenAI and processes the response
func (d *DockerFluent) AIChain(ctx context.Context, prompt string, aiChainFunc AIChainFunc) *DockerFluent {
	if d.err != nil {
		return d
	}

	token := os.Getenv("OPENAI_API_KEY")

	openAIClient := openai.NewClient(token) // Replace with your actual API key

	// Prepare the current state for AI analysis
	state, err := d.getState(ctx)
	if err != nil {
		d.err = fmt.Errorf("error getting Docker state: %w", err)
		return d
	}

	fullPrompt := fmt.Sprintf("%s\n\nCurrent Docker State:\n%s", prompt, state)

	resp, err := openAIClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fullPrompt,
				},
			},
		},
	)

	if err != nil {
		d.err = fmt.Errorf("error from OpenAI API: %w", err)
		return d
	}

	aiResponse := resp.Choices[0].Message.Content

	// Process the AI response and continue the chain
	nextOperation := aiChainFunc(aiResponse)
	return nextOperation(d)
}

// getState retrieves the current Docker state
func (d *DockerFluent) getState(ctx context.Context) (string, error) {
	containers, err := d.Client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return "", err
	}

	images, err := d.Client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return "", err
	}

	state := struct {
		Containers []types.Container `json:"containers"`
		Images     []image.Summary   `json:"images"`
	}{
		Containers: containers,
		Images:     images,
	}

	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", err
	}

	return string(stateJSON), nil
}

// BulkInspectAndAnalyze inspects all containers and sends the results to OpenAI for analysis
func (d *DockerFluent) BulkInspectAndAnalyze(ctx context.Context) *DockerFluent {
	if d.err != nil {
		return d
	}

	// Get list of all containers
	containers, err := d.Client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		d.err = fmt.Errorf("error listing containers: %w", err)
		return d
	}

	// Collect inspect results for all containers
	inspectResults := make(map[string]*InspectResult)
	for _, container := range containers {
		result, err := d.EnhancedInspect(ctx, container.ID)
		if err != nil {
			d.err = fmt.Errorf("error inspecting container %s: %w", container.ID, err)
			return d
		}
		inspectResults[container.ID] = result
	}

	// Convert inspect results to JSON
	inspectJSON, err := json.MarshalIndent(inspectResults, "", "  ")
	if err != nil {
		d.err = fmt.Errorf("error marshaling inspect results: %w", err)
		return d
	}

	// Prepare the prompt for OpenAI
	prompt := fmt.Sprintf("Analyze the following Docker inspect results for multiple containers. Identify any potential issues, misconfigurations, or areas for improvement for each container. Also, provide an overall assessment of the Docker environment:\n\n%s", string(inspectJSON))

	// Send to OpenAI for analysis
	aiResponse, err := d.getAIAnalysis(ctx, prompt)
	if err != nil {
		d.err = fmt.Errorf("error getting AI analysis: %w", err)
		return d
	}

	// Print the AI analysis
	fmt.Printf("AI Analysis of Containers:\n%s\n", aiResponse)

	return d
}

func (d *DockerFluent) getAIAnalysis(ctx context.Context, prompt string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	openAIClient := openai.NewClient(apiKey)

	resp, err := openAIClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an expert Docker container analyst. Your task is to review Docker inspect results for multiple containers and identify potential issues, misconfigurations, or areas for improvement. Provide a detailed analysis for each container and an overall assessment of the Docker environment.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("error from OpenAI API: %w", err)
	}

	return resp.Choices[0].Message.Content, nil
}

func (d *DockerFluent) WaitForHealthy(ctx context.Context, containerID string, timeout time.Duration) *DockerFluent {
	if d.err != nil {
		return d
	}

	if d.Client == nil {
		d.err = fmt.Errorf("Docker client is nil")
		return d
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			d.err = fmt.Errorf("timeout waiting for container %s to become healthy", containerID)
			return d
		default:
			container, err := d.Client.ContainerInspect(ctx, containerID)
			if err != nil {
				d.err = fmt.Errorf("error inspecting container %s: %w", containerID, err)
				return d
			}

			if container.State != nil && container.State.Health != nil {
				if container.State.Health.Status == "healthy" {
					return d
				}
			} else {
				// If the container doesn't have health check configured, consider it healthy
				return d
			}

			time.Sleep(1 * time.Second)
		}
	}
}

// BuildImage builds a Docker image from a Dockerfile
func (d *DockerFluent) BuildImage(ctx context.Context, dockerfileContent string, tags ...string) *DockerFluent {
	if d.err != nil {
		return d
	}

	// Create a temporary directory for the Dockerfile
	tempDir, err := os.MkdirTemp("", "dockerfile-")
	if err != nil {
		d.err = fmt.Errorf("error creating temporary directory: %w", err)
		return d
	}
	defer os.RemoveAll(tempDir)

	// Write the Dockerfile content to a file
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		d.err = fmt.Errorf("error writing Dockerfile: %w", err)
		return d
	}

	// Create a tar archive of the Dockerfile
	tar, err := archive.TarWithOptions(tempDir, &archive.TarOptions{})
	if err != nil {
		d.err = fmt.Errorf("error creating tar archive: %w", err)
		return d
	}

	// Build the image
	resp, err := d.Client.ImageBuild(ctx, tar, types.ImageBuildOptions{
		Tags:       tags,
		Dockerfile: "Dockerfile",
	})
	if err != nil {
		d.err = fmt.Errorf("error building image: %w", err)
		return d
	}
	defer resp.Body.Close()

	// Read the response
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		d.err = fmt.Errorf("error reading build response: %w", err)
		return d
	}

	return d
}

// CreateNetwork creates a Docker network
func (d *DockerFluent) CreateNetwork(ctx context.Context, name string) *DockerFluent {
	if d.err != nil {
		return d
	}

	_, err := d.Client.NetworkCreate(ctx, name, types.NetworkCreate{
		//CheckDuplicate: true,
		Driver: "bridge",
	})
	if err != nil {
		d.err = fmt.Errorf("error creating network %s: %w", name, err)
		return d
	}

	return d
}

// RemoveNetwork removes a Docker network
func (d *DockerFluent) RemoveNetwork(ctx context.Context, name string) *DockerFluent {
	if d.err != nil {
		return d
	}

	err := d.Client.NetworkRemove(ctx, name)
	if err != nil {
		d.err = fmt.Errorf("error removing network %s: %w", name, err)
		return d
	}

	return d
}

func (d *DockerFluent) GenerateContainerSVG(inspectResult *types.ContainerJSON) string {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="800" height="600">
		<style>
			.container { fill: #f0f0f0; stroke: #333; stroke-width: 2; }
			.label { font-family: Arial; font-size: 14px; }
			.port { fill: #4CAF50; }
			.volume { fill: #2196F3; }
			.network { fill: #FFC107; }
		</style>
		<rect class="container" x="50" y="50" width="700" height="500" rx="10" ry="10" />`

	// Container Name and ID
	svg += fmt.Sprintf(`<text x="400" y="30" text-anchor="middle" class="label" font-weight="bold">Container: %s</text>`, inspectResult.Name)
	svg += fmt.Sprintf(`<text x="70" y="80" class="label">ID: %s</text>`, inspectResult.ID[:12])

	// Image
	svg += fmt.Sprintf(`<text x="70" y="110" class="label">Image: %s</text>`, inspectResult.Config.Image)

	// State
	stateColor := "#4CAF50" // Green for running
	if !inspectResult.State.Running {
		stateColor = "#F44336" // Red for stopped
	}
	svg += fmt.Sprintf(`<circle cx="700" cy="80" r="10" fill="%s" />`, stateColor)
	svg += fmt.Sprintf(`<text x="720" y="85" class="label">%s</text>`, inspectResult.State.Status)

	// Ports
	svg += `<text x="70" y="150" class="label" font-weight="bold">Ports:</text>`
	var i int
	for privatePort, bindings := range inspectResult.NetworkSettings.Ports {
		for _, binding := range bindings {
			svg += fmt.Sprintf(`<circle class="port" cx="%d" cy="180" r="5" />
				<text x="%d" y="200" class="label">%s -> %s</text>`,
				100+i*120, 100+i*120, privatePort, binding.HostPort)
			i++
			if i > 5 {
				break // Limit to 6 ports for space
			}
		}
		if i > 5 {
			break
		}
	}

	// Volumes
	svg += `<text x="70" y="240" class="label" font-weight="bold">Volumes:</text>`
	for i, mount := range inspectResult.Mounts {
		if i > 2 {
			break // Limit to 3 volumes for space
		}
		svg += fmt.Sprintf(`<rect class="volume" x="70" y="%d" width="40" height="20" />
			<text x="120" y="%d" class="label">%s -> %s</text>`,
			270+i*30, 285+i*30, strings.Split(mount.Source, "/")[len(strings.Split(mount.Source, "/"))-1], mount.Destination)
	}

	// Networks
	svg += `<text x="70" y="380" class="label" font-weight="bold">Networks:</text>`
	var j int
	for networkName, networkInfo := range inspectResult.NetworkSettings.Networks {
		svg += fmt.Sprintf(`<circle class="network" cx="%d" cy="410" r="5" />
			<text x="%d" y="430" class="label">%s: %s</text>`,
			100+j*200, 100+j*200, networkName, networkInfo.IPAddress)
		j++
		if j > 2 {
			break // Limit to 3 networks for space
		}
	}

	// Environment Variables (limited to first 3 for space)
	svg += `<text x="70" y="470" class="label" font-weight="bold">Env Variables:</text>`
	for i, env := range inspectResult.Config.Env {
		if i > 2 {
			break
		}
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			svg += fmt.Sprintf(`<text x="70" y="%d" class="label">%s: %s</text>`, 500+i*20, parts[0], parts[1])
		}
	}

	svg += `</svg>`
	return svg
}

// ContainerSorter is a function type for comparing two containers
type ContainerSorter func(c1, c2 types.Container) bool

// Sort sorts the containers based on the provided sorter function
func (d *DockerFluent) Sort(ctx context.Context, sorter ContainerSorter) *DockerFluent {
	if d.err != nil {
		return d
	}

	containers, err := d.Client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		d.err = fmt.Errorf("error listing containers: %w", err)
		return d
	}

	sort.Slice(containers, func(i, j int) bool {
		return sorter(containers[i], containers[j])
	})

	// You might want to store the sorted containers or perform some action with them
	// For now, we'll just print them
	for _, c := range containers {
		fmt.Printf("Container ID: %s, Name: %s\n", c.ID[:12], c.Names[0])
	}

	return d
}

// ContainerPredicate is a function type for testing a condition on a container
type ContainerPredicate func(c types.Container) bool

// PartitionResult holds the result of a partition operation
type PartitionResult struct {
	Matching    []types.Container
	NonMatching []types.Container
}

// Partition splits containers into two groups based on the provided predicate
func (d *DockerFluent) Partition(ctx context.Context, predicate ContainerPredicate) (*PartitionResult, error) {
	if d.err != nil {
		return nil, d.err
	}

	containers, err := d.Client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("error listing containers: %w", err)
	}

	result := &PartitionResult{
		Matching:    []types.Container{},
		NonMatching: []types.Container{},
	}

	for _, c := range containers {
		if predicate(c) {
			result.Matching = append(result.Matching, c)
		} else {
			result.NonMatching = append(result.NonMatching, c)
		}
	}

	return result, nil
}

// ContainerKeySelector is a function type for selecting a grouping key from a container
type ContainerKeySelector func(c types.Container) string

// GroupByResult holds the result of a group by operation
type GroupByResult map[string][]types.Container

// GroupBy groups containers based on the provided key selector
func (d *DockerFluent) GroupBy(ctx context.Context, keySelector ContainerKeySelector) (GroupByResult, error) {
	if d.err != nil {
		return nil, d.err
	}

	containers, err := d.Client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("error listing containers: %w", err)
	}

	result := make(GroupByResult)

	for _, c := range containers {
		key := keySelector(c)
		result[key] = append(result[key], c)
	}

	return result, nil
}
