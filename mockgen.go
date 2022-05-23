package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/types"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

var errNotFound = errors.New("entity not found")

var structNameRegex = regexp.MustCompile(`type (.*) struct \{`)
var preambleRegex = regexp.MustCompile(`(?s)^(.*type .*? struct \{\n\tmock.Mock\n\}\n\n)(.*)$`)
var signatureRegex = regexp.MustCompile(`func \(..? \*.*?\) (.*) \{`)

type functionBlock struct {
	fullFunction string
	signature    string
}

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage of %s\n\n", os.Args[0])
		fmt.Printf("--- Generate a new mock file ---\n")
		fmt.Printf("%s <folderPath> <interfaceName> <mockStructName>\n", os.Args[0])
		fmt.Printf("Example: %s ./pkg/gateway Gatewayer Gateway\n\n", os.Args[0])

		fmt.Println("--- Update an existing mock file ---")
		fmt.Printf("%s <folderPath> <interfaceName> -f\n", os.Args[0])
		fmt.Printf("Example: %s ./pkg/gateway Gatewayer -f\n\n", os.Args[0])
	}

	flag.Parse()

	if os.Args[3] != "-f" {
		fmt.Println("Generating new mock...")
		err := generateNewMock(os.Args[1], os.Args[2], os.Args[3])
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println("Success!")
	} else {
		fmt.Println("Updating existing mock...")
		err := updateMock(os.Args[1], os.Args[2])
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println("Success!")
	}
}

func generateNewMock(interfaceFile, interfaceName, structName string) error {
	iface, err := findInterface(interfaceFile, interfaceName)
	if err != nil {
		return errors.Wrap(err, "problem finding interface")
	}

	path := filepath.Dir(interfaceFile)
	// Create mock folder if it does not exist.
	folderPath := folderPath(path, iface)
	if _, err = os.Stat(folderPath); os.IsNotExist(err) {
		err = os.Mkdir(folderPath, os.ModePerm)
		if err != nil {
			return errors.Wrap(err, "failed to create directory")
		}
	}

	// Check if the mock file already exists.
	filePath := folderPath + "/" + strings.ToLower(structName) + ".go"
	if _, err = os.Stat(filePath); err == nil {
		return fmt.Errorf("file %s already exists, delete it before re-generating it", filePath)
	} else if !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to determine whether file exists")
	}

	// Create file to write the mock into.
	f, err := os.Create(filePath) //nolint: gosec
	if err != nil {
		return errors.Wrap(err, "failed to create file")
	}
	defer close(f)

	err = generateMock(iface, structName, f)
	if err != nil {
		return errors.Wrap(err, "failed to generate mock")
	}

	return nil
}

func updateMock(interfaceFile, interfaceName string) error {
	iface, err := findInterface(interfaceFile, interfaceName)
	if err != nil {
		return errors.Wrap(err, "problem finding interface")
	}

	path := filepath.Dir(interfaceFile)
	existingMock, pathToExistingFile, err := readExistingFiles(path, iface)
	if err != nil {
		return errors.Wrap(err, "failed to read existing file(s)")
	}

	structName, err := extractStructName(existingMock)
	if err != nil {
		return errors.Wrap(err, "failed to find struct name")
	}

	buf := &bytes.Buffer{}
	err = generateMock(iface, structName, buf)
	if err != nil {
		return errors.Wrap(err, "failed to generate virtual mock")
	}
	virtualNewMock := buf.String()

	result, err := calcResultMock(existingMock, virtualNewMock)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(pathToExistingFile, []byte(result), os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "failed to write result in file")
	}

	return nil
}

// findInterface loads the package that the given interfaceFile is part of and retrieves the interface data
// for the interface with the provided name.
func findInterface(interfaceFile, interfaceName string) (*Interface, error) {
	absPath, err := filepath.Abs(interfaceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to find absolute file path: %w", err)
	}

	mode := packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports
	mode = mode | packages.NeedDeps | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo
	pkgs, err := packages.Load(&packages.Config{Mode: mode}, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package with import path: %w", err)
	}

	if len(pkgs) != 1 {
		return nil, fmt.Errorf("invalid number of packages found: %d", len(pkgs))
	}

	obj := pkgs[0].Types.Scope().Lookup(interfaceName)
	if obj == nil {
		fmt.Println("lookup failed")
		return nil, errNotFound
	}

	typ, ok := obj.Type().(*types.Named)
	if !ok {
		fmt.Println("cast of Type failed")
		return nil, errNotFound
	}

	i := &Interface{
		Name:          interfaceName,
		Pkg:           pkgs[0].Types,
		QualifiedName: pkgs[0].Types.Path(),
		FileName:      absPath,
		NamedType:     typ,
	}

	iface, ok := typ.Underlying().(*types.Interface)
	if ok {
		i.IsFunction = false
		i.ActualInterface = iface
	} else {
		sig, ok := typ.Underlying().(*types.Signature)
		if !ok {
			fmt.Println("cast to signature failed")
			return nil, errNotFound
		}
		i.IsFunction = true
		i.SingleFunction = &Method{Name: "Execute", Signature: sig}
	}

	return i, nil
}

