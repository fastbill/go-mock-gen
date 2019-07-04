package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/build"
	"go/types"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/imports"
)

// ATTENTION: The code in this file is mostly copied from https://github.com/vektra/mockery/blob/2620c4a2f583acab65d8b053c38cf315d3ad1deb/mockery/generator.go.

var representationMap = map[string]string{
	"string": "String",
	"error":  "Error",
	"int":    "Int",
	"bool":   "Bool",
}

var invalidIdentifierChar = regexp.MustCompile("[^[:digit:][:alpha:]_]")

// Interface represents the interface that should be mocked.
type Interface struct {
	Name      string
	Pkg       *types.Package
	Type      *types.Interface
	NamedType *types.Named
}

// Generator is responsible for generating the string containing
// imports and the mock struct that will later be written out as file.
type Generator struct {
	buf bytes.Buffer

	iface      *Interface
	pkg        string
	structName string

	importsWerePopulated bool
	localizationCache    map[string]string
	packagePathToName    map[string]string
	nameToPackagePath    map[string]string

	packageRoots []string
}

// NewGenerator builds a Generator.
func NewGenerator(iface *Interface, structName string) *Generator {

	var roots []string

	for _, root := range filepath.SplitList(build.Default.GOPATH) {
		roots = append(roots, filepath.Join(root, "src"))
	}

	g := &Generator{
		iface:             iface,
		pkg:               iface.Pkg.Path(),
		structName:        structName,
		localizationCache: make(map[string]string),
		packagePathToName: make(map[string]string),
		nameToPackagePath: make(map[string]string),
		packageRoots:      roots,
	}

	g.addPackageImportWithName("github.com/stretchr/testify/mock", "mock")
	return g
}

func (g *Generator) populateImports() {
	if g.importsWerePopulated {
		return
	}
	for i := 0; i < g.iface.Type.NumMethods(); i++ {
		fn := g.iface.Type.Method(i)
		ftype := fn.Type().(*types.Signature)
		g.addImportsFromTuple(ftype.Params())
		g.addImportsFromTuple(ftype.Results())
		g.renderType(g.iface.NamedType)
	}
}

func (g *Generator) addImportsFromTuple(list *types.Tuple) {
	for i := 0; i < list.Len(); i++ {
		// We use renderType here because we need to recursively
		// resolve any types to make sure that all named types that
		// will appear in the interface file are known
		g.renderType(list.At(i).Type())
	}
}

func (g *Generator) addPackageImport(pkg *types.Package) string {
	return g.addPackageImportWithName(pkg.Path(), pkg.Name())
}

func (g *Generator) addPackageImportWithName(path, name string) string {
	path = g.getLocalizedPath(path)
	if existingName, pathExists := g.packagePathToName[path]; pathExists {
		return existingName
	}

	nonConflictingName := g.getNonConflictingName(path, name)
	g.packagePathToName[path] = nonConflictingName
	g.nameToPackagePath[nonConflictingName] = path
	return nonConflictingName
}

func (g *Generator) getNonConflictingName(path, name string) string {
	if !g.importNameExists(name) {
		return name
	}

	// The path will always contain '/' because it is enforced in getLocalizedPath
	// regardless of OS.
	directories := strings.Split(path, "/")

	cleanedDirectories := make([]string, 0, len(directories))
	for _, directory := range directories {
		cleaned := invalidIdentifierChar.ReplaceAllString(directory, "_")
		cleanedDirectories = append(cleanedDirectories, cleaned)
	}
	numDirectories := len(cleanedDirectories)
	var prospectiveName string
	for i := 1; i <= numDirectories; i++ {
		prospectiveName = strings.Join(cleanedDirectories[numDirectories-i:], "")
		if !g.importNameExists(prospectiveName) {
			return prospectiveName
		}
	}
	// Try adding numbers to the given name
	i := 2
	for {
		prospectiveName = fmt.Sprintf("%v%d", name, i)
		if !g.importNameExists(prospectiveName) {
			return prospectiveName
		}
		i++
	}
}

func (g *Generator) importNameExists(name string) bool {
	_, nameExists := g.nameToPackagePath[name]
	return nameExists
}

func calculateImport(set []string, path string) string {
	for _, root := range set {
		if strings.HasPrefix(path, root) {
			packagePath, err := filepath.Rel(root, path)
			if err == nil {
				return packagePath
			}

			log.Printf("Unable to localize path %v, %v", path, err)
		}
	}
	return path
}

