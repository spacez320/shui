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
	rpc.Register(&results)
	rpc.HandleHTTP()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	e(err)

	slog.Debug("Listening on :%s.\n", port)
	go http.Serve(listener, nil)
}
