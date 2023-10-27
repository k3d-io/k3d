/*
Copyright © 2020-2023 The k3d Author(s)

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
package util

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/liggitt/tabwriter"
	"sigs.k8s.io/yaml"

	l "github.com/k3d-io/k3d/v5/pkg/logger"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

type NodePrinter interface {
	Print(*tabwriter.Writer, *k3d.Node)
}

type NodePrinterFunc func(*tabwriter.Writer, *k3d.Node)

func (npf NodePrinterFunc) Print(writter *tabwriter.Writer, node *k3d.Node) {
	npf(writter, node)
}

// PrintNodes prints a list of nodes, either as a table or as a JSON/YAML listing
func PrintNodes(nodes []*k3d.Node, outputFormat string, headers *[]string, nodePrinter NodePrinter) {
	outputFormat = strings.ToLower(outputFormat)

	tabwriter := tabwriter.NewWriter(os.Stdout, 6, 4, 3, ' ', tabwriter.RememberWidths)
	defer tabwriter.Flush()

	if outputFormat != "json" && outputFormat != "yaml" {
		if headers != nil {
			_, err := fmt.Fprintf(tabwriter, "%s\n", strings.Join(*headers, "\t"))
			if err != nil {
				l.Log().Fatalln("Failed to print headers")
			}
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	if outputFormat == "json" || outputFormat == "yaml" {
		var b []byte
		var err error

		switch outputFormat {
		case "json":
			b, err = json.Marshal(nodes)
		case "yaml":
			b, err = yaml.Marshal(nodes)
		}
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(b))
	} else {
		for _, node := range nodes {
			if !(outputFormat == "json" || outputFormat == "yaml") {
				nodePrinter.Print(tabwriter, node)
			}
		}
	}
}
