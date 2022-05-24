package compiler

import (
	"github.com/iancoleman/strcase"
	"log"
	"os"
	"path"
	"strings"
)

type fileConfigPair struct {
	config *config
	file   file
}

func CompileFiles(in, out, packageName, acronyms string) {
	// configure strcase acronyms
	// the ID acronym is for common usage with GORM and other ORMs. you can override it by passing ID=id.
	strcase.ConfigureAcronym("ID", "ID")
	if acronyms != "" {
		for _, mapping := range strings.Split(acronyms, ",") {
			splitString := strings.Split(mapping, "=")
			if len(splitString) != 2 {
				log.Fatalln("Couldn't parse acronym mappings")
			}
			strcase.ConfigureAcronym(splitString[0], splitString[1])
		}
	}

	files, err := getYamlList(in)
	if err != nil {
		log.Println("Failed to find list of files to compile.")
		log.Fatalln(err)
	}

	var configs []*fileConfigPair
	for _, file := range files {
		contents, err := os.ReadFile(path.Join(file.path, file.name))
		if err != nil {
			log.Printf("Couldn't read file %s", file.name)
			log.Fatalln(err)
		}

		data, err := parseFile(contents)
		if err != nil {
			log.Printf("Failed to parse YAML in file %s", file.name)
			log.Fatalln(err)
		}
		configs = append(configs, &fileConfigPair{
			config: data,
			file:   file,
		})
	}

	for _, pair := range configs {
		err = outputConfig(pair.config, pair.file, configs, out, packageName)
		if err != nil {
			log.Printf("Failed to generate output for file %s", pair.file.name)
			log.Fatalln(err)
		}
	}
}
