package cmd

import (
	"context"
	"fmt"

	"github.com/Twacqwq/mitmfoxy/proxy"
	"github.com/spf13/cobra"
)

var (
	// network server port
	port int
)

var rootCmd = &cobra.Command{
	Use:   "mitmproxy",
	Short: "a mitm proxy tools",
	Run: func(cmd *cobra.Command, args []string) {
		mitmproxy := proxy.New(&proxy.Config{
			Addr: fmt.Sprintf(":%d", port),
		})

		if err := mitmproxy.Start(); err != nil {
			panic(err)
		}
	},
}

func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.Flags().IntVarP(&port, "port", "p", 8989, "network server port")
}
