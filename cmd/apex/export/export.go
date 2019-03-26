// Package export outputs a json file of Lambda function information.
package export

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/tj/cobra"

	"github.com/apex/apex/cmd/apex/root"
	"github.com/apex/apex/utils"
	"github.com/apex/log"
)

// env file.
var envFile string

// env supplied.
var env []string

// output file name.
var outputFile string

// example output.
const example = `
    export all functions information to export.json file
    $ apex export`

// Command config.
var Command = &cobra.Command{
	Use:     "export ",
	Short:   "Exports all functions information to export.json file",
	Example: example,
	RunE:    run,
}

// Initialize.
func init() {
	root.Register(Command)

	f := Command.Flags()
	f.StringSliceVarP(&env, "set", "s", nil, "Set environment variable")
	f.StringVarP(&envFile, "env-file", "E", "", "Set environment variables from JSON file")
	f.StringVarP(&outputFile, "output-file", "O", ".protego/export.json", "The name of the output file")

}

// Run command.
func run(c *cobra.Command, args []string) error {
	if err := root.Project.LoadFunctions(args...); err != nil {
		return err
	}

	return exportFunctions()
}

type exportedFunction struct {
	FunctionName     string
	Description      string `json:",omitempty"`
	Runtime          string
	Role             string
	Region           string
	Handler          string
	Timeout          int64
	FunctionArn      string `json:",omitempty"`
	MemorySize       int64
	Environment      map[string]map[string]string `json:",omitempty"`
	VpcConfig        map[string][]string          `json:",omitempty"`
	DeadLetterConfig map[string]string            `json:",omitempty"`
	CodeLocation     string
	// TracingConfig map[string]string // XRay is not supported by apex yet

}

// exportFunctions format.
func exportFunctions() error {

	// read the envFile if supplied and add it to the project env
	if envFile != "" {
		if err := root.Project.LoadEnvFromFile(envFile); err != nil {
			return fmt.Errorf("reading env file %q: %s", envFile, err)
		}
	}

	// read the env if supplied and add it to the project env
	vars, err := utils.ParseEnv(env)
	if err != nil {
		return err
	}

	for k, v := range vars {
		root.Project.Setenv(k, v)
	}

	// For every function create a exportedFunction object and add it to the functionsMap
	functionsMap := make(map[string]exportedFunction)
	for _, fn := range root.Project.Functions {
		awsFn, _ := fn.GetConfigCurrent()

		functionArn := ""
		if awsFn != nil && awsFn.Configuration != nil && awsFn.Configuration.FunctionArn != nil {
			functionArn = *awsFn.Configuration.FunctionArn
		}

		environment := make(map[string]map[string]string)
		if len(fn.Environment) > 0 {
			environment["Variables"] = fn.Environment
		}

		vpcConfig := make(map[string][]string)
		if len(fn.VPC.Subnets) > 0 {
			vpcConfig["SubnetIds"] = fn.VPC.Subnets
			vpcConfig["SecurityGroupIds"] = fn.VPC.SecurityGroups
		}

		deadLetterConfig := make(map[string]string)
		if len(fn.DeadLetterARN) > 0 {
			deadLetterConfig["TargetArn"] = fn.DeadLetterARN
		}

		codeLocation := ".protego/" + fn.Path + "/out.zip"
		if _, err := os.Stat(codeLocation); err == nil {
			fn.Log.Debug("using protego zip file: " + codeLocation)
		} else if os.IsNotExist(err) {
			codeLocation = fn.Path
			fn.Log.Debug("using function path: " + codeLocation)
		} else {
			return err
		}

		f := &exportedFunction{
			FunctionName:     fn.FunctionName,
			Description:      fn.Description,
			FunctionArn:      functionArn,
			Runtime:          fn.Runtime,
			Role:             fn.Role,
			Handler:          fn.Handler,
			Timeout:          fn.Timeout,
			MemorySize:       fn.Memory,
			Region:           fn.Region,
			CodeLocation:     codeLocation,
			Environment:      environment,
			VpcConfig:        vpcConfig,
			DeadLetterConfig: deadLetterConfig}

		functionsMap[fn.Name] = *f

	}
	// convert functionsMap to a json string
	jsonFunctions, _ := json.MarshalIndent(functionsMap, "", "  ")

	// Create the .protego dir
	err = os.MkdirAll(".protego", 0700)
	if err != nil {
		fmt.Println("Failed to create .protego dir")
		return err
	}

	// Write the json file
	err = ioutil.WriteFile(outputFile, jsonFunctions, 0700)
	if err != nil {
		fmt.Println("Failed to create " + outputFile + " file")
		return err
	}

	log.Debugf("%s", jsonFunctions)
	log.Debugf("export data to: " + outputFile)
	return nil
}