// TODO(@IvanMalison): Is there not a better way to get the actual
// import path of a package?
func (g *Generator) getLocalizedPath(path string) string {
	if strings.HasSuffix(path, ".go") {
		path, _ = filepath.Split(path)
	}
	if localized, ok := g.localizationCache[path]; ok {
		return localized
	}
	directories := strings.Split(path, string(filepath.Separator))
	numDirectories := len(directories)
	vendorIndex := -1
	for i := 1; i <= numDirectories; i++ {
		dir := directories[numDirectories-i]
		if dir == "vendor" {
			vendorIndex = numDirectories - i
			break
		}
	}

	toReturn := path
	if vendorIndex >= 0 {
		toReturn = filepath.Join(directories[vendorIndex+1:]...)
	} else if filepath.IsAbs(path) {
		toReturn = calculateImport(g.packageRoots, path)
	}

	// Enforce '/' slashes for import paths in every OS.
	toReturn = filepath.ToSlash(toReturn)

	g.localizationCache[path] = toReturn
	return toReturn
}

func (g *Generator) mockName() string {
	return g.structName
}

func (g *Generator) sortedImportNames() (importNames []string) {
	for name := range g.nameToPackagePath {
		importNames = append(importNames, name)
	}
	sort.Strings(importNames)
	return
}

func (g *Generator) generateImports() {
	g.printf("import (")
	// Sort by import name so that we get a deterministic order
	for _, name := range g.sortedImportNames() {
		path := g.nameToPackagePath[name]
		parts := strings.Split(path, "/")
		if parts[len(parts)-1] == name {
			g.printf("\t\"%s\"\n", path)
		} else {
			g.printf("\t%s \"%s\"\n", name, path)
		}
	}
	g.printf(")")
}

// GeneratePrologue generates the prologue of the mock.
func (g *Generator) GeneratePrologue(pkg string) {
	g.populateImports()
	g.printf("package %v%s\n\n", pkg, "mock")

	g.generateImports()
	g.printf("\n")
}

func (g *Generator) printf(s string, vals ...interface{}) {
	_, err := fmt.Fprintf(&g.buf, s, vals...)
	if err != nil {
		log.Print(err)
	}
}

type namer interface {
	Name() string
}

// nolint: gocyclo
func (g *Generator) renderType(typ types.Type) string {
	switch t := typ.(type) {
	case *types.Named:
		o := t.Obj()
		if o.Pkg() == nil || o.Pkg().Name() == "main" {
			return o.Name()
		}
		return g.addPackageImport(o.Pkg()) + "." + o.Name()
	case *types.Basic:
		return t.Name()
	case *types.Pointer:
		return "*" + g.renderType(t.Elem())
	case *types.Slice:
		return "[]" + g.renderType(t.Elem())
	case *types.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), g.renderType(t.Elem()))
	case *types.Signature:
		switch t.Results().Len() {
		case 0:
			return fmt.Sprintf(
				"func(%s)",
				g.renderTypeTuple(t.Params()),
			)
		case 1:
			return fmt.Sprintf(
				"func(%s) %s",
				g.renderTypeTuple(t.Params()),
				g.renderType(t.Results().At(0).Type()),
			)
		default:
			return fmt.Sprintf(
				"func(%s)(%s)",
				g.renderTypeTuple(t.Params()),
				g.renderTypeTuple(t.Results()),
			)
		}
	case *types.Map:
		kt := g.renderType(t.Key())
		vt := g.renderType(t.Elem())

		return fmt.Sprintf("map[%s]%s", kt, vt)
	case *types.Chan:
		switch t.Dir() {
		case types.SendRecv:
			return "chan " + g.renderType(t.Elem())
		case types.RecvOnly:
			return "<-chan " + g.renderType(t.Elem())
		default:
			return "chan<- " + g.renderType(t.Elem())
		}
	case *types.Struct:
		var fields []string

		for i := 0; i < t.NumFields(); i++ {
			f := t.Field(i)

			if f.Anonymous() {
				fields = append(fields, g.renderType(f.Type()))
			} else {
				fields = append(fields, fmt.Sprintf("%s %s", f.Name(), g.renderType(f.Type())))
			}
		}

		return fmt.Sprintf("struct{%s}", strings.Join(fields, ";"))
	case *types.Interface:
		if t.NumMethods() != 0 {
			panic("Unable to mock inline interfaces with methods")
		}

		return "interface{}"
	case namer:
		return t.Name()
	default:
		panic(fmt.Sprintf("un-namable type: %#v (%T)", t, t))
	}
}

