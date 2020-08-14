/*
Copyright © 2020 The k3d Author(s)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rancher/k3d/v3/cmd/cluster"
	cfg "github.com/rancher/k3d/v3/cmd/config"
	"github.com/rancher/k3d/v3/cmd/image"
	"github.com/rancher/k3d/v3/cmd/kubeconfig"
	"github.com/rancher/k3d/v3/cmd/node"
	"github.com/rancher/k3d/v3/pkg/config"
	"github.com/rancher/k3d/v3/pkg/runtimes"
	"github.com/rancher/k3d/v3/version"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
)

// RootFlags describes a struct that holds flags that can be set on root level of the command
type RootFlags struct {
	debugLogging bool
	version      bool
}

var flags = RootFlags{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k3d",
	Short: "https://k3d.io/ -> Run k3s in Docker!",
	Long: `https://k3d.io/
k3d is a wrapper CLI that helps you to easily create k3s clusters inside docker.
Nodes of a k3d cluster are docker containers running a k3s image.
All Nodes of a k3d cluster are part of the same docker network.`,
	Run: func(cmd *cobra.Command, args []string) {
		if flags.version {
			printVersion()
		} else {
			if err := cmd.Usage(); err != nil {
				log.Fatalln(err)
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}

func init() {
	cobra.OnInitialize(initLogging, initRuntime, config.InitConfig)

	// add persistent flags (present to all subcommands)
	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "", "Use a config file")
	rootCmd.PersistentFlags().BoolVar(&flags.debugLogging, "verbose", false, "Enable verbose output (debug logging)")

	// add local flags
	rootCmd.Flags().BoolVar(&flags.version, "version", false, "Show k3d and default k3s version")

	// add subcommands
	rootCmd.AddCommand(NewCmdCompletion())
	rootCmd.AddCommand(cluster.NewCmdCluster())
	rootCmd.AddCommand(kubeconfig.NewCmdKubeconfig())
	rootCmd.AddCommand(node.NewCmdNode())
	rootCmd.AddCommand(image.NewCmdImage())
	rootCmd.AddCommand(cfg.NewCmdConfig())

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show k3d and default k3s version",
		Long:  "Show k3d and default k3s version",
		Run: func(cmd *cobra.Command, args []string) {
			printVersion()
		},
	})
}

// initLogging initializes the logger
func initLogging() {
	if flags.debugLogging {
		log.SetLevel(log.DebugLevel)
	} else {
		switch logLevel := strings.ToUpper(os.Getenv("LOG_LEVEL")); logLevel {
		case "DEBUG":
			log.SetLevel(log.DebugLevel)
		case "WARN":
			log.SetLevel(log.WarnLevel)
		case "ERROR":
			log.SetLevel(log.ErrorLevel)
		default:
			log.SetLevel(log.InfoLevel)
		}
	}
	log.SetOutput(ioutil.Discard)
	log.AddHook(&writer.Hook{
		Writer: os.Stderr,
		LogLevels: []log.Level{
			log.PanicLevel,
			log.FatalLevel,
			log.ErrorLevel,
			log.WarnLevel,
		},
	})
	log.AddHook(&writer.Hook{
		Writer: os.Stdout,
		LogLevels: []log.Level{
			log.InfoLevel,
			log.DebugLevel,
		},
	})
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
}

func initRuntime() {
	runtime, err := runtimes.GetRuntime("docker")
	if err != nil {
		log.Fatalln(err)
	}
	runtimes.SelectedRuntime = runtime
	log.Debugf("Selected runtime is '%T'", runtimes.SelectedRuntime)
}

func printVersion() {
	fmt.Printf("k3d version %s\n", version.GetVersion())
	fmt.Printf("k3s version %s (default)\n", version.K3sVersion)
}

// Completion
var completionFunctions = map[string]func(io.Writer) error{
	"bash":       rootCmd.GenBashCompletion,
	"zsh":        rootCmd.GenZshCompletion,
	"psh":        rootCmd.GenPowerShellCompletion,
	"powershell": rootCmd.GenPowerShellCompletion,
}

// NewCmdCompletion creates a new completion command
func NewCmdCompletion() *cobra.Command {
	// create new cobra command
	cmd := &cobra.Command{
		Use:   "completion SHELL",
		Short: "Generate completion scripts for [bash, zsh, powershell | psh]",
		Long:  `Generate completion scripts for [bash, zsh, powershell | psh]`,
		Args:  cobra.ExactArgs(1), // TODO: NewCmdCompletion: add support for 0 args = auto detection
		Run: func(cmd *cobra.Command, args []string) {
			if f, ok := completionFunctions[args[0]]; ok {
				if err := f(os.Stdout); err != nil {
					log.Fatalf("Failed to generate completion script for shell '%s'", args[0])
				}
				return
			}
			log.Fatalf("Shell '%s' not supported for completion", args[0])
		},
	}
	return cmd
}
