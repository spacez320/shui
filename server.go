//
// Server for RPC.

package main

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"
)

// Establish the RPC server to allow access to stored results.
func initServer() {
	rpc.Register(&results)
	rpc.HandleHTTP()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	e(err)

	logger.Printf("Listening on :%s.\n", port)
	go http.Serve(listener, nil)
}