func (g *Generator) renderTypeTuple(tup *types.Tuple) string {
	var parts []string

	for i := 0; i < tup.Len(); i++ {
		v := tup.At(i)

		parts = append(parts, g.renderType(v.Type()))
	}

	return strings.Join(parts, " , ")
}

func isNillable(typ types.Type) bool {
	switch t := typ.(type) {
	case *types.Pointer, *types.Array, *types.Map, *types.Interface, *types.Signature, *types.Chan, *types.Slice:
		return true
	case *types.Named:
		return isNillable(t.Underlying())
	}
	return false
}

type paramList struct {
	Names    []string
	Types    []string
	Params   []string
	Nilable  []bool
	Variadic bool
}

func (g *Generator) genList(list *types.Tuple, variadic bool) *paramList {
	var params paramList

	if list == nil {
		return &params
	}

	for i := 0; i < list.Len(); i++ {
		v := list.At(i)

		ts := g.renderType(v.Type())

		if variadic && i == list.Len()-1 {
			t := v.Type()
			switch t := t.(type) {
			case *types.Slice:
				params.Variadic = true
				ts = "..." + g.renderType(t.Elem())
			default:
				panic("bad variadic type!")
			}
		}

		pname := v.Name()

		if g.nameCollides(pname) || pname == "" {
			pname = fmt.Sprintf("_a%d", i)
		}

		params.Names = append(params.Names, pname)
		params.Types = append(params.Types, ts)

		params.Params = append(params.Params, fmt.Sprintf("%s %s", pname, ts))
		params.Nilable = append(params.Nilable, isNillable(v.Type()))
	}

	return &params
}

func (g *Generator) nameCollides(pname string) bool {
	if pname == g.pkg {
		return true
	}
	return g.importNameExists(pname)
}

// ErrNotSetup is returned when the generator is not configured.
var ErrNotSetup = errors.New("not setup")

// Generate builds a string that constitutes a valid go source file
// containing the mock of the relevant interface.
// nolint: gocyclo
func (g *Generator) Generate() error {
	g.populateImports()
	if g.iface == nil {
		return ErrNotSetup
	}

	g.printf(
		"// %s is a mock implementation of the %s interface.\n", g.mockName(),
		g.iface.Pkg.Name()+"."+g.iface.Name,
	)

	g.printf(
		"type %s struct {\n\tmock.Mock\n}\n\n", g.mockName(),
	)

	for i := 0; i < g.iface.Type.NumMethods(); i++ {
		fn := g.iface.Type.Method(i)

		ftype := fn.Type().(*types.Signature)
		fname := fn.Name()

		params := g.genList(ftype.Params(), ftype.Variadic())
		returns := g.genList(ftype.Results(), false)

		g.printf("// %s is a mock implementation of %s.%s#%s.\n", fname, g.iface.Pkg.Name(), g.iface.Name, fname)

		g.printf(
			"func (m *%s) %s(%s) ", g.mockName(), fname,
			strings.Join(params.Params, ", "),
		)

		switch len(returns.Types) {
		case 0:
			g.printf("{\n")
		case 1:
			g.printf("%s {\n", returns.Types[0])
		default:
			g.printf("(%s) {\n", strings.Join(returns.Types, ", "))
		}

		var formattedParamNames string
		for i, name := range params.Names {
			if i > 0 {
				formattedParamNames += ", "
			}

			paramType := params.Types[i]
			// for variable args, move the ... to the end.
			if strings.Index(paramType, "...") == 0 {
				name += "..."
			}
			formattedParamNames += name
		}

		called := g.generateCalled(params, formattedParamNames) // m.Called invocation string

		if len(returns.Types) > 0 {
			g.printf("\targs := %s\n\n", called)

			hasNilable := hasNilableBesidesError(returns)
			if !hasNilable {
				r := representation(returns, false)
				g.printf("\treturn %s\n", r)
			} else {
				ifStatements := []string{}
				for i := range returns.Types {
					if returns.Nilable[i] && returns.Types[i] != "error" {
						ifStatements = append(ifStatements, fmt.Sprintf("args.Get(%d) != nil", i))
					}
				}
				g.printf("\tif %s {\n", strings.Join(ifStatements, " && "))
				r := representation(returns, false)
				g.printf("\t\treturn %s\n", r)
				g.printf("\t}\n\n")
				r = representation(returns, true)
				g.printf("\treturn %s\n", r)
			}
		} else {
			g.printf("\t%s\n", called)
		}

		g.printf("}\n")
	}

	return nil
}

