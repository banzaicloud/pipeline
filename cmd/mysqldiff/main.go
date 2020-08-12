// TODO: (pregnor) TEMPORARY experiment.

// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	// differenceOptionIgnoreColumnOrder ignores differring order of columns in
	// compared tables.
	differenceOptionIgnoreColumnOrder = differenceOptionType("ignore-column-order")

	// DifferenceOptionIgnoreComments ignores differring comments.
	differenceOptionIgnoreComments = differenceOptionType("ignore-comments")

	// DifferenceOptionIgnoreConstraintNames ignores differring constraint names
	// in tables.
	differenceOptionIgnoreConstraintNames = differenceOptionType("ignore-constraint-names")

	// differenceOptionIgnoreConstraintOrder ignores differring constraint order
	// in tables.
	differenceOptionIgnoreConstraintOrder = differenceOptionType("ignore-constraint-order")

	// DifferenceOptionIgnoreKeyNames ignores differring key names in tables.
	differenceOptionIgnoreKeyNames = differenceOptionType("ignore-key-names")

	// differenceOptionIgnoreKeyOrder ignores differring key order in tables.
	differenceOptionIgnoreKeyOrder = differenceOptionType("ignore-key-order")

	// DifferenceOptionIgnoreTableOptions ignores differring table options.
	differenceOptionIgnoreTableOptions = differenceOptionType("ignore-table-options")
)

// differenceOptions collects the available difference options.
var differenceOptions = map[string]differenceOptionType{ // nolint:gochecknoglobals // Currently no better way for enum value validation.
	string(differenceOptionIgnoreColumnOrder):     differenceOptionIgnoreColumnOrder,
	string(differenceOptionIgnoreComments):        differenceOptionIgnoreComments,
	string(differenceOptionIgnoreConstraintNames): differenceOptionIgnoreConstraintNames,
	string(differenceOptionIgnoreConstraintOrder): differenceOptionIgnoreConstraintOrder,
	string(differenceOptionIgnoreKeyNames):        differenceOptionIgnoreKeyNames,
	string(differenceOptionIgnoreKeyOrder):        differenceOptionIgnoreKeyOrder,
	string(differenceOptionIgnoreTableOptions):    differenceOptionIgnoreTableOptions,
}

// configuration defines strongly typed configuration parameters.
type configuration struct {
	DatabaseConfig      *databaseConfiguration
	ComparableDatabases []string
	DifferenceOptions   map[differenceOptionType]bool
}

// newConfigurationFromCLIArguments parses raw CLI arguments into a
// configuration object.
func newConfigurationFromCLIArguments(rawArguments []string) (config *configuration, err error) {
	if len(rawArguments) != 0 &&
		rawArguments[0] == os.Args[0] {
		rawArguments = rawArguments[1:]
	}

	config = &configuration{
		DatabaseConfig: &databaseConfiguration{},
	}
	configFlags := flag.NewFlagSet("cli-arguments", flag.ContinueOnError)
	rawDatabases := ""
	rawOptions := ""
	rawOptionValues := make([]string, 0, len(differenceOptions))
	for rawOptionValue := range differenceOptions {
		rawOptionValues = append(rawOptionValues, rawOptionValue)
	}

	configFlags.StringVar(&rawDatabases, "databases", "", "Comma (,) separated list of names of databases to compare.")
	configFlags.StringVar(&config.DatabaseConfig.Host, "host", "", "Host name of the database server instance to connect to.")
	configFlags.StringVar(&rawOptions, "options", "", fmt.Sprintf("Comma (,) separated list of difference options. Available values: %+v", rawOptionValues))
	configFlags.StringVar(&config.DatabaseConfig.Password, "password", "", "Password of the user to log in with.")
	configFlags.UintVar(&config.DatabaseConfig.Port, "port", 3306, "Port used for the database connection.")
	configFlags.StringVar(&config.DatabaseConfig.User, "user", "", "User to connect to the database with.")

	err = configFlags.Parse(rawArguments)
	if err != nil {
		configFlags.Usage()
		return nil, err
	}

	config.ComparableDatabases = strings.Split(rawDatabases, ",")
	splitOptions := strings.Split(rawOptions, ",")
	config.DifferenceOptions = make(map[differenceOptionType]bool, len(splitOptions))
	for _, option := range splitOptions {
		if value, isExisting := differenceOptions[option]; isExisting {
			config.DifferenceOptions[value] = true
		}
	}

	if len(config.ComparableDatabases) != 2 {
		configFlags.Usage()
		return nil, fmt.Errorf(
			"invalid number of databases specified, expected count: '%+v', actual count: '%+v', raw value: '%+v'",
			2, len(config.ComparableDatabases), rawDatabases)
	} else if config.DatabaseConfig.Host == "" {
		configFlags.Usage()
		return nil, fmt.Errorf("invalid empty host specified")
	} else if config.DatabaseConfig.Port > 65535 {
		configFlags.Usage()
		return nil, fmt.Errorf("invalid port specified, expected maximum port number: '%+v', actual port number: '%+v'",
			65535, config.DatabaseConfig.Port)
	} else if config.DatabaseConfig.User == "" {
		configFlags.Usage()
		return nil, fmt.Errorf("invalid empty user specified")
	}

	return config, nil
}

// databaseConfiguration encapsulates information required to connect
// to a database.
type databaseConfiguration struct {
	Host     string
	Password string
	Port     uint
	User     string
}

type differenceOptionType string

func (option differenceOptionType) String() (text string) {
	return string(option)
}

