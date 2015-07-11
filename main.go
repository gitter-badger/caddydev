package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/caddyserver/buildsrv/features"
	"github.com/caddyserver/caddydev/caddybuild"
	"github.com/mholt/custombuild"
)

const (
	usage = `Usage: caddydev [options] directive [caddy flags] [go [build flags]]

options:
  -s, --source="."   Source code directory or go get path.
  -a, --after=""     Priority. After which directive should our new directive be placed.
  -u, --update=false Pull latest caddy source code before building.
  -o, --output=""    Path to save custom build. If set, the binary will only be generated, not executed.
                     Set GOOS, GOARCH, GOARM environment variables to generate for other platforms.
  -h, --help=false   Show this usage.

directive:
  directive of the middleware being developed.

caddy flags:
  flags to pass to the resulting custom caddy binary.

go build flags:
  flags to pass to 'go build' while building custom binary prefixed with 'go'.
  go keyword is used to differentiate caddy flags from go build flags.
  e.g. go -race -x -v.
`
)

type cliArgs struct {
	directive string
	after     string
	source    string
	output    string
	update    bool
	caddyArgs []string
	goArgs    []string
}

func main() {
	// parse cli arguments.
	args, err := parseArgs()
	exitIfErr(err)

	// read config file.
	config, err := readConfig(args)
	exitIfErr(err)

	fetched := false
	// if caddy source does not exits, pull source.
	if !isLocalPkg(caddybuild.CaddyPackage) {
		fmt.Print("Caddy source not found. Fetching...")
		err := fetchCaddy()
		exitIfErr(err)
		fmt.Println(" done.")
		fetched = true
	}

	// if update flag is set, pull source.
	if args.update && !fetched {
		fmt.Print("Updating caddy source...")
		err := fetchCaddy()
		exitIfErr(err)
		fmt.Println(" done.")
	}

	caddybuild.SetConfig(config)

	var builder custombuild.Builder
	var f *os.File
	var caddyProcess *os.Process
	// remove temp files.
	addCleanUpFunc(func() {
		if caddyProcess != nil {
			caddyProcess.Kill()
		}
		builder.Teardown()
		if f != nil {
			os.Remove(f.Name())
		}
	})

	builder, err = caddybuild.PrepareBuild(features.Middlewares{config.Middleware})
	exitIfErr(err)

	// if output is set, generate binary only.
	if args.output != "" {
		err := saveCaddy(builder, args.output, args.goArgs)
		exitIfErr(err)
		return
	}

	// create temporary file for binary
	f, err = ioutil.TempFile("", "caddydev")
	exitIfErr(err)
	f.Close()

	// perform custom build
	err = builder.Build("", "", f.Name(), args.goArgs...)
	exitIfErr(err)

	fmt.Println("Starting caddy...")

	// trap os interrupts to ensure cleaning up temp files.
	done := trapInterrupts()

	// start custom caddy.
	go func() {
		cmd, err := startCaddy(f.Name(), args.caddyArgs)
		exitIfErr(err)
		caddyProcess = cmd.Process
		err = cmd.Wait()
		cleanUp()
		exitIfErr(err)
		done <- struct{}{}
	}()

	// wait for exit signal
	<-done

}

// parseArgs parses cli arguments. This caters for parsing extra flags to caddy.
func parseArgs() (cliArgs, error) {
	args := cliArgs{source: "."}

	fs := flag.FlagSet{}
	fs.SetOutput(ioutil.Discard)
	h := false

	fs.StringVar(&args.after, "a", args.after, "")
	fs.StringVar(&args.after, "after", args.after, "")
	fs.StringVar(&args.source, "s", args.source, "")
	fs.StringVar(&args.source, "source", args.source, "")
	fs.StringVar(&args.output, "o", args.output, "")
	fs.StringVar(&args.output, "output", args.output, "")
	fs.BoolVar(&args.update, "u", args.update, "")
	fs.BoolVar(&args.update, "update", args.update, "")
	fs.BoolVar(&h, "h", h, "")
	fs.BoolVar(&h, "help", h, "")

	err := fs.Parse(os.Args[1:])
	if h || err != nil {
		return args, fmt.Errorf(usage)
	}
	if fs.NArg() < 1 {
		return args, usageError(fmt.Errorf("directive not set."))
	}
	args.directive = fs.Arg(0)
	// extract caddy and go args
	if fs.NArg() > 1 {
		remArgs := fs.Args()[1:]
		for i, arg := range remArgs {
			if arg == "go" {
				if len(remArgs) > i+1 {
					args.goArgs = remArgs[i+1:]
				}
				break
			}
			args.caddyArgs = append(args.caddyArgs, arg)
		}
	}
	return args, err
}

