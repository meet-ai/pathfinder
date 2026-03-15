package main

import (
	"flag"
	"fmt"
	"os"

	"pathfinder/internal/app"
)

func main() {
	var message string
	flag.StringVar(&message, "m", "", "任务描述（目标）")
	flag.Parse()

	if message == "" {
		fmt.Fprintln(os.Stderr, "用法: pathfinder -m \"任务描述\"")
		flag.Usage()
		os.Exit(1)
	}

	if err := app.Run(message); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
