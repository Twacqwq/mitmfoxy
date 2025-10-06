package main

import (
	"context"
	"log"

	"github.com/Twacqwq/mitmfoxy/cmd"
)

func main() {
	ctx := context.Background()
	if err := cmd.Execute(ctx); err != nil {
		log.Fatal(err)
	}
}
