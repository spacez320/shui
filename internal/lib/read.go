//
// Logic for 'read' mode.

package lib

func Read(port string) (done chan int) {
	// Start the RPC client.
	initClient(port)

	return
}
