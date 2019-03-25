// Package export outputs a json file of Lambda function information.
package export

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/tj/cobra"

	"github.com/apex/apex/cmd/apex/root"
)

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
	jsonFunctions, _ := json.MarshalIndent(functionsMap, "", "  ")

	err := os.MkdirAll(".protego", 0700)
	if err != nil {
		fmt.Println("Failed to create .protego dir")
		return err
	}

	err = ioutil.WriteFile(".protego/export.json", jsonFunctions, 0700)
	if err != nil {
		fmt.Println("Failed to create export.json file")
		return err
	}

	fmt.Println("export data to: .protego/export.json")
	return nil
}
