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

	// root ca cert file
	certFile string

	// root ca key file
	keyFile string

	// use websocket to recv packet capture
	useWebsocket bool
)

var rootCmd = &cobra.Command{
	Use:   "mitmproxy",
	Short: "a mitm proxy tools",
	Run: func(cmd *cobra.Command, args []string) {
		mitmproxy := proxy.New(&proxy.Config{
			Addr:         fmt.Sprintf(":%d", port),
			CertFile:     certFile,
			KeyFile:      keyFile,
			UseWebsocket: useWebsocket,
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
	rootCmd.Flags().StringVarP(&certFile, "cert", "c", "", "root ca cert file")
	rootCmd.Flags().StringVarP(&keyFile, "key", "k", "", "root ca key file")
	rootCmd.Flags().BoolVarP(&useWebsocket, "ws", "w", false, "use websocket to recv packet capture")
}