func calcResultMock(existingMock string, newMock string) (string, error) {
	existingFunctions, _, err := extractBlocksAndPreamble(existingMock)
	if err != nil {
		return "", errors.Wrap(err, "extraction failed for existing file")
	}
	newFunctions, newPreamble, err := extractBlocksAndPreamble(newMock)
	if err != nil {
		return "", errors.Wrap(err, "extraction failed for new mock")
	}

	bodyFunctions := combineFunctions(existingFunctions, newFunctions)
	bodyStr := ""
	for _, fn := range bodyFunctions {
		bodyStr += fn.fullFunction + "\n\n"
	}
	bodyStr = strings.TrimSuffix(bodyStr, "\n")
	return newPreamble + bodyStr, nil
}

func generateMock(iface *Interface, structName string, out io.Writer) error {
	gen := NewGenerator(iface, structName)
	gen.GeneratePrologue(context.TODO(), iface.Pkg.Name())
	err := gen.Generate(context.TODO())
	if err != nil {
		return err
	}

	return gen.Write(out)
}

func close(resource io.Closer) {
	err := resource.Close()
	if err != nil {
		log.Print(err)
	}
}

func folderPath(path string, iface *Interface) string {
	return path + "/" + iface.Pkg.Name() + "mock"
}

func readExistingFiles(path string, iface *Interface) (string, string, error) {
	var files []string
	err := filepath.Walk(folderPath(path, iface), func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		return "", "", err
	}

	for _, file := range files {
		// nolint: gosec
		blob, err := ioutil.ReadFile(file)
		if err != nil {
			return "", "", err
		}
		content := string(blob)
		if strings.Contains(content, iface.Pkg.Name()+"."+iface.Name) {
			return content, file, nil
		}
	}

	return "", "", errNotFound
}

func extractStructName(content string) (string, error) {
	matches := structNameRegex.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", errNotFound
	}

	return matches[1], nil
}

func extractBlocksAndPreamble(input string) ([]functionBlock, string, error) {
	bodyMatches := preambleRegex.FindStringSubmatch(input)
	if len(bodyMatches) < 3 {
		return nil, "", errors.New("failed to find preamble and functions")
	}

	functions := strings.SplitAfter(bodyMatches[2], "\n}\n\n")

	blocks := []functionBlock{}
	for _, fn := range functions {
		if strings.HasSuffix(fn, "\n\n") {
			fn = strings.TrimRight(fn, "\n")
		}

		matches := signatureRegex.FindStringSubmatch(fn)
		if len(matches) < 2 {
			return nil, "", errors.New("failed to find function signature: " + fn)
		}
		block := functionBlock{
			fullFunction: fn,
			signature:    matches[1],
		}
		blocks = append(blocks, block)
	}

	return blocks, bodyMatches[1], nil
}

func combineFunctions(existingFunctions []functionBlock, newFunctions []functionBlock) []functionBlock {
	for i, fn := range newFunctions {
		if foundFunction := findFunction(existingFunctions, fn.signature); foundFunction != nil {
			newFunctions[i] = *foundFunction
		}
	}

	result := []functionBlock{}
	new := []functionBlock{}
	// Collect all functions from newFunctions in the order they appear in the existing list.
	// By keeping the order we can keep git diffs minimal.
	for _, fn := range existingFunctions {
		matches := strings.SplitN(fn.signature, "(", 2)
		foundFunction := findFunction(newFunctions, matches[0])
		if foundFunction != nil {
			result = append(result, *foundFunction)
		}
	}

	// Collect all new functions that are still missing in the result from above.
	for _, fn := range newFunctions {
		matches := strings.SplitN(fn.signature, "(", 2)
		foundFunction := findFunction(result, matches[0])
		if foundFunction == nil {
			new = append(new, fn)
		}
	}
	return append(result, new...)
}

func findFunction(functions []functionBlock, signaturePrefix string) *functionBlock {
	for _, fn := range functions {
		if strings.HasPrefix(fn.signature, signaturePrefix) {
			return &fn
		}
	}
	return nil
}
