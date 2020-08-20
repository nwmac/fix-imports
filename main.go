package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Typescript Hero config
const lineSplitLength = 125

var files []string
var importFiles map[string]*exportsInfo

type exportsInfo struct {
	total   int
	exports map[string]int
}

type fileContents struct {
	// Map of imports from given files/locations
	path     string
	imprts   map[string][]string
	extras   []string
	contents []string
}

// Contains all fo the symbols that we need to export for the packages that we have replaced
var pkgExports map[string]map[string][]string

var importRegex *regexp.Regexp

var topLevel string

func main() {

	importRegex = regexp.MustCompile("^import\\s+{\\s+([a-z_A-Z0-9,\\s]*)\\s+}\\s+from\\s+'(.*)';")

	importFiles = make(map[string]*exportsInfo)

	pkgExports = make(map[string]map[string][]string)

	log.Println("Fix Imports")

	if len(os.Args) != 2 {
		log.Println("Need source folder")
		return
	}

	// Process all of the .ts files (not .spec.ts)
	sourceFolder := os.Args[1]

	folder := path.Join(sourceFolder, "src/frontend/packages/core")
	base := path.Join(sourceFolder, "src/frontend/packages/store/src")
	topLevel, _ = filepath.Abs(folder)

	log.Println(topLevel)
	prcoessFolder(folder)

	log.Println(len(files))

	// Process each file in term
	// for _, file := range files {
	// 	log.Println(file)
	// 	//processFile(file)
	// 	//readFileContents(file)

	// }

	// for file, info := range importFiles {
	// 	fmt.Printf("%s,%d\n", file, info.total)
	// 	for key, value := range info.exports {
	// 		fmt.Printf("  - %s,%d\n", key, value)
	// 	}
	// }

	// Test file
	// test := path.Join(folder, "src/shared/components/date-time/date-time.component.ts")

	// f, err := readFileContents(test)
	// if err != nil {
	// 	log.Panic(err)
	// }
	// log.Println(f)
	// replaceImports(f, base, "@stratosui/store")
	// writeFile(f)

	// return

	for _, file := range files {
		f, _ := readFileContents(file)
		replaceImports(f, base, "@stratosui/store")
		writeFile(f)
	}
	//writeContents(f)

	publicAPIFilePath := path.Join(base, "public-api.ts")
	log.Println("Writing update public-api.ts to " + publicAPIFilePath)
	publicAPIFile, err := os.OpenFile(publicAPIFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Panicf("Error reading file %s", publicAPIFilePath)
	}
	defer publicAPIFile.Close()

	// Now write all of the exports we need to add to the public-api file
	keyword := "export"
	writer := bufio.NewWriter(publicAPIFile)
	defer writer.Flush()

	writer.WriteString("\n\n// Auto-generated from fiximports tool\n\n")

	for pkg, pkgs := range pkgExports {
		fmt.Println(pkg)
		for file, symbols := range pkgs {
			rel, _ := filepath.Rel(base, file)
			file = "./" + rel
			sort.Strings(symbols)

			line := formatImportOnSingleLine(keyword, file, file, symbols)
			if len(line) > lineSplitLength {
				line = formatMultilineImport(keyword, file, file, symbols)
			}
			writer.WriteString(line)
			writer.WriteString("\n")

			// fmt.Println(file)
			// fmt.Println(symbols)
		}
	}

}

func getExportInfo(file string) *exportsInfo {
	if _, ok := importFiles[file]; !ok {
		ei := &exportsInfo{
			total:   0,
			exports: make(map[string]int),
		}
		importFiles[file] = ei
	}
	return importFiles[file]
}

func prcoessFolder(folder string) {
	filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(info.Name(), ".ts") && !strings.HasSuffix(info.Name(), ".spec.ts") {
			files = append(files, path)
		}
		return nil
	})
}

func readFileContents(filePath string) (*fileContents, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Error reading file %s", filePath)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	f := &fileContents{
		path:     filePath,
		imprts:   make(map[string][]string),
		contents: nil,
	}

	for scanner.Scan() {
		txt := scanner.Text()
		if strings.HasPrefix(txt, "import ") {
			// Import statement
			if strings.HasSuffix(txt, ";") {
				// Single-line import
				processFileImport(filePath, f, txt)
			} else {
				ln := txt
				for {
					if !scanner.Scan() {
						break
					}
					txt = scanner.Text()
					ln = fmt.Sprintf("%s %s", ln, txt)
					if strings.HasSuffix(txt, ";") {
						processFileImport(filePath, f, ln)
						break
					}
				}
			}
		} else {
			f.contents = append(f.contents, txt)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Error reading file %s", filePath)
	}

	return f, nil
}

func processFile(filePath string) error {
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
			fmt.Println(txt)
			if strings.HasSuffix(txt, ";") {
				// Single-line import
				fmt.Println(txt)
				processImport(filePath, txt)
			} else {
				ln := txt
				for {
					if !scanner.Scan() {
						break
					}
					txt = scanner.Text()
					ln = fmt.Sprintf("%s %s", ln, txt)
					if strings.HasSuffix(txt, ";") {
						processImport(filePath, ln)
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

func processImport(filePath, txt string) {
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

func processFileImport(filePath string, f *fileContents, txt string) {
	res := importRegex.FindStringSubmatch(txt)
	if len(res) == 3 {
		exportNames := res[1]
		pkg := res[2]
		full := pkg

		// Check package path
		// if strings.HasPrefix(pkg, ".") {
		// 	folder := filepath.Dir(filePath)
		// 	full = filepath.Join(folder, pkg)
		// }

		if _, ok := f.imprts[full]; !ok {
			f.imprts[full] = make([]string, 0)
		}
		for _, export := range strings.Split(exportNames, ",") {
			name := strings.TrimSpace(export)
			if len(name) > 0 {
				f.imprts[full] = append(f.imprts[full], name)
			}
		}
	} else {
		// Maybe its of form import * as from _;
		fmt.Println("** IGNORE")
		fmt.Println(txt)
		f.extras = append(f.extras, txt)

	}
}
