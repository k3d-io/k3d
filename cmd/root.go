/*
Copyright © 2020-2021 The k3d Author(s)

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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/rancher/k3d/v4/cmd/cluster"
	cfg "github.com/rancher/k3d/v4/cmd/config"
	"github.com/rancher/k3d/v4/cmd/debug"
	"github.com/rancher/k3d/v4/cmd/image"
	"github.com/rancher/k3d/v4/cmd/kubeconfig"
	"github.com/rancher/k3d/v4/cmd/node"
	"github.com/rancher/k3d/v4/cmd/plugin"
	"github.com/rancher/k3d/v4/cmd/registry"
	cliutil "github.com/rancher/k3d/v4/cmd/util"
	l "github.com/rancher/k3d/v4/pkg/logger"
	"github.com/rancher/k3d/v4/pkg/runtimes"
	"github.com/rancher/k3d/v4/version"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
)

// RootFlags describes a struct that holds flags that can be set on root level of the command
type RootFlags struct {
	debugLogging       bool
	traceLogging       bool
	timestampedLogging bool
	version            bool
}

var flags = RootFlags{}

func NewCmdK3d() *cobra.Command {

	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
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
					l.Log().Fatalln(err)
				}
			}
		},
	}

	rootCmd.PersistentFlags().BoolVar(&flags.debugLogging, "verbose", false, "Enable verbose output (debug logging)")
	rootCmd.PersistentFlags().BoolVar(&flags.traceLogging, "trace", false, "Enable super verbose output (trace logging)")
	rootCmd.PersistentFlags().BoolVar(&flags.timestampedLogging, "timestamps", false, "Enable Log timestamps")

	// add local flags
	rootCmd.Flags().BoolVar(&flags.version, "version", false, "Show k3d and default k3s version")

	// add subcommands
	rootCmd.AddCommand(NewCmdCompletion(rootCmd))
	rootCmd.AddCommand(cluster.NewCmdCluster())
	rootCmd.AddCommand(kubeconfig.NewCmdKubeconfig())
	rootCmd.AddCommand(node.NewCmdNode())
	rootCmd.AddCommand(image.NewCmdImage())
	rootCmd.AddCommand(cfg.NewCmdConfig())
	rootCmd.AddCommand(registry.NewCmdRegistry())
	rootCmd.AddCommand(debug.NewCmdDebug())
	rootCmd.AddCommand(plugin.NewCmdPlugin())

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show k3d and default k3s version",
		Long:  "Show k3d and default k3s version",
		Run: func(cmd *cobra.Command, args []string) {
			printVersion()
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "runtime-info",
		Short: "Show runtime information",
		Long:  "Show some information about the runtime environment (e.g. docker info)",
		Run: func(cmd *cobra.Command, args []string) {
			info, err := runtimes.SelectedRuntime.Info()
			if err != nil {
				l.Log().Fatalln(err)
			}
			err = yaml.NewEncoder(os.Stdout).Encode(info)
			if err != nil {
				l.Log().Fatalln(err)
			}
		},
		Hidden: true,
	})

	// Init
	cobra.OnInitialize(initLogging, initRuntime)

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cmd := NewCmdK3d()
	if len(os.Args) > 1 {
		parts := os.Args[1:]
		// Check if it's a built-in command, else try to execute it as a plugin
		if _, _, err := cmd.Find(parts); err != nil {
			pluginFound, err := cliutil.HandlePlugin(context.Background(), parts)
			if err != nil {
				l.Log().Errorf("Failed to execute plugin '%+v'", parts)
				l.Log().Fatalln(err)
			} else if pluginFound {
				os.Exit(0)
			}
		}
	}
	if err := cmd.Execute(); err != nil {
		l.Log().Fatalln(err)
	}
}

// initLogging initializes the logger
func initLogging() {
	if flags.traceLogging {
		l.Log().SetLevel(logrus.TraceLevel)
	} else if flags.debugLogging {
		l.Log().SetLevel(logrus.DebugLevel)
	} else {
		switch logLevel := strings.ToUpper(os.Getenv("LOG_LEVEL")); logLevel {
		case "TRACE":
			l.Log().SetLevel(logrus.TraceLevel)
		case "DEBUG":
			l.Log().SetLevel(logrus.DebugLevel)
		case "WARN":
			l.Log().SetLevel(logrus.WarnLevel)
		case "ERROR":
			l.Log().SetLevel(logrus.ErrorLevel)
		default:
			l.Log().SetLevel(logrus.InfoLevel)
		}
	}
	l.Log().SetOutput(ioutil.Discard)
	l.Log().AddHook(&writer.Hook{
		Writer: os.Stderr,
		LogLevels: []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
		},
	})
	l.Log().AddHook(&writer.Hook{
		Writer: os.Stdout,
		LogLevels: []logrus.Level{
			logrus.InfoLevel,
			logrus.DebugLevel,
			logrus.TraceLevel,
		},
	})

	formatter := &logrus.TextFormatter{
		ForceColors: true,
	}

	if flags.timestampedLogging || os.Getenv("LOG_TIMESTAMPS") != "" {
		formatter.FullTimestamp = true
	}

	l.Log().SetFormatter(formatter)

}

func initRuntime() {
	runtime, err := runtimes.GetRuntime("docker")
	if err != nil {
		l.Log().Fatalln(err)
	}
	runtimes.SelectedRuntime = runtime
	if rtinfo, err := runtime.Info(); err == nil {
		l.Log().Debugf("Runtime Info:\n%+v", rtinfo)
	}
}

func printVersion() {
	fmt.Printf("k3d version %s\n", version.GetVersion())
	fmt.Printf("k3s version %s (default)\n", version.K3sVersion)
}

// NewCmdCompletion creates a new completion command
func NewCmdCompletion(rootCmd *cobra.Command) *cobra.Command {

	completionFunctions := map[string]func(io.Writer) error{
		"bash": rootCmd.GenBashCompletion,
		"zsh": func(writer io.Writer) error {
			if err := rootCmd.GenZshCompletion(writer); err != nil {
				return err
			}

			fmt.Fprintf(writer, "\n# source completion file\ncompdef _k3d k3d\n")

			return nil
		},
		"psh":        rootCmd.GenPowerShellCompletion,
		"powershell": rootCmd.GenPowerShellCompletionWithDesc,
		"fish": func(writer io.Writer) error {
			return rootCmd.GenFishCompletion(writer, true)
		},
	}

	// create new cobra command
	cmd := &cobra.Command{
		Use:   "completion SHELL",
		Short: "Generate completion scripts for [bash, zsh, fish, powershell | psh]",
		Long: `To load completions:

Bash:

	$ source <(k3d completion bash)

	# To load completions for each session, execute once:
	# Linux:
	$ k3d completion bash > /etc/bash_completion.d/k3d
	# macOS:
	$ k3d completion bash > /usr/local/etc/bash_completion.d/k3d

Zsh:

	# If shell completion is not already enabled in your environment,
	# you will need to enable it.  You can execute the following once:

	$ echo "autoload -U compinit; compinit" >> ~/.zshrc

	# To load completions for each session, execute once:
	$ k3d completion zsh > "${fpath[1]}/k3d"

	# You will need to start a new shell for this setup to take effect.

fish:

	$ k3d completion fish | source

	# To load completions for each session, execute once:
	$ k3d completion fish > ~/.config/fish/completions/k3d.fish

PowerShell:

	PS> k3d completion powershell | Out-String | Invoke-Expression

	# To load completions for every new session, run:
	PS> k3d completion powershell > k3d.ps1
	# and source this file from your PowerShell profile.
`,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		ArgAliases:            []string{"psh"},
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if completionFunc, ok := completionFunctions[args[0]]; ok {
				if err := completionFunc(os.Stdout); err != nil {
					l.Log().Fatalf("Failed to generate completion script for shell '%s'", args[0])
				}
				return
			}
			l.Log().Fatalf("Shell '%s' not supported for completion", args[0])
		},
	}
	return cmd
}
