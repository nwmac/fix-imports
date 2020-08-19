package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func writeFile(f *fileContents) {
	file, err := os.Create(f.path)
	if err != nil {
		fmt.Println("ERROR writing file")
		return
	}

	defer file.Close()
	writeToFile(f, file)
}

func writeContents(f *fileContents) {
	writeToFile(f, os.Stdout)
}

func writeToFile(f *fileContents, iow io.Writer) {

	writer := bufio.NewWriter(iow)

	writeImports(writer, f)
	lastLength := -1
	for _, line := range f.contents {
		if !(len(line) == 0 && lastLength <= 0) {
			writer.WriteString(line)
			writer.WriteString("\n")
		}
		lastLength = len(line)
	}

	// if lastLength != 0 {
	// 	writer.WriteString("\n")
	// }

	writer.Flush()
}

func filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func writeImports(writer *bufio.Writer, f *fileContents) {

	var keys []string
	for key := range f.imprts {
		keys = append(keys, key)
		log.Println(key)
	}

	pkgImports := filter(keys, func(i string) bool {
		return !strings.HasPrefix(i, ".")
	})

	fileImports := filter(keys, func(i string) bool {
		return strings.HasPrefix(i, ".")
	})

	writeImportsSection(writer, f, pkgImports)
	writeImportsSection(writer, f, fileImports)
}

func writeImportsSection(writer *bufio.Writer, f *fileContents, items []string) {

	sort.Strings(items)

	keyword := "import"

	for _, file := range items {
		fmt.Println(file)
		imprts := f.imprts[file]
		line := formatImportOnSingleLine(keyword, f.path, file, imprts)
		if len(line) > lineSplitLength {
			line = formatMultilineImport(keyword, f.path, file, imprts)
		}
		writer.WriteString(line)
		writer.WriteString("\n")
	}

	if len(items) > 0 {
		// Blank line sep
		writer.WriteString("\n")
	}
}

func formatMultilineImport(keyword, fileImport, importName string, imprts []string) string {
	res := fmt.Sprintf("%s {\n", keyword)

	for _, symbol := range imprts {
		res = fmt.Sprintf("%s  %s,\n", res, strings.TrimSpace(symbol))
	}
	return fmt.Sprintf("%s} from '%s';", res, importName)
}

func formatImportOnSingleLine(keyword, fileImport, importName string, imprts []string) string {

	symbols := strings.Join(imprts, ", ")
	symbols = strings.TrimSpace(symbols)

	line := fmt.Sprintf("%s { %s } from '%s';", keyword, symbols, importName)
	return line
}

func getImportPath(file, fileImport string) string {
	fmt.Println(fileImport)
	if strings.HasPrefix(fileImport, "/") {
		p, err := filepath.Rel(filepath.Dir(file), fileImport)
		if err == nil {
			if !strings.HasPrefix(p, ".") {
				return "./" + p
			}
			return p
		}
	}
	return fileImport
}
