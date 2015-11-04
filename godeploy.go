package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dynport/gossh"
)

// Config contains the configuration file structure from a JSON file
type Config struct {
	Host            string
	User            string
	DeployDirectory string
	App             string
	Repo            string
	Commands        []string
}

// MakeLogger just adds a prefix (DEBUG, INFO, ERROR)
// returns a function of type gossh.Writer func(...interface{})
func MakeLogger(prefix string) gossh.Writer {
	return func(args ...interface{}) {
		log.Println((append([]interface{}{prefix}, args...))...)
	}
}

// ReadConfig reads from the passed in configuration filename
func ReadConfig(filename string) Config {
	file, _ := os.Open(filename)
	decoder := json.NewDecoder(file)
	configuration := Config{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error:", err)
	}
	return configuration
}

// PerformDeploy performs all of the deployment steps
func PerformDeploy(host string, app string, branch string) {
	config := ReadConfig(fmt.Sprintf("%s.json", app))

	client := gossh.New(host, config.User)

	client.DebugWriter = MakeLogger("DEBUG")
	client.InfoWriter = MakeLogger("INFO ")
	client.ErrorWriter = MakeLogger("ERROR")

	defer client.Close()

	SetupWorkDirectory(config, client)
	CheckoutRepo(branch, config, client)
	RunCommandChain(config.Commands, client)
	CleanupRelease(config, client)
}

// RunCommandChain runs a set of commands on the client
func RunCommandChain(commands []string, client *gossh.Client) {
	for _, cmd := range commands {
		RunCommand(cmd, client)
	}
}

// CleanupRelease moves the build directory to a timestamped directory
func CleanupRelease(config Config, client *gossh.Client) {
	timestamp := time.Now().Format("20060102150405")
	from := fmt.Sprintf("%s/releases/build", config.DeployDirectory)
	to := fmt.Sprintf("%s/releases/%s", config.DeployDirectory, timestamp)

	RunCommand(fmt.Sprintf("mv %s %s", from, to), client)
	RunCommand(fmt.Sprintf("ln -sf %s %s/current", to, config.DeployDirectory), client)
}

// SetupWorkDirectory makes sure the build directory exists
func SetupWorkDirectory(config Config, client *gossh.Client) {
	RunCommand(fmt.Sprintf("mkdir %s", config.DeployDirectory), client)
	RunCommand(fmt.Sprintf("mkdir %s/releases", config.DeployDirectory), client)
	RunCommand(fmt.Sprintf("mkdir %s/shared", config.DeployDirectory), client)
	RunCommand(fmt.Sprintf("mkdir %s/releases/build", config.DeployDirectory), client)
}

// CheckoutRepo makes sure the application has been checked out from the repo
func CheckoutRepo(branch string, config Config, client *gossh.Client) {
	directory := fmt.Sprintf("%s/releases/build", config.DeployDirectory)
	RunCommand(fmt.Sprintf("git clone %s %s", config.Repo, directory), client)
	RunCommand(fmt.Sprintf("cd %s && git checkout %s", directory, branch), client)
}

// RunCommand runs one specific command on the client
func RunCommand(cmd string, client *gossh.Client) {
	fmt.Println("Running ", cmd)
	rsp, e := client.Execute(cmd)
	if e != nil {
		client.ErrorWriter(e.Error())
		client.ErrorWriter("STDOUT: " + rsp.Stdout())
		client.ErrorWriter("STDERR: " + rsp.Stderr())
	}
	client.InfoWriter(rsp.String())
}

func main() {
	fmt.Println("Enter the application to deploy: ")
	var application string
	fmt.Scanln(&application)

	fmt.Println("Enter the server(s) (multiple servers must be comma-separated) to deploy to: ")
	var server string
	fmt.Scanln(&server)

	fmt.Println("Enter the branch/tag/version number to deploy: ")
	var branch string
	fmt.Scanln(&branch)

	fmt.Println("Preparing to deploy", branch, "of", application, "to", server, "...")

	PerformDeploy(server, application, branch)
}
