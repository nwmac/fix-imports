package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func replaceImports(f *fileContents, folder, pkg string) {

	// The new set of imports
	imprts := make(map[string][]string)

	// Look for any imports that need replacing
	for file, symbols := range f.imprts {
		name := file
		if strings.HasPrefix(file, ".") {
			dir := filepath.Dir(f.path)
			p := filepath.Join(dir, file)
			if strings.HasPrefix(p, folder) {
				name = pkg
				file = filepath.Join(filepath.Dir(f.path), file)

				// We are replacing
				for _, symbol := range symbols {
					addPackageExport(pkg, file, symbol)
				}
			}
		}
		if _, ok := imprts[name]; !ok {
			imprts[name] = make([]string, 0)
		}
		for _, symbol := range symbols {
			imprts[name] = append(imprts[name], symbol)
		}
	}

	for name, values := range imprts {
		fmt.Println(name)
		for _, v := range values {
			fmt.Println("  " + v)
		}
	}

	// Replace imports
	f.imprts = imprts
}

func addPackageExport(pkg, file, symbol string) {

	if _, ok := pkgExports[pkg]; !ok {
		pkgExports[pkg] = make(map[string][]string)
	}

	info := pkgExports[pkg]
	if _, ok := info[file]; !ok {
		info[file] = make([]string, 0)
	}
	symbols := info[file]
	if !contains(symbols, symbol) {
		info[file] = append(info[file], symbol)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
