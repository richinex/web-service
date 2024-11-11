// cmd/server/main.go

package main

import (
    "context"
    "fmt"
    "os"
    "web-service/internal/server"
)

func main() {
    ctx := context.Background()
    if err := server.Run(ctx, os.Stdout, os.Args, os.Getenv); err != nil {
        fmt.Fprintf(os.Stderr, "%s\n", err)
        os.Exit(1)
    }
}