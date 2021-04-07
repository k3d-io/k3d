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
package main

import (
	"context"
	"fmt"
	"github.com/dtomasi/go-event-bus/v2"
	v1Cmd "github.com/rancher/k3d/v4/cmd/v1"
	v2Cmd "github.com/rancher/k3d/v4/cmd/v2"
	"github.com/rancher/k3d/v4/pkg/constants"
	"os"
	"os/signal"
	"syscall"
)

var (
	bus *eventbus.EventBus
	mainContext context.Context
	cancelFunc  context.CancelFunc
)

func init() {
	mainContext, cancelFunc = context.WithCancel(context.Background())
}

func main() {
	// Run experimental CLI
	if os.Getenv("K3D_CLI_EXPERIMENTAL") == "true" {

		bus = eventbus.DefaultBus()

		// Create a channel to listen to Done Event
		exitChan := eventbus.NewEventChannel()
		bus.SubscribeChannel(constants.EventExit, exitChan)

		// Create a channel to get os Signal SIGTERM
		osSignalCh := make(chan os.Signal)
		signal.Notify(osSignalCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			// Wait for os signals which we registered
			<-osSignalCh
			// Push an empty error
			bus.Publish(constants.EventExit, fmt.Errorf(""))
		}()

		// Run CLI in goroutine to not block here
		go func() {
			// recover panic
			defer func() {
				if err := recover(); err != nil {
					// Push the error we recovered to exit event channel
					bus.Publish(constants.EventExit, err)
				}
			}()
			// Execute Root Command
			v2Cmd.Execute(mainContext)
		}()

		// Wait for exit event
		exitEvent := <-exitChan

		// cancel main context
		cancelFunc()

		// Check if exit event has data
		if exitEvent.Data != nil {

			// Check if we got a error
			var ok bool
			_, ok = exitEvent.Data.(error)
			if ok {
				// TODO pretty print errors
				fmt.Println(exitEvent.Data)
			}

			if bus.HasSubscribers(constants.EventCleanup) {

				fmt.Println("Cleaning up. Please wait ... this may take a few moments!")

				// Publish the cleanup event
				bus.Publish(constants.EventCleanup, nil)
				fmt.Println("Cleanup done. Exiting ...")
			}

			// Exit with non zero since this was raised by an error
			os.Exit(1)

		}
		os.Exit(0)

	} else {
		v1Cmd.Execute()
	}
}
