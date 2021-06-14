package compiler

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
)

func _write(f *bytes.Buffer, data string) {
	f.WriteString(data)
}

func _writeln(f *bytes.Buffer, data string) {
	_write(f, data+"\n")
}

func _writelni(f *bytes.Buffer, indent int, data string) {
	indentString := ""
	for i := 0; i < indent; i++ {
		indentString += "\t"
	}

	_writeln(f, indentString+data)
}

func uniqifyImportSlice(imports []string) []string {
	keys := make(map[string]bool)
	var list []string

	for _, entry := range imports {
		if _, ok := keys[entry]; !ok {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func outputConfig(config *config, file file, configs []*fileConfigPair, outPath, packageName string) error {
	goFileName := strings.Split(file.name, ".hermod.yaml")[0] + ".go"

	f, err := os.OpenFile(path.Join(outPath, goFileName), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	_ = f.Truncate(0)

	var imports []string
	var contentBuffer bytes.Buffer
	for _, unit := range config.Units {
		imports, err = writeUnit(configs, &contentBuffer, &unit, packageName)
		if err != nil {
			return err
		}
	}

	for _, service := range config.Services {
		imports = append(imports, writeService(&contentBuffer, &service, packageName)...)
	}

	var importBuffer bytes.Buffer
	_writeln(&importBuffer, "// GENERATED FILE — DO NOT EDIT")
	_writeln(&importBuffer, fmt.Sprintf("package %s", config.Package))

	_writeln(&importBuffer, "import (")
	imports = uniqifyImportSlice(imports)
	sort.Strings(imports)
	for _, i := range uniqifyImportSlice(imports) {
		_writelni(&importBuffer, 1, fmt.Sprintf("\"%s\"", i))
	}
	_writeln(&importBuffer, ")")

	_, _ = f.Seek(0, 0)
	_, _ = importBuffer.WriteTo(f)
	_, _ = contentBuffer.WriteTo(f)
	_ = f.Sync()
	return nil
}
