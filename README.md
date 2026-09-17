# sveda-go-sdk

Go SDK for the [Sveda AI](https://github.com/neresson/sveda) sidecar HTTP API.

Module: `github.com/neresson/sveda-go-sdk`

## Install

```bash
go get github.com/neresson/sveda-go-sdk
```

## Usage

```go
package main

import (
	"context"

	sveda "github.com/neresson/sveda-go-sdk"
)

func main() {
	client := sveda.New(sveda.Config{
		BaseURL:    "https://sveda.example.com",
		HostAPIKey: hostKey,
	})
	token, _ := client.Embed.CreateToken(context.Background(), sveda.TokenRequest{
		VisitorID: "user-1",
	})

	session, _ := sveda.StartHostSession(context.Background(), sveda.Config{
		BaseURL:    "https://sveda.example.com",
		HostAPIKey: hostKey,
	}, "user-1")
	_ = token
	_ = session
}
```

## License

MIT
