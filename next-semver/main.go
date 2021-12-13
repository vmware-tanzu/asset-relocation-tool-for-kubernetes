package main

import (
	"fmt"
	"os"

	semverv3 "github.com/Masterminds/semver/v3"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s {semver-version}\n$ %s 1.2.3\n", os.Args[0], os.Args[0])
		os.Exit(-1)
	}
	v := os.Args[1]
	version := semverv3.MustParse(v)
	next := version.IncPatch()
	fmt.Println(next)
}
