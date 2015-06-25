package caddybuild

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/caddyserver/buildsrv/features"
	"github.com/mholt/custombuild"
	"golang.org/x/tools/go/ast/astutil"
)

type Config struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Import      string `json:"import"`
	Repo        string `json:"repository"`
	Directive   string `json:"directive`
	After       string `json:"after"`
}

const (
	directivesFile = "config/directives.go"
)

var (
	config          Config
	errParse        = errors.New("Error creating custom build, check setup setting.")
	errPackageNames = errors.New("Error retrieving package names.")
)

// PrepareBuild prepares a custombuild.Builder for generating a custom binary using
// middlewares. A call to Build on the returned builder will generate the binary.
// If error is not nil, the returned Builder should be ignored.
func PrepareBuild(middlewares features.Middlewares) (custombuild.Builder, error) {
	// create builder
	builder, err := custombuild.NewUnready("github.com/mholt/caddy", gen(middlewares), middlewares.Packages())
	if err != nil {
		return builder, err
	}

	// TODO make this configurable and enable only for dev
	builder.UseNetworkForAll(false)

	err = builder.Setup()
	if err != nil {
		return builder, err
	}

	// necessary to ensure import "github.com/mholt/caddy..." is referencing
	// this code.
	err = builder.SetImportPath("github.com/mholt/caddy")
	if err != nil {
		return builder, err
	}

	return builder, nil
}

// gen is code generation function that insert custom directives at runtime.
func gen(middlewares features.Middlewares) custombuild.CodeGenFunc {
	return func(src string, packages []string) (err error) {
		// prevent possible panic from assertions.
		defer func() {
			if recover() != nil {
				err = errParse
			}
		}()

		fset := token.NewFileSet()
		file := filepath.Join(src, directivesFile)
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			return err
		}
		packageNames, err := getPackageNames(middlewares.Packages())
		if err != nil {
			return err
		}
		for _, m := range middlewares {
			astutil.AddImport(fset, f, m.Package)
		}
		var buf bytes.Buffer
		err = printer.Fprint(&buf, fset, f)
		if err != nil {
			return err
		}

		out := buf.String()

		for _, m := range middlewares {
			f, err = parser.ParseFile(token.NewFileSet(), "", out, 0)
			node, ok := f.Scope.Lookup("directiveOrder").Decl.(ast.Node)
			if !ok {
				return errParse
			}

			snippet := fmt.Sprintf(`{"%s", %s.Setup},`+"\n", m.Directive, packageNames[m.Package])

			// add to end of directives.
			end := int(node.End()) - 2

			// if after is set, locate directive and add after it.
			if config.After != "" {
				found := false
				c := node.(*ast.ValueSpec).Values[0].(*ast.CompositeLit)
				for _, m := range c.Elts {
					directive := m.(*ast.CompositeLit).Elts[0].(*ast.BasicLit)
					if strconv.Quote(config.After) == directive.Value {
						end = int(m.End()) + 1
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("Directive '%s' not found. Ensure 'after' in middleware.json is a valid directive.", config.After)
				}
			}

			out = out[:end] + snippet + out[end:]
		}

		return ioutil.WriteFile(file, []byte(out), os.FileMode(0660))
	}
}

// getPackageNames gets the package names of packages. Useful for packages that
// has a name different to their folder name. It returns a map of each package
// to its name.
func getPackageNames(packages []string) (map[string]string, error) {
	//go list -f '{{.Name}}'
	args := append([]string{"list", "-f", "{{.Name}}"}, packages...)
	output, err := exec.Command("go", args...).Output()
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	names := strings.Fields(string(output))
	if len(names) < len(packages) {
		return nil, errPackageNames
	}
	for i, v := range names {
		m[packages[i]] = v
	}
	return m, nil
}

func SetConfig(c Config) {
	config = c
}

