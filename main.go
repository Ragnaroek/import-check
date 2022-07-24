package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func main() {

	monoRepoPath, err := os.Getwd() //TODO allow overwrite via flag
	if err != nil {
		panic(err)
	}
	modName, err := readModuleName(monoRepoPath)
	if err != nil {
		panic(err)
	}

	entries, err := os.ReadDir(path.Join(monoRepoPath, "services/"))
	if err != nil {
		panic(err)
	}

	pkgModFolder := path.Join(modName, "pkg/")

	for _, entry := range entries {
		if !entry.IsDir() { // only a folder can be microservice
			continue
		}

		svcPath := path.Join(monoRepoPath, "services/", entry.Name())
		svcScriptPath := path.Join(svcPath, "scripts/")
		svcModFolder := path.Join(modName, "services/", entry.Name())
		unallowedForService := make([]string, 0)
		err := filepath.WalkDir(svcPath, func(path string, fs fs.DirEntry, err error) error {
			if strings.HasPrefix(path, svcScriptPath) {
				return nil
			}
			if !fs.IsDir() && strings.HasSuffix(fs.Name(), ".go") {
				unallowed, err := checkImports(path, modName, pkgModFolder, svcModFolder)
				if err != nil {
					return err
				}
				unallowedForService = append(unallowedForService, unallowed...)
			}
			return nil
		})

		if len(unallowedForService) != 0 {
			fmt.Printf("%s has unallowed import(s)\n", entry.Name())
			for _, ui := range unallowedForService {
				fmt.Printf("\x1b[1;31m\t%s\n\x1b[1;0m", ui)
			}

		}

		if err != nil {
			panic(err)
		}
	}
}

func checkImports(path, modName, pkgModFolder, svcModFolder string) ([]string, error) {
	fset := token.NewFileSet()
	ast, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	unallowed := make([]string, 0)

	for _, imp := range ast.Imports {
		if importHasPrefix(imp, modName) { // only check module internal imports
			if !(importHasPrefix(imp, pkgModFolder) || importHasPrefix(imp, svcModFolder) || importHasSuffix(imp, "pkg/proto")) {
				unallowed = append(unallowed, fmt.Sprintf("uses %s in file %s", imp.Path.Value, path))
			}
		}
	}
	return unallowed, nil
}

func importHasPrefix(imp *ast.ImportSpec, prefix string) bool {
	// The import literal contains the " of the string, that why we have to add " to the prefix for
	// prefix checking
	return strings.HasPrefix(imp.Path.Value, "\""+prefix)
}

func importHasSuffix(imp *ast.ImportSpec, suffix string) bool {
	return strings.HasSuffix(imp.Path.Value, suffix+"\"")
}

func readModuleName(monoRepoPath string) (string, error) {
	bytes, err := os.ReadFile(path.Join(monoRepoPath, "go.mod"))
	if err != nil {
		return "", err
	}

	module, _, found := strings.Cut(string(bytes), "\n")
	if !found {
		return "", fmt.Errorf("invalid go.mod")
	}

	return strings.TrimPrefix(module, "module "), nil
}
