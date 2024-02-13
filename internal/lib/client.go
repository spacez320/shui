//
// Client for RPC.

package lib

import (
	"fmt"
	"net/rpc"

	"pkg/storage"
)

var (
	client *rpc.Client
)

// Establish the RPC client to query results.
func initClient(port string) {
	var (
		err   error
		reply storage.ResultsRPC
	)

	client, err = rpc.DialHTTP("tcp", fmt.Sprintf("localhost:%v", port))
	e(err)

	err = client.Call("Results.GetAllRPC", storage.ArgsRPC{}, &reply)
	e(err)

	// TODO For now, just print results until we define actions that may be done upon them.
	fmt.Printf("Got: %v\n", reply.Results)
}
