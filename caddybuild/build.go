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
	Middleware features.Middleware
	After      string
}

const (
	directivesFile = "config/directives.go"
	CaddyPackage   = "github.com/mholt/caddy"
)

var (
	config          Config
	errParse        = errors.New("Error creating custom build, check settings.")
	errPackageNames = errors.New("Error retrieving package names.")

	// directivePos maps directive name to its position on the Registry.
	directivesPos = map[string]int{}
)

func init() {
	// prevent repetitive loop through Registry list
	for i, m := range features.Registry {
		directivesPos[m.Directive] = i
	}
}

// PrepareBuild prepares a custombuild.Builder for generating a custom binary using
// middlewares. A call to Build on the returned builder will generate the binary.
// If error is not nil, the returned Builder should be ignored.
func PrepareBuild(middlewares features.Middlewares) (custombuild.Builder, error) {
	// create builder
	builder, err := custombuild.NewUnready(CaddyPackage, gen(middlewares), middlewares.Packages())
	if err != nil {
		return builder, err
	}

	// TODO make this configurable and enable only for dev
	builder.UseNetworkForAll(false)

	err = builder.Setup()
	if err != nil {
		// not useful, clear assets
		go builder.Teardown()
		return builder, err
	}

	// necessary to ensure import "github.com/mholt/caddy..." is referencing
	// this code.
	err = builder.SetImportPath(CaddyPackage)
	if err != nil {
		// not useful, clear assets
		go builder.Teardown()
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

		// if no middleware is added, no code generation needed.
		if len(middlewares) == 0 {
			return nil
		}

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
			after := getPrevDirective(m.Directive)
			if after != "" {
				found := false
				c := node.(*ast.ValueSpec).Values[0].(*ast.CompositeLit)
				for _, m := range c.Elts {
					directive := m.(*ast.CompositeLit).Elts[0].(*ast.BasicLit)
					if strconv.Quote(after) == directive.Value {
						end = int(m.End()) + 1
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("Directive '%s' not found.", config.After)
				}
			}

			out = out[:end] + snippet + out[end:]
		}

		return ioutil.WriteFile(file, []byte(out), os.FileMode(0660))
	}
}

// getPrevDirective gets the previous directive to d. It returns
// the directive if found or an empty string otherwise.
func getPrevDirective(d string) string {
	// check if dev mode
	if config.After != "" {
		return config.After
	}
	// use registry order
	pos, ok := directivesPos[d]
	if !ok || pos <= 0 {
		return ""
	}
	return features.Registry[pos-1].Directive
}

// getPackageNames gets the package names of packages. Useful for packages that
// has a name different to their folder name. It returns a map of each package
// to its name.
func getPackageNames(packages []string) (map[string]string, error) {
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
	for i, p := range packages {
		m[p] = names[i]
	}
	return m, nil
}

func SetConfig(c Config) {
	config = c
}