// readConfig reads configs from the cli arguments.
func readConfig(args cliArgs) (caddybuild.Config, error) {
	var config = caddybuild.Config{
		Middleware: features.Middleware{Directive: args.directive},
		After:      args.after,
	}
	if args.source != "" {
		if src := pkgFromDir(args.source); src != "" {
			config.Middleware.Package = src
			return config, nil
		}
	}
	return config, fmt.Errorf("Invalid source directory.")
}

// pkgFromDir extracts package import path from dir. dir can be a path on file system
// or go get path.
func pkgFromDir(dir string) string {
	gopaths := strings.Split(os.Getenv("GOPATH"), string(filepath.ListSeparator))

	// if directory exits, infer import path from dir relative to GOPATH.
	if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
		for _, gopath := range gopaths {
			absgopath, _ := filepath.Abs(gopath)
			gosrc := filepath.Join(absgopath, "src") + string(filepath.Separator)
			absdir, _ := filepath.Abs(dir)
			if strings.HasPrefix(absdir, gosrc) {
				return strings.TrimPrefix(absdir, gosrc)
			}
		}
		// not in GOPATH, create symlink to fake GOPATH.
		if newpath, err := symlinkGOPATH(dir); err == nil {
			return filepath.Base(newpath)
		}
	}

	return ""
}

// startCaddy starts custom caddy.
func startCaddy(file string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(file, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return cmd, err
}

func saveCaddy(builder custombuild.Builder, file string, buildArgs []string) error {
	goos := os.Getenv("GOOS")
	goarch := os.Getenv("GOARCH")
	goarm, _ := strconv.Atoi(os.Getenv("GOARM"))
	if goarch == "arm" {
		return builder.BuildARM(goos, goarm, file, buildArgs...)
	}
	return builder.Build(goos, goarch, file, buildArgs...)
}

func fetchCaddy() error {
	_, err := exec.Command("go", "get", "-u", caddybuild.CaddyPackage).Output()
	return err
}

// isLocalPkg takes a go package name and validate if it exists on the filesystem.
func isLocalPkg(p string) bool {
	gopaths := strings.Split(os.Getenv("GOPATH"), string(filepath.ListSeparator))
	for _, gopath := range gopaths {
		absPath := filepath.Join(gopath, "src", p)
		if _, err := os.Stat(absPath); err == nil {
			return true
		}
	}
	return false
}

// symlinkGOPATH creates a symlink to the source directory in a valid GOPATH.
func symlinkGOPATH(dir string) (string, error) {
	var gopath string
	var err error
	if gopath, err = ioutil.TempDir("", "caddydev"); err != nil {
		return "", err
	}
	src := filepath.Join(gopath, "src")
	if err := os.MkdirAll(src, os.FileMode(0700)); err != nil {
		return "", err
	}
	if dir, err = filepath.Abs(dir); err != nil {
		return "", err
	}
	newpath := filepath.Join(src, "caddyaddon")
	if err = os.Symlink(dir, newpath); err != nil {
		return "", err
	}
	err = os.Setenv("GOPATH", gopath+string(filepath.ListSeparator)+os.Getenv("GOPATH"))
	if err == nil {
		addCleanUpFunc(func() {
			os.Remove(newpath)
		})
	}
	return newpath, err
}

// trapInterrupts traps OS interrupt signals.
func trapInterrupts() chan struct{} {
	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Print("OS Interrupt signal received. Performing cleanup...")
		cleanUp()
		fmt.Println(" done.")
		done <- struct{}{}
	}()
	return done
}

// cleanUpFuncs is list of functions to call before application exits.
var cleanUpFuncs []func()

// addCleanUpFunc adds a function to cleanUpFuncs.
func addCleanUpFunc(f func()) {
	cleanUpFuncs = append(cleanUpFuncs, f)
}

// cleanUp calls all functions in cleanUpFuncs.
func cleanUp() {
	for i := range cleanUpFuncs {
		cleanUpFuncs[i]()
	}
}

func exitIfErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func usageError(err error) error {
	return fmt.Errorf("Error: %v\n\n%v", err, usage)
}