// compareDatabases generates the difference between the specified
// databases.
func compareDatabases(config *configuration) (difference []byte, err error) {
	dumps := make([][]byte, 0, len(config.ComparableDatabases))
	for _, database := range config.ComparableDatabases {
		dump, err := dumpDatabaseSchema(config.DatabaseConfig, database, config.DifferenceOptions)
		if err != nil {
			return nil, fmt.Errorf("dumping database schema failed, error: '%w'", err)
		}

		dumps = append(dumps, dump)
	}

	// if config.DifferenceOptions[differenceOptionIgnoreColumnOrder] ||
	// 	config.DifferenceOptions[differenceOptionIgnoreConstraintNames] ||
	// 	config.DifferenceOptions[differenceOptionIgnoreConstraintOrder] ||
	// 	config.DifferenceOptions[differenceOptionIgnoreKeyNames] ||
	// 	config.DifferenceOptions[differenceOptionIgnoreKeyOrder] {
	// 	columnRawRegex := `^\s+` + "`" + `([0-9a-zA-Z_]+)` + "`" + ` .+$` // https://regex101.com/r/8Ozlev/2
	// 	columnRegex := regexp.MustCompile(columnRawRegex)
	// 	constraintRawRegex := `^\s+CONSTRAINT ` + "`" + `([0-9a-zA-Z_]+)` + "`" + ` .+$` // https://regex101.com/r/dbxulO/1
	// 	constraintRegex := regexp.MustCompile(constraintRawRegex)
	// 	keyRawRegex := `^\s+([A-Z]+ )?KEY (` + "`" + `([0-9a-zA-Z_]+)` + "`" + ` )?.+$` // https://regex101.com/r/TYsLwy/1
	// 	keyRegex := regexp.MustCompile(keyRawRegex)
	// 	//
	// 	createRawRegex := `(CREATE [^\n]+ \(\n)((\s+.+\n)+)(\);)` // https://regex101.com/r/2BVN8z/2
	// 	createRegex := regexp.MustCompile(createRawRegex)

	// }

	filePaths := make([]string, 0, len(config.ComparableDatabases))
	for databaseIndex, database := range config.ComparableDatabases {
		filePath := path.Join(database, ".sql")
		filePaths = append(filePaths, filePath)

		err = ioutil.WriteFile(filePath, dumps[databaseIndex], 0o777)
		if err != nil {
			return nil, fmt.Errorf("writing dump file failed, database: '%+v', error: '%w'", database, err)
		}

		defer os.Remove(filePath)
	}

	gitDiffArguments := make([]string, 0, 4+len(filePaths))
	gitDiffArguments = append(gitDiffArguments, "--no-pager")
	gitDiffArguments = append(gitDiffArguments, "diff")
	gitDiffArguments = append(gitDiffArguments, "--no-index")
	gitDiffArguments = append(gitDiffArguments, "--")
	gitDiffArguments = append(gitDiffArguments, filePaths...)

	gitDiffOutput, gitDiffErrorOutput, err := executeCommand("git", gitDiffArguments...)
	if err != nil &&
		len(gitDiffOutput) == 0 {
		return nil, fmt.Errorf(
			"generating difference failed, error: '%+v', error output: '%+v'", err, gitDiffErrorOutput)
	} else if err != nil &&
		len(gitDiffErrorOutput) != 0 {
		diffArguments := make([]string, 0, 2+len(filePaths))
		diffArguments = append(diffArguments, "--side-by-side")
		diffArguments = append(diffArguments, "width=300")
		diffArguments = append(diffArguments, filePaths...)
		diffOutput, _, _ := executeCommand("diff", diffArguments...)

		return append(diffOutput, append([]byte("\n\n\n"), gitDiffOutput...)...), nil // Note: difference found.
	}

	return nil, nil // Note: no difference, no error.
}

func dumpDatabaseSchema(
	config *databaseConfiguration,
	database string,
	options map[differenceOptionType]bool,
) (dump []byte, err error) {
	arguments := []string{
		"--host", config.Host,
		fmt.Sprintf("--password='%s'", config.Password),
		"--port", fmt.Sprintf("%d", config.Port),
		"--user", config.User,
		"--no-data",
	}

	if options[differenceOptionIgnoreComments] {
		arguments = append(arguments, "--skip-comments")
	}

	if options[differenceOptionIgnoreTableOptions] {
		arguments = append(arguments, "--skip-create-options")
	}

	arguments = append(arguments, database)

	output, _, err := executeCommand("mysqldump", arguments...)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// executeCommand executes a command.
func executeCommand(command string, arguments ...string) (output, errorOutput []byte, err error) {
	stderr := bytes.NewBuffer(nil)
	stdout := bytes.NewBuffer(nil)

	executableCommand := exec.Command(command, arguments...)
	executableCommand.Stderr = stderr
	executableCommand.Stdout = stdout

	err = executableCommand.Run()
	errorOutput, _ = ioutil.ReadAll(stderr)
	output, _ = ioutil.ReadAll(stdout)

	if err != nil {
		return nil, nil, fmt.Errorf(
			"dumping database failed, error: '%w', command: '%s %+v', stdout: '%+v', stderr: '%+v'",
			err, executableCommand.Path, executableCommand.Args, output, errorOutput)
	}

	return output, errorOutput, nil
}

// handleFatalCondition gracefully aborts the program in case the specified
// condition evaluates to true.
func handleFatalCondition(condition bool, exitCode int, messages ...interface{}) {
	if condition {
		log.Println(messages...)
		os.Exit(exitCode)
	}
}

func main() {
	config, err := newConfigurationFromCLIArguments(os.Args)
	handleFatalCondition(err != nil, 1, err)

	difference, err := compareDatabases(config)
	handleFatalCondition(err != nil, 2)

	fmt.Printf("%s\n", difference)

	if len(difference) != 0 {
		os.Exit(3)
	}
}
