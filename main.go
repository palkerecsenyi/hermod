// Package main provides a compiler for Hermod YAML to Go.
//
// For instructions on usage, type:
//     hermod --help
//
// For information about composing YAML files, see README.md
package main

import (
	"flag"
	"github.com/palkerecsenyi/hermod/compiler"
	"log"
)

func main() {
	inputPath := flag.String("in", "", "The path to read .hermod.yaml files from")
	outputPath := flag.String("out", "", "The path to place compiled Go files in")
	packageName := flag.String("package", "github.com/palkerecsenyi/hermod", "The base name of the Go package to use for Hermod")
	acronyms := flag.String("acronyms", "", "A map of acronyms to use with strcase in form: key=value,key=value")

	flag.Parse()

	if *inputPath == "" {
		log.Fatalln("--in must be specified")
	}
	if *outputPath == "" {
		log.Fatalln("--out must be specified")
	}

	compiler.CompileFiles(*inputPath, *outputPath, *packageName, *acronyms)
}
