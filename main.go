package main

import (
	"fmt"
)

/*
	cdflow release:
		terraform container: terraform init (no config container dependency yet)
		config container: get environment for artefact build & publish (e.g. aws creds for docker push)
		release container: do build and publish
		config container: upload release archive

*/
func main() {
	fmt.Println("hello world")
}
