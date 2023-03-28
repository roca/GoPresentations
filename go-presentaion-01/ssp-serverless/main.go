package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"ritscm.regeneron.com/ssp/ssp-serverless/projects"
)
func main() {
	projectName := flag.String("n", "serverless-project", "Name of the project")
	stackName := flag.String("s", "", "Name of the stack")
	projectType := flag.String("t", "", "Type of the project {Lambda, StateMachine, APIGateway, Layer}")
	gitProtocol := flag.String("g", "ssh", "Git protocol to use {ssh, https}")
	flag.Parse()

	if *stackName == "" {
		*stackName = fmt.Sprintf("RnpDSSP_%s", *projectName)
	}
	currentDir, _ := os.Getwd()
	var project projects.Project

// START OMIT
	switch strings.ToUpper(*projectType) {
	case "LAMBDA":
		project = projects.NewLambdaProject(currentDir, *projectName, *stackName, *gitProtocol)
	case "STATEMACHINE":
		project = projects.NewStateMachineProject(currentDir, *projectName, *stackName, *gitProtocol)
	case "APIGATEWAY":
		project = projects.NewApiGatewayProject(currentDir, *projectName, *stackName, *gitProtocol)
	case "LAYER":
		project = projects.NewLayer(currentDir, *projectName, *stackName, *gitProtocol)
	default:
		flag.Usage()
		log.Fatal("A valid project type must be specified")

	}
// END OMIT

	// Create the project
	err := project.Create()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

}