package main

import "fmt"

const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

var skipDirs = map[string]bool{
	"node_modules": true,
	"venv":         true,
	"__pycache__":  true,
	"target":       true,
}

func main() {
	fmt.Println("Hello, Go")
}

