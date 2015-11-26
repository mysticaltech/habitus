package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloud66/habitus/build"
	"github.com/cloud66/habitus/configuration"
	"github.com/op/go-logging"

	"github.com/bugsnag/bugsnag-go"
)

var format = logging.MustStringFormatter(
	"%{color}▶ %{message} %{color:reset}",
)

var (
	flagLevel  string
	VERSION    string = "dev"
	BUILD_DATE string = ""
)

func init() {
	bugsnag.Configure(bugsnag.Configuration{
		APIKey:     "ba9d7ae6b333e27971d86e5bf7abe996",
		AppVersion: VERSION,
	})
}

func main() {
	args := os.Args[1:]
	defer bugsnag.AutoNotify()

	var log = logging.MustGetLogger("cxbuilder")
	logging.SetFormatter(format)

	config := configuration.CreateConfig()
	flag.StringVar(&config.Buildfile, "f", "build.yml", "Build file path. Defaults to build.yml in the workdir")
	flag.StringVar(&config.Workdir, "d", "", "work directory for this build. Defaults to the current directory")
	flag.BoolVar(&config.NoCache, "no-cache", false, "Use cache in build")
	flag.BoolVar(&config.SuppressOutput, "suppress", false, "Suppress build output")
	flag.BoolVar(&config.RmTmpContainers, "rm", true, "Remove intermediate containers")
	flag.BoolVar(&config.ForceRmTmpContainer, "force-rm", false, "Force remove intermediate containers")
	flag.StringVar(&config.StartStep, "s", "", "Starting step for the build")
	flag.StringVar(&config.UniqueID, "uid", "", "Unique ID for the build. Used only for multi-tenanted build environments")
	flag.StringVar(&flagLevel, "level", "debug", "Log level: debug, info, notice, warning, error and critical")
	flag.StringVar(&config.DockerHost, "host", os.Getenv("DOCKER_HOST"), "Docker host link. Uses DOCKER_HOST if missing")
	flag.StringVar(&config.DockerCert, "certs", os.Getenv("DOCKER_CERT_PATH"), "Docker cert folder. Uses DOCKER_CERT_PATH if missing")
	flag.Var(&config.EnvVars, "env", "Environment variables to be used during build. If empty cxbuild uses parent process environment variables")
	flag.BoolVar(&config.KeepSteps, "keep-steps", false, "Keep all stpes. Used for debugging each step")
	flag.BoolVar(&config.NoSquash, "no-cleanup", false, "Skip cleanup commands for this run. Used for debugging")
	flag.BoolVar(&config.FroceRmImages, "force-rmi", false, "Force remove of unwanted images")
	flag.BoolVar(&config.NoPruneRmImages, "noprune-rmi", false, "No pruning of unwanted images")

	config.Logger = *log

	flag.Parse()

	if len(args) > 0 && args[0] == "help" {
		fmt.Println("cxbuild - (c) 2015 Cloud 66 Inc.")
		flag.PrintDefaults()
		return
	}

	level, err := logging.LogLevel(flagLevel)
	if err != nil {
		fmt.Println("Invalid log level value. Falling back to debug")
		level = logging.DEBUG
	}
	logging.SetLevel(level, "cxbuilder")

	if config.Workdir == "" {
		if curr, err := os.Getwd(); err != nil {
			log.Fatal("Failed to get the current directory")
			os.Exit(1)
		} else {
			config.Workdir = curr
		}
	}

	if config.Buildfile == "build.yml" {
		config.Buildfile = filepath.Join(config.Workdir, "build.yml")
	}

	c, err := build.LoadBuildFromFile(&config)
	if err != nil {
		log.Fatalf("Failed: %s", err.Error())
	}

	if c.IsPrivileged && os.Getenv("SUDO_USER") == "" {
		log.Fatal("Some of the build steps require admin privileges (sudo). Please run with sudo\nYou might want to use --certs=$DOCKER_CERT_PATH --host=$DOCKER_HOST params to make sure all environment variables are available to the process")
		os.Exit(1)
	}

	b := build.NewBuilder(c, &config)
	err = b.StartBuild(config.StartStep)
	if err != nil {
		log.Error("Error during build %s", err.Error())
	}
}