// generateCalled returns the Mock.Called invocation string and, if necessary, prints the
// steps to prepare its argument list.
//
// It is separate from Generate to avoid cyclomatic complexity through early return statements.
func (g *Generator) generateCalled(list *paramList, formattedParamNames string) string {
	namesLen := len(list.Names)
	if namesLen == 0 {
		return "m.Called()"
	}

	if !list.Variadic {
		return "m.Called(" + formattedParamNames + ")"
	}

	var variadicArgsName string
	variadicName := list.Names[namesLen-1]
	variadicIface := strings.Contains(list.Types[namesLen-1], "interface{}")

	if variadicIface {
		// Variadic is already of the interface{} type, so we don't need special handling.
		variadicArgsName = variadicName
	} else {
		// Define _va to avoid "cannot use t (type T) as type []interface {} in append" error
		// whenever the variadic type is non-interface{}.
		g.printf("\t_va := make([]interface{}, len(%s))\n", variadicName)
		g.printf("\tfor _i := range %s {\n\t\t_va[_i] = %s[_i]\n\t}\n", variadicName, variadicName)
		variadicArgsName = "_va"
	}

	// _ca will hold all arguments we'll mirror into Called, one argument per distinct value
	// passed to the method.
	//
	// For example, if the second argument is variadic and consists of three values,
	// a total of 4 arguments will be passed to Called. The alternative is to
	// pass a total of 2 arguments where the second is a slice with those 3 values from
	// the variadic argument. But the alternative is less accessible because it requires
	// building a []interface{} before calling Mock methods like On and AssertCalled for
	// the variadic argument, and creates incompatibility issues with the diff algorithm
	// in github.com/stretchr/testify/mock.
	//
	// This mirroring will allow argument lists for methods like On and AssertCalled to
	// always resemble the expected calls they describe and retain compatibility.
	//
	// It's okay for us to use the interface{} type, regardless of the actual types, because
	// Called receives only interface{} anyway.
	g.printf("\tvar _ca []interface{}\n")

	if namesLen > 1 {
		nonVariadicParamNames := formattedParamNames[0:strings.LastIndex(formattedParamNames, ",")]
		g.printf("\t_ca = append(_ca, %s)\n", nonVariadicParamNames)
	}
	g.printf("\t_ca = append(_ca, %s...)\n", variadicArgsName)

	return "m.Called(_ca...)"
}

func (g *Generator) Write(w io.Writer) error {
	opt := &imports.Options{Comments: true}
	theBytes := g.buf.Bytes()

	res, err := imports.Process("mock.go", theBytes, opt)
	if err != nil {
		line := "--------------------------------------------------------------------------------------------"
		_, printErr := fmt.Fprintf(os.Stderr, "Between the lines is the file (mock.go) mockery generated in-memory but detected as invalid:\n%s\n%s\n%s\n", line, g.buf.String(), line)
		if printErr != nil {
			log.Print(printErr)
		}
		return err
	}

	_, err = w.Write(res)
	if err != nil {
		log.Print(err)
	}
	return nil
}

func representation(returns *paramList, withoutNilable bool) string {
	var ret []string
	for idx, typ := range returns.Types {
		var r string
		if representationMap[typ] != "" {
			r = fmt.Sprintf("args.%s(%d)", representationMap[typ], idx)
		} else {
			if withoutNilable {
				r = "nil"
			} else {
				r = fmt.Sprintf("args.Get(%d).(%s)", idx, typ)
			}
		}
		ret = append(ret, r)
	}
	return strings.Join(ret, ", ")
}

func hasNilableBesidesError(returns *paramList) bool {
	for i := range returns.Nilable {
		if returns.Nilable[i] && returns.Types[i] != "error" {
			return true
		}
	}
	return false
}
