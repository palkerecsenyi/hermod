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
	flag.Parse()

	if *inputPath == "" {
		log.Fatalln("--in must be specified")
	}
	if *outputPath == "" {
		log.Fatalln("--out must be specified")
	}

	compiler.CompileFiles(*inputPath, *outputPath, *packageName)
}
