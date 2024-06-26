/*
  Copyright 2024 Philip P. Ide

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.

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
	fetchServerTime(client, args) // fetch adjusted server time
	fmt.Sprintf("%d = adjusted server time\n", reply)

	args.Moment = now
	fetchAdjustTime(client, args)
	fmt.Sprintf("%d = adjusted local time\n", reply)

	fetchCalcRelativeTime(client, args)
	fmt.Sprintf("%d = adjustment for local time\n", reply)

	fetchAddRelativeTime(client, args)
	fmt.Sprintf("%d = adjusted Earth time to relativistic time\n", reply)

	// now set the private epoch for the dilation server
	// results are returned in global variable 'reply'
	// private epoch is the moment from which dilation should be adjusted
	// e.g. the moment when the clock is synchronised with UTC, which might happen at regular intervals
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

// Invoke the remote function ServerTime attached to TimeServer pointer
// Sending the arguments and reply variable address to the server as well
func fetchServerTime(client *rpc.Client, args Args) {
	err := client.Call("TimeServer.ServerTime", args, &reply)
	chkErr(err, "fetching time: %s", false)
	return
}

// Invoke the remote function ServerTime attached to TimeServer pointer
// Sending the arguments and reply variable address to the server as well
func fetchAdjustTime(client *rpc.Client, args Args) {
	err := client.Call("TimeServer.AdjustTime", args, &reply)
	chkErr(err, "fetching time: %s", false)
	return
}

// Invoke the remote function ServerTime attached to TimeServer pointer
// Sending the arguments and reply variable address to the server as well
func fetchAddRelativeTime(client *rpc.Client, args Args) {
	err := client.Call("TimeServer.AddRelativeTime", args, &reply)
	chkErr(err, "fetching time: %s", false)
	return
}

// Invoke the remote function ServerTime attached to TimeServer pointer
// Sending the arguments and reply variable address to the server as well
func fetchCalcRelativeTime(client *rpc.Client, args Args) {
	err := client.Call("TimeServer.CalcRelativeTime", args, &reply)
	chkErr(err, "fetching time: %s", false)
	return
}

func setPrivateEpoch(client *rpc.Client, args Args) {
	err := client.Call("TimeServer.SetPrvEpoch", args, &reply)
	chkErr(err, "setting private epoch: %s", false)
	return
}
