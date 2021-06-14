package compiler

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

type file struct {
	name string
	path string
}

func getYamlList(root string) ([]file, error) {
	var files []file
	err := filepath.Walk(root, func(currentPath string, info fs.FileInfo, err error) error {
		if info == nil {
			return nil
		}

		if info.IsDir() {
			list, err := getYamlList(path.Join(currentPath, info.Name()))
			if err != nil {
				return err
			}

			files = append(files, list...)
			return nil
		}

		fileName := info.Name()
		if !strings.HasSuffix(fileName, ".hermod.yaml") {
			return nil
		}

		files = append(files, file{
			name: fileName,
			path: path.Dir(currentPath),
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
