package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TODO dont use a global var, add this to the ctx
// FileLogLocation to which we write all cmd stdout, stderr
var FileLogLocation = "/dev/null"

// Command interface execute a cli command and
// returns the stdout, stderr and any error msgs
type Command interface {
	Execute() ([]byte, []byte, error)
	Exists() bool
}

// The CommandLine contains the env var, command name and args to be run
type CommandLine struct {
	EnvVars     map[string]string
	CommandName string
	Args        []string
	Ctx         *context.Context
}

// NewCommandLine constructs a new CommandLine instance
func NewCommandLine(envs map[string]string, cmd string, args []string, ctx *context.Context) *CommandLine {
	return &CommandLine{
		EnvVars:     envs,
		CommandName: cmd,
		Args:        args,
		Ctx:         ctx,
	}
}

// The CommandSession contains the CommandLine
type CommandSession struct {
	CommandLine *CommandLine
	Ctx         context.Context
}

// Execute runs the cli command
func (c *CommandSession) Execute() ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer
	logfile := FileLogLocation
	os.MkdirAll(filepath.Dir(logfile), 0644)
	filehandle, _ := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer filehandle.Close()

	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.CommandLine.CommandName, c.CommandLine.Args...)

	cmd.Stdout = io.MultiWriter(&stdout, filehandle)
	cmd.Stderr = io.MultiWriter(&stderr, filehandle)

	if c.CommandLine.EnvVars != nil {
		additionalEnv := createEnvVars(c.CommandLine.EnvVars)
		newEnv := append(os.Environ(), additionalEnv...)
		cmd.Env = newEnv
	}
	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("Command timed out: %v %v", c.CommandLine.CommandName, strings.Join(c.CommandLine.Args, " "))
	}
	if err != nil {
		return stdout.Bytes(), stderr.Bytes(), err
	}

	return stdout.Bytes(), stderr.Bytes(), err
}

// Exists checks if a command is in the $PATH var
func (c *CommandSession) Exists() bool {
	var err error

	_, err = exec.LookPath(c.CommandLine.CommandName)
	if err != nil {
		return false
	}

	return true
}

// Program is the only method executed for the CommandLine
func (c *CommandLine) Program() Command {
	return &CommandSession{
		CommandLine: c,
	}
}

func createEnvVars(m map[string]string) []string {
	var envVars []string
	for index, elem := range m {
		envVars = append(envVars, fmt.Sprintf("%v=%v", index, elem))
	}
	return envVars
}

// Retry command for specifc time or until successful condition
func Retry(c *CommandLine, timeout time.Duration, grepString string, grepNum int, event chan string) bool {
	var ok bool
	var count, counter, errCounter int
	tout := time.After(timeout)
	retryInterval := 3 * time.Second
	event <- fmt.Sprintf("checking for %v instances of '%v' from command: %v %v", grepNum, grepString, c.CommandName, strings.Join(c.Args, " "))
	FileLogLocationOriginal := FileLogLocation
	FileLogLocation = "/dev/null"
	for {
		select {
		case <-tout:
			ok = false
			break
		default:
			stdout, stderr, err := c.Program().Execute()
			if err != nil || string(stderr) != "" {
				if errCounter == 10 {
					event <- fmt.Sprintf("err: %v, stderr: %v", err, string(stderr))
					ok = false
					break
				}
				errCounter++
			}
			count = strings.Count(string(stdout), grepString)
			if count == grepNum {
				event <- fmt.Sprintf("found %v/%v instances of '%v' from command: %v %v", count, grepNum, grepString, c.CommandName, strings.Join(c.Args, " "))
				ok = true
				break
			} else if count > counter {
				event <- fmt.Sprintf("found %v/%v instances of '%v' from command: %v %v", count, grepNum, grepString, c.CommandName, strings.Join(c.Args, " "))
				counter++
			}
			time.Sleep(retryInterval)
		}
		if count == grepNum || errCounter == 10 {
			break
		}
	}
	FileLogLocation = FileLogLocationOriginal
	return ok
}

// GenericExecute runs a command and only reports back error message
func GenericExecute(envs map[string]string, name string, args []string, ctx *context.Context) error {
	var err error

	c := NewCommandLine(envs, name, args, ctx)

	if !c.Program().Exists() {
		return fmt.Errorf("exec: '%v': executable file not found in $PATH", name)
	}

	_, stderr, err := c.Program().Execute()
	/*
		if err != nil || string(stderr) != "" {
			return fmt.Errorf("err: %v, stderr: %v", err, string(stderr))
		}
	*/

	if name != "kind" {
		if string(stderr) != "" {
			err = fmt.Errorf("err: %v, stderr: %v, cmd: %v %v", err, string(stderr), name, strings.Join(args, " "))
		}
	}

	if err != nil {
		return fmt.Errorf("err: %v, stderr: %v, cmd: %v %v", err, string(stderr), name, strings.Join(args, " "))
	}

	return err
}

// ProvisionerCommands head of command list
type ProvisionerCommands struct {
	Name string
	head *ExternalCommand
}

// ExternalCommand is a single external command
type ExternalCommand struct {
	Name    string
	Actions *CommandLine
	next    *ExternalCommand
}

// CreateCommandList constructor
func CreateCommandList(name string) *ProvisionerCommands {
	return &ProvisionerCommands{
		Name: name,
	}
}

// AddCommand adds a command to the list
func (c *ProvisionerCommands) AddCommand(name string, actions *CommandLine) error {
	s := &ExternalCommand{
		Name:    name,
		Actions: actions,
	}
	if c.head == nil {
		c.head = s
	} else {
		currentNode := c.head
		for currentNode.next != nil {
			currentNode = currentNode.next
		}
		currentNode.next = s
	}
	return nil
}

// GetAll returns all commands in the linked list
func (c *ProvisionerCommands) GetAll() []string {
	var all []string
	currentNode := c.head
	if currentNode == nil {
		return all
	}
	all = append(all, currentNode.Name)
	for currentNode.next != nil {
		currentNode = currentNode.next
		all = append(all, currentNode.Name)
	}

	return all
}

// NotInPath returns commands in the linked list that dont exist in the $PATH
func (c *ProvisionerCommands) NotInPath() []string {
	var result []string
	currentNode := c.head
	if currentNode == nil {
		return result
	}
	if exists := currentNode.Actions.Program().Exists(); exists != true {
		result = append(result, currentNode.Name)
	}
	for currentNode.next != nil {
		currentNode = currentNode.next
		if exists := currentNode.Actions.Program().Exists(); exists != true {
			result = append(result, currentNode.Name)
		}
	}

	return result
}

// Remove removes a node
func (c *ProvisionerCommands) Remove(t string) error {
	var previous *ExternalCommand
	node := c.head
	if node != nil {
		if node.Name == t {
			c.head = node.next
			return nil
		}
	}
	for {
		if node == nil {
			return fmt.Errorf("Not found")
		}

		if node.Name == t {
			break
		}
		previous = node
		node = node.next

	}
	previous.next = node.next

	return nil
}
