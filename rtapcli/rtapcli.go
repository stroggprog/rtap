/*
This code is released under the GNU GPL v3.0 licence. Any third-prty modules or software this code
relies on in order to run, may have their own licencing requirements and stipulations. A copy of the licence
should always accompany this code.

Code originally written by Philip Ide, and is based on code by atanda nafiu, from the code
example at https://dev.to/atanda0x/a-beginners-guide-to-rpc-in-golang-understanding-the-basics-4eeb
*/
package main

import (
	"fmt"
	"log"
	"net/rpc"
	"time"
)

var reply int64

func main() {
	// Address to this variable will be sent to the RPC server
	// Type of reply should be same as that specified on server
	args := Args{}

	// DialHTTP connects to an HTTP RPC server at the specified network
	client, err := rpc.DialHTTP("tcp", "localhost"+":1234")
	if err != nil {
		log.Fatal("Client connection error: ", err)
	}

	// firstly output current time as seen on system clock
	now := fmt.Sprintf("%d", time.Now().UnixNano())
	fmt.Printf("%s = now\n", now)

	// now fetch the current time adjusted for lunar time dilation
	// results are returned in global variable 'reply'
	args.Moment = "0" // dummy value, not used in his call
	fetchTime(client, args)
	fmt.Sprintf("%d = timeserver\n", reply)

	// now set the private epoch for the dilation server
	// results are returned in global variable 'reply'
	// private epoch is the moment from which dilation should be adjusted
	// e.g. the moment when the clock is synchronised with UTC, which might happen at regular intervals
	args.Moment = now
	setPrivateEpoch(client, args)
	fmt.Printf("Clock private epoch reset to %d\n", reply)
}

func chkErr(err error, s string, fatal bool) {
	if err != nil && fatal {
		log.Panicf(s, err)
	} else if err != nil {
		log.Printf(s, err)
	}
}

// Invoke the remote function GiveServerTime attached to TimeServer pointer
// Sending the arguments and reply variable address to the server as well
func fetchTime(client *rpc.Client, args ppirpcstructs.Args) {
	err := client.Call("TimeServer.ServerTime", args, &reply)
	chkErr(err, "fetching time: %s", false)
	return
}

func setPrivateEpoch(client *rpc.Client, args ppirpcstructs.Args) {
	err := client.Call("TimeServer.SetPrvEpoch", args, &reply)
	chkErr(err, "setting private epoch: %s", false)
	return
}
