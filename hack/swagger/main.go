package main

import (
	"os"
)

func main() {
	switch os.Args[1] {
	case "kubeifyswagger":
		kubeifySwagger(os.Args[2], os.Args[3])
	default:
		panic(os.Args[1])
	}
}
