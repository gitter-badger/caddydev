package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/caddyserver/buildsrv/features"
	"github.com/caddyserver/caddydev/caddybuild"
	"github.com/mholt/custombuild"
)

const (
	usage = `Usage: caddydev [options] directive [caddy flags]

options:
  -s, -source="."   Source code directory or go get path.
  -a, -after=""     Priority. After which directive should our new directive be placed.
  -h, -help=false   Show this usage.

directive:
  directive of the middleware being developed.

caddy flags:
  flags to pass to the resulting custom caddy binary.
`
)

type Args struct {
	directive string
	after     string
	source    string
	caddyArgs []string
}

func usageError(err error) error {
	return fmt.Errorf("Error: %v\n\n%v", err, usage)
}

func main() {
	// parse cli arguments.
	args, err := parseArgs()
	exitIfErr(err)

	// read config file.
	config, err := readConfig(args)
	exitIfErr(err)
	caddybuild.SetConfig(config)

	var builder custombuild.Builder
	var f *os.File
	// remove temp files.
	var cleanup = func() {
		builder.Teardown()
		if f != nil {
			os.Remove(f.Name())
		}
	}

	builder, err = caddybuild.PrepareBuild(features.Middlewares{config.Middleware})
	exitIfErr(err)

	// create temp file for custom binary.
	f, err = ioutil.TempFile("", "caddydev")
	exitIfErr(err)
	f.Close()

	// perform custom build
	err = builder.Build("", "", f.Name())
	exitIfErr(err)

	fmt.Println("Starting caddy...")

	// trap os interrupts to ensure cleaning up temp files.
	done := trapInterrupts(cleanup)

	// start custom caddy.
	go func() {
		err = startCaddy(f.Name(), args.caddyArgs)
		cleanup()
		exitIfErr(err)
	}()

	// wait for exit signal
	<-done

}

// parseArgs parses cli arguments. This caters for parsing extra flags to caddy.
func parseArgs() (Args, error) {
	args := Args{source: "."}

	fs := flag.FlagSet{}
	fs.SetOutput(ioutil.Discard)
	h := false

	fs.StringVar(&args.after, "a", args.after, "")
	fs.StringVar(&args.after, "after", args.after, "")
	fs.StringVar(&args.source, "s", args.source, "")
	fs.StringVar(&args.source, "source", args.source, "")
	fs.BoolVar(&h, "h", false, "")
	fs.BoolVar(&h, "help", h, "")

	err := fs.Parse(os.Args[1:])
	if h || err != nil {
		return args, fmt.Errorf(usage)
	}

	if fs.NArg() < 1 {
		return args, usageError(fmt.Errorf("directive not set."))
	}

	args.directive = fs.Arg(0)

	if fs.NArg() > 1 {
		args.caddyArgs = fs.Args()[1:]
	}

	return args, err
}

// readConfig reads the middleware.json config file.
func readConfig(args Args) (caddybuild.Config, error) {
	var config = caddybuild.Config{
		Middleware: features.Middleware{args.directive, ""},
	}
	if args.source == "" {
		return config, fmt.Errorf("Invalid source")
	}
	if src := pkgFromDir(args.source); src != "" {
		config.Middleware.Package = src
		return config, nil
	}
	return config, fmt.Errorf("Invalid source")
}

func pkgFromDir(dir string) string {
	gopaths := strings.Split(os.Getenv("GOPATH"), string(filepath.ListSeparator))

	// if directory exits, infer package name relative to GOPATH
	if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
		for _, gopath := range gopaths {
			absgopath, _ := filepath.Abs(gopath)
			gosrc := filepath.Join(absgopath, "src") + "/"
			absdir, _ := filepath.Abs(dir)
			if strings.HasPrefix(absdir, gosrc) {
				return strings.TrimPrefix(absdir, gosrc)
			}
		}
	}

	// check if valid package
	for _, gopath := range gopaths {
		absPath := filepath.Join(gopath, "src", dir)
		if _, err := os.Stat(absPath); err == nil {
			return dir
		}
	}
	return ""
}

// startCaddy starts custom caddy and blocks until process stops.
func startCaddy(file string, args []string) error {
	cmd := exec.Command(file, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

// trapInterrupts traps OS interrupt signals.
func trapInterrupts(cleanup func()) chan struct{} {
	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Print("OS Interrupt signal received. Performing cleanup...")
		cleanup()
		fmt.Println(" done.")
		done <- struct{}{}
	}()
	return done
}

func exitIfErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
