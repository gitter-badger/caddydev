package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/caddyserver/buildsrv/features"
	"github.com/caddyserver/caddydev/caddybuild"
	"github.com/mholt/custombuild"
)

const (
	configFile = "middleware.json"
	usage      = `Usage:
	caddydev [-c|-h|help] [caddy flags]

	-c=middleware.json - Path to config file.
	-h=false - show this usage.
	help - alias for -h=true
	caddy flags - flags to pass to caddy.
`
)

func main() {
	// parse cli arguments.
	args, err := parseArgs()
	exitIfErr(err)

	// read config file.
	config, err := readConfig(args.configFile)
	exitIfErr(err)
	caddybuild.SetConfig(config)

	var builder custombuild.Builder
	var f *os.File
	// remove temp files.
	var cleanup = func() {
		if f != nil {
			builder.Teardown()
			os.Remove(f.Name())
		}
	}

	middleware := features.Middleware{
		Directive: config.Directive,
		Package:   config.Import,
	}
	builder, err = caddybuild.PrepareBuild(features.Middlewares{middleware})
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

// trapInterrupts traps OS interrupt signals.
func trapInterrupts(cleanup func()) chan struct{} {
	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Print("OS Interrupt signal received. Performing cleanup...")
		// TODO find how to buy more CPU time and run synchronously
		go cleanup()
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

// parseArgs parses cli arguments. This caters for parsing extra flags to caddy.
func parseArgs() (Args, error) {
	args := Args{configFile: configFile}
	if len(os.Args) == 1 {
		return args, nil
	}

	if os.Args[1] == "-h" || os.Args[1] == "help" {
		return args, fmt.Errorf(usage)
	}

	if !strings.HasPrefix(os.Args[1], "-c") {
		args.caddyArgs = os.Args[1:]
		return args, nil
	}
	// for -c=middleware.json
	if c := strings.Split(os.Args[1], "="); len(c) > 1 {
		args.configFile = c[1]
		if len(os.Args) > 2 {
			args.caddyArgs = os.Args[2:]
		}
		return args, nil
	}
	if len(os.Args) < 3 {
		return args, fmt.Errorf("config file path missing after using -c flag")
	}
	args.configFile = os.Args[2]
	if len(os.Args) > 3 {
		args.caddyArgs = os.Args[3:]
	}
	return args, nil
}

// readConfig reads the middleware.json config file.
func readConfig(file string) (caddybuild.Config, error) {
	var config caddybuild.Config
	f, err := os.Open(file)
	if err != nil {
		return config, err
	}
	err = json.NewDecoder(f).Decode(&config)
	return config, err
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

type Args struct {
	configFile string
	caddyArgs  []string
}
