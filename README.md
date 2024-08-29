# DockerFluent

DockerFluent is a Go library that provides a fluent interface for interacting with Docker. It offers a chainable API for
common Docker operations and includes advanced features like sorting, partitioning, and grouping containers, as well as
AI-powered analysis of Docker environments.

## Installation

```bash
go get github.com/vinodhalaharvi/dockerfluent
```

## Usage

Here's a basic example of using DockerFluent:

```go
package main

import (
	"context"
	"log"

	"github.com/vinodhalaharvi/dockerfluent"
	"github.com/docker/docker/api/types/container"
)

func main() {
	ctx := context.Background()
	d, err := dockerfluent.New()
	if err != nil {
		log.Fatal(err)
	}

	err = d.
		PullImage(ctx, "nginx:latest").
		CreateContainer(ctx, &container.Config{
			Image: "nginx:latest",
		}, nil, "my-nginx").
		StartContainer(ctx, "my-nginx").
		Error()

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Nginx container started successfully")
}
```

## Features

- Fluent interface for Docker operations
- Container sorting, partitioning, and grouping
- AI-powered analysis of Docker environments
- Concurrent operations on multiple containers

## Documentation

For full documentation, please visit [pkg.go.dev](https://pkg.go.dev/github.com/vinodhalaharvi/dockerfluent).

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
