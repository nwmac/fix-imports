package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func fixImports(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Error reading file %s", filePath)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		txt := scanner.Text()

		if strings.HasPrefix(txt, "import ") {
			// Import statement
			if strings.HasSuffix(txt, ";") {
				// Single-line import
				fixProcessImport(filePath, txt)
			} else {
				ln := txt
				for {
					if !scanner.Scan() {
						break
					}
					txt = scanner.Text()
					ln = fmt.Sprintf("%s %s", ln, txt)
					if strings.HasSuffix(txt, ";") {
						fixProcessImport(filePath, ln)
						break
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Error reading file %s", filePath)
	}
	return nil
}

func fixProcessImport(filePath, txt string) {
	res := importRegex.FindStringSubmatch(txt)
	if len(res) == 3 {
		exportNames := res[1]
		pkg := res[2]

		// Check package path
		folder := filepath.Dir(filePath)
		full := filepath.Join(folder, pkg)

		if !strings.HasPrefix(full, topLevel) {
			info := getExportInfo(full)
			info.total = info.total + 1

			for _, export := range strings.Split(exportNames, ",") {
				name := strings.TrimSpace(export)
				if len(name) > 0 {
					if _, ok := info.exports[name]; !ok {
						info.exports[name] = 0
					}
					info.exports[name] = info.exports[name] + 1
				}
			}
		}
	}
}
