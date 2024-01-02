//
// Server for RPC.

package lib

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"

	"golang.org/x/exp/slog"
)

// Establish the RPC server to allow access to stored results.
func initServer(port string) {
	rpc.Register(&store)
	rpc.HandleHTTP()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	e(err)

	slog.Debug(fmt.Sprintf("Listening on :%v.\n", port))
	go http.Serve(listener, nil)
}
