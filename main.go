package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
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
	total int
	exports map[string]int
}

type fileContents struct {
	// Map of imports from given files/locations
	path     string
	imprts   map[string][]string
	contents []string
}

// Contains all fo the symbols that we need to export for the packages that we have replaced
var pkgExports map[string]map[string][]string

var importRegex *regexp.Regexp

var topLevel string

func main() {

	importRegex = regexp.MustCompile("^import\\s+{\\s+([a-zA-Z,\\s]*)\\s+}\\s+from\\s+'(.*)';")

	importFiles = make(map[string]*exportsInfo)

	pkgExports = make(map[string]map[string][]string)

	// Process all of the .ts files (not .spec.ts)
	folder := "/Users/nwm/dev/a9/store-core/src/frontend/packages/core"
	//folder := "/Users/nwm/dev/a9/core-sep/src/frontend/packages/store"
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


	//test := "/Users/nwm/dev/a9/store-core/src/frontend/packages/cloud-foundry/src/shared/components/list/list-types/cf-select-users/cf-select-users-list-config.service.ts"

	base := "/Users/nwm/dev/a9/store-core/src/frontend/packages/store/src"


	for _, file := range files {
		f, _ := readFileContents(file)
		replaceImports(f, base, "@stratosui/store")
	}
	//writeFile(f)
	//writeContents(f)

	// Now write all of the exports we need to add to the public-api file
	keyword := "export"
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	for pkg, pkgs := range pkgExports {
		fmt.Println(pkg)
		for file, symbols := range pkgs {
			rel, _ := filepath.Rel(base, file)
			file = "./" + rel
			sort.Strings(symbols)

			line := formatImportOnSingleLine(keyword, file, symbols)
			if len(line) > lineSplitLength {
				line = formatMultilineImport(keyword, file, symbols)
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
			total: 0,
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
		path: filePath,
		imprts:  make(map[string][]string),
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
			if strings.HasSuffix(txt, ";") {
				// Single-line import
				processImport(filePath, txt);
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

		if (!strings.HasPrefix(full, topLevel)) {
			info := getExportInfo(full)
			info.total = info.total + 1
			for _, export := range(strings.Split(exportNames, ",")) {
				name := strings.TrimSpace(export)
				if len(name) > 0 {
					if _, ok := info.exports[name]; !ok {
						info.exports[name] = 0
					}
					info.exports[name] = info.exports[name]+1
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
		for _, export := range(strings.Split(exportNames, ",")) {
			name := strings.TrimSpace(export)
			if len(name) > 0 {
				f.imprts[full] = append(f.imprts[full], name)
			}
		}
	}
}
