package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/elo7/harbor/commandline"
	"github.com/elo7/harbor/config"
	"github.com/elo7/harbor/download"
	"github.com/elo7/harbor/execute"
	"github.com/elo7/harbor/execute/docker"
	"os"
)

const VERSION = "0.2.5"

func main() {
	usage := `Harbor, a Docker wrapper

Harbor takes a YAML configuration file with the following structure.

 imagetag: <tag to be used on 'docker build'>
 tags:
   - <YAML array of custom tags to create and push into registry>
 downloadpath: <local root path to download files into>
 s3:
   bucket: <base bucket to download files from>
   basepath: <inside the bucket the root path for files to be downloaded>
   region: <[optional] region of the bucket, default us-east-1>
 files:
   - s3path: <path to file in S3 after [s3.bucket]/[s3.basepath]>
     filename: <local path + name of the file, will be downloaded into [downloadpath]/[localname]>
     permission: <[optional] file permissions, default 0644>
 commands:
   - <YAML array containing shell commands (currently /bin/bash) to be run before 'docker build'>
 buildargs:
   <KEY>:<VALUE pair to be used as --build-arg KEY=VALUE>

 You can use ${<KEY>} as a placeholder in harbor.yml to be replaced by the value passed in a -e flag

 By default, it looks up a file named harbor.yml in the current directory, but you can specify another path.

Usage:
  harbor -h | --help
  harbor --version
  harbor [-e KEY=VALUE]... [options]
  harbor [options]

Options:
  -h, --help                    Show this screen.
  -v, --version                 Show version.
  --config <name>               Path to config file. By default, Harbor looks up for 'harbor.yml' in the current directory, or in the project path (when --project-path is passed).
  --project-path <path>         Project source files path.
  --list-variables              Parses Harbor config file, prints out every ${KEY} found and exits, without building anything.
  -e KEY=VALUE                  Replaces every ${KEY} in harbor.yml with VALUE
  --debug                       Dry-run and print command executions.
  --no-download                 Prevents downloading files from S3.
  --no-commands                 Prevents commands in harbor.yml from being run.
  --no-docker                   Do not run 'docker build', 'docker tag' and 'docker push' commands.
  --no-docker-push              Do not run 'docker push' after building and tagging images.
  --docker-opts=<DOCKER_OPTS>   Will be appended to the base docker command (ex.: 'docker <docker-opts> command').
  --no-latest-tag               Do not build image tagging as 'latest',
                                will use the first tag in 'tags' list from harbor.yml or
                                generate a timestamped tag if no 'tags' list exists.`

	arguments, _ := docopt.Parse(usage, nil, true, "Harbor "+VERSION, true)

	configVars := arguments["-e"].([]string)
	debugFlag := arguments["--debug"].(bool)
	listVariablesFlag := arguments["--list-variables"].(bool)
	noDownloadFlag := arguments["--no-download"].(bool)
	noCommandFlag := arguments["--no-commands"].(bool)
	noDockerFlag := arguments["--no-docker"].(bool)
	noDockerPushFlag := arguments["--no-docker-push"].(bool)
	dockerOpts, _ := arguments["--docker-opts"].(string)
	noLatestTagFlag := arguments["--no-latest-tag"].(bool)

	projectPath := "."
	if arguments["--project-path"] != nil {
		projectPath = arguments["--project-path"].(string)
	}

	configFile := projectPath + "/harbor.yml"
	if arguments["--config"] != nil {
		configFile = arguments["--config"].(string)
	}

	if listVariablesFlag {
		listVariables(configFile)
	}

	cliConfigVars, err := commandline.NewConfigVarsMap(configVars)
	if err != nil {
		checkError(err)
	}

	harborConfig, err := config.Load(cliConfigVars, projectPath, configFile)
	checkError(err)

	config.Options.Debug = debugFlag
	config.Options.DockerOpts = dockerOpts
	config.Options.NoDockerPush = noDockerPushFlag
	config.Options.NoLatestTag = noLatestTagFlag

	if !noDownloadFlag {
		err = download.FromS3(harborConfig)
		checkError(err)
	}

	if !noCommandFlag {
		err = execute.Commands(harborConfig)
		checkError(err)
	}

	if !noDockerFlag {

		// Caso docker não existir ou estiver mal-configurado, falho aqui
		if dockerVersion, err := docker.GetDockerVersion(); err != nil {
			fmt.Printf("There was a problem running the docker version command.\n")
			os.Exit(1)
		} else {
			fmt.Printf("Your Docker client version: %s\n", dockerVersion)
		}

		err = docker.Build(harborConfig)
		checkError(err)
	}
}

func listVariables(configFile string) {
	harborConfigFile, err := config.LoadFile(configFile)
	if err != nil {
		checkError(err)
	}

	variablesFound := config.ReadEnv(harborConfigFile)

	fmt.Printf("--- Found %d variables in %s\n", len(variablesFound), configFile)

	for _, variable := range variablesFound {
		fmt.Printf("---   Found: %s\n", variable)
	}

	os.Exit(0)
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
