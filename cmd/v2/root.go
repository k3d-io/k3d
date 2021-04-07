package v2

import (
	"context"
	"github.com/dtomasi/go-event-bus/v2"
	"github.com/rancher/k3d/v4/pkg/constants"
	"github.com/spf13/cobra"
)

var (
	err error

	// EventBus that handles graceful shutdown and more
	eventBus = eventbus.DefaultBus()

	// The root command ... This is the only one which we persist to package variable
	rootCmd *cobra.Command

	// The config
	k       = koanf.New(".")
)

// Initialize cobra with configs
func init() {
	rootCmd = NewRootCmd()
	cobra.OnInitialize(initConfig)
}

func initConfig() {

}

// Entrypoint from main.go
func Execute(ctx context.Context)  {
	eventBus.Publish(constants.EventExit, rootCmd.ExecuteContext(ctx))
}

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
	}
}
