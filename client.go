//
// Client for RPC.

package main

import (
	"fmt"
	"net/rpc"
)

var (
	client *rpc.Client
)

// Establish the RPC client to query results.
func initClient() {
	var (
		err   error
		reply ResultsRPC
	)

	client, err = rpc.DialHTTP("tcp", fmt.Sprintf("localhost:%v", port))
	e(err)

	err = client.Call("Results.GetAllRPC", ArgsRPC{}, &reply)
	e(err)

	// TODO For now, just print results until we define actions that
	// may be done upon them.
	fmt.Printf("Got: %v\n", reply.Results)
}
