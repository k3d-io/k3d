/*
Copyright © 2020-2022 The k3d Author(s)

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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/k3d-io/k3d/v5/cmd/cluster"
	cfg "github.com/k3d-io/k3d/v5/cmd/config"
	"github.com/k3d-io/k3d/v5/cmd/debug"
	"github.com/k3d-io/k3d/v5/cmd/image"
	"github.com/k3d-io/k3d/v5/cmd/kubeconfig"
	"github.com/k3d-io/k3d/v5/cmd/node"
	plugin "github.com/k3d-io/k3d/v5/cmd/plugin"
	"github.com/k3d-io/k3d/v5/cmd/registry"
	cliutil "github.com/k3d-io/k3d/v5/cmd/util"
	l "github.com/k3d-io/k3d/v5/pkg/logger"
	k3dPlugin "github.com/k3d-io/k3d/v5/pkg/plugin"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/version"
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
	rootCmd.AddCommand(
		NewCmdVersion(),
		NewCmdCompletion(rootCmd),
		cluster.NewCmdCluster(),
		kubeconfig.NewCmdKubeconfig(),
		node.NewCmdNode(),
		image.NewCmdImage(),
		cfg.NewCmdConfig(),
		registry.NewCmdRegistry(),
		debug.NewCmdDebug(),
		plugin.NewCmdPlugin(),
		&cobra.Command{
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
		},
	)

	loadPluginCommands(rootCmd)

	// Init
	cobra.OnInitialize(initLogging, initRuntime)

	return rootCmd
}

// loadPluginCommands dinamically adds plugin commands
func loadPluginCommands(cmd *cobra.Command) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		l.Log().Fatalln(err)
	}

	// TODO: Override this variables from cobra/viper or using env vars
	pluginPath := filepath.Join(userHomeDir, ".k3d", "plugins")
	manifestName := k3dPlugin.DefaultManifestName

	plugins, err := k3dPlugin.LoadAll(pluginPath, manifestName)
	if err != nil {
		l.Log().Fatalln(err)
	}

	for _, plugin := range plugins {
		pluginCommand := &cobra.Command{
			Use:   plugin.Manifest.Name,
			Short: plugin.Manifest.ShortHelpMessage,
			Long:  plugin.Manifest.HelpMessage,
			Run: func(cmd *cobra.Command, args []string) {
				// Lookup $SHELL variable to determine user's shell
				userShell := os.Getenv("SHELL")

				// Execute plugin command in a separate shell in order to allow using pipes
				pluginExec := exec.Command(userShell, "-c", plugin.Manifest.Command)
				pluginExec.Env = os.Environ()
				pluginExec.Stdin = os.Stdin
				pluginExec.Stdout = os.Stdout
				pluginExec.Stderr = os.Stderr

				err := pluginExec.Run()
				if err != nil {
					l.Log().Errorln(err)
				}
			},
		}
		cmd.AddCommand(pluginCommand)
	}
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
	l.Log().SetLevel(logrus.InfoLevel) // default log level: info
	if flags.traceLogging {
		l.Log().SetLevel(logrus.TraceLevel)
	} else if flags.debugLogging {
		l.Log().SetLevel(logrus.DebugLevel)
	} else {
		if ll := os.Getenv("LOG_LEVEL"); ll != "" {
			level, err := logrus.ParseLevel(ll)
			if err == nil {
				l.Log().SetLevel(level)
			}
		}
	}
	l.Log().SetOutput(io.Discard)
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

	if logColor, err := strconv.ParseBool(os.Getenv("LOG_COLORS")); err == nil {
		formatter.ForceColors = logColor
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

func NewCmdVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show k3d and default k3s version",
		Long:  "Show k3d and default k3s version",
		Run: func(cmd *cobra.Command, args []string) {
			printVersion()
		},
		Args: cobra.NoArgs,
	}

	cmd.AddCommand(NewCmdVersionLs())

	return cmd

}

func printVersion() {
	fmt.Printf("k3d version %s\n", version.GetVersion())
	fmt.Printf("k3s version %s (default)\n", version.K3sVersion)
}

func NewCmdVersionLs() *cobra.Command {

	type VersionLsOutputFormat string
	type VersionLsSortMode string

	const (
		VersionLsOutputFormatRaw  VersionLsOutputFormat = "raw"
		VersionLsOutputFormatRepo VersionLsOutputFormat = "repo"

		VersionLsSortDesc VersionLsSortMode = "desc"
		VersionLsSortAsc  VersionLsSortMode = "asc"
		VersionLsSortOff  VersionLsSortMode = "off"
	)

	var VersionLsOutputFormats = map[string]VersionLsOutputFormat{
		string(VersionLsOutputFormatRaw):  VersionLsOutputFormatRaw,
		string(VersionLsOutputFormatRepo): VersionLsOutputFormatRepo,
	}

	var VersionLsSortModes = map[string]VersionLsSortMode{
		string(VersionLsSortDesc): VersionLsSortDesc,
		string(VersionLsSortAsc):  VersionLsSortAsc,
		string(VersionLsSortOff):  VersionLsSortOff,
	}

	type Flags struct {
		includeRegexp string
		excludeRegexp string
		format        string
		sortMode      string
		limit         int
	}

	flags := Flags{}

	cmd := &cobra.Command{
		Use:       "list",
		Aliases:   []string{"ls"},
		Short:     "List k3d/K3s versions",
		ValidArgs: []string{"k3d", "k3s", "k3d-proxy", "k3d-tools"},
		Args:      cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var format VersionLsOutputFormat
			if f, ok := VersionLsOutputFormats[flags.format]; !ok {
				l.Log().Fatalf("Unknown output format '%s'", flags.format)
			} else {
				format = f
			}

			var sortMode VersionLsSortMode
			if m, ok := VersionLsSortModes[flags.sortMode]; !ok {
				l.Log().Fatalf("Unknown sort mode '%s'", flags.sortMode)
			} else {
				sortMode = m
			}

			urlTpl := "https://registry.hub.docker.com/v1/repositories/%s/tags"
			org := "rancher"
			repo := fmt.Sprintf("%s/%s", org, args[0])
			resp, err := http.Get(fmt.Sprintf(urlTpl, repo))
			if err != nil {
				l.Log().Fatalln(err)
			}
			defer resp.Body.Close()
			type Layers struct {
				Layer string
				Name  string
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				l.Log().Fatalln(err)
			}
			respJSON := &[]Layers{}
			if err := json.Unmarshal(body, respJSON); err != nil {
				l.Log().Fatalln(err)
			}

			includeRegexp, err := regexp.Compile(flags.includeRegexp)
			if err != nil {
				l.Log().Fatalln(err)
			}

			excludeRegexp, err := regexp.Compile(flags.excludeRegexp)
			if err != nil {
				l.Log().Fatalln(err)
			}

			tags := []string{}

			for _, tag := range *respJSON {
				if includeRegexp.Match([]byte(tag.Name)) {
					if flags.excludeRegexp == "" || !excludeRegexp.Match([]byte(tag.Name)) {
						switch format {
						case VersionLsOutputFormatRaw:
							tags = append(tags, tag.Name)
						case VersionLsOutputFormatRepo:
							tags = append(tags, fmt.Sprintf("%s:%s\n", repo, tag.Name))
						default:
							l.Log().Fatalf("Unknown output format '%+v'", format)
						}
					} else {
						l.Log().Tracef("Tag %s excluded (regexp: `%s`)", tag.Name, flags.excludeRegexp)
					}
				} else {
					l.Log().Tracef("Tag %s not included (regexp: `%s`)", tag.Name, flags.includeRegexp)
				}
			}

			// Sort
			if sortMode != VersionLsSortOff {
				sort.Slice(tags, func(i, j int) bool {
					if sortMode == VersionLsSortAsc {
						return tags[i] < tags[j]
					}
					return tags[i] > tags[j]
				})
			}

			if flags.limit > 0 {
				tags = tags[0:flags.limit]
			}
			fmt.Println(strings.Join(tags, "\n"))

		},
	}

	cmd.Flags().StringVarP(&flags.includeRegexp, "include", "i", ".*", "Include Regexp (default includes everything")
	cmd.Flags().StringVarP(&flags.excludeRegexp, "exclude", "e", ".+(rc|engine|alpha|beta|dev|test|arm|arm64|amd64).*", "Exclude Regexp (default excludes pre-releases and arch-specific tags)")
	cmd.Flags().StringVarP(&flags.format, "format", "f", string(VersionLsOutputFormatRaw), "Output Format")
	cmd.Flags().StringVarP(&flags.sortMode, "sort", "s", string(VersionLsSortDesc), "Sort Mode (asc | desc | off)")
	cmd.Flags().IntVarP(&flags.limit, "limit", "l", 0, "Limit number of tags in output (0 = unlimited)")

	return cmd
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
	$ k3d completion zsh > "${fpath[1]}/_k3d"

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
