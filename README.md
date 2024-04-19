# rtap
The software is an RPC server written in go (golang), which either:
* accepts a timestamp, adjusts and returns it or
* fetches a timestamp from its own system clock, adjusts and returns it

The adjustment accounts for the difference between UTC as experienced on Earth versus UTC experienced in the local time-frame (e.g. on Earth's moon), where time passes at a different rate than it does on Earth. The purpose is to return the timestamp synchronised with UTC on Earth by removing relativistic effects.

There's nothing particularly clever going on, the clock simply calculates how far adrift the local clock is from Earth-UTC and adjusts. There are three points in time:

1. the start of the epoch 
2. the moment when the local clock was last synched with Earth-UTC (private epoch) 
3. the current (local) time expressed as UTC in nanoseconds

The server can either accept a timestamp from a remote device, or it can fetch one from its own clock. It then calculates the number of nanoseconds since the start of the unix epoch. The clock though, was accurate up to the moment it was last synchronised with Earth (known as the private epoch), so time after that point is separated, with time prior to that event stored (let's call it PRIOR). The drift per day due to relativistic effects are known, and used to divide the remaining time into granules. The number of granules are then multiplied by the divisor, which are then subtracted from the 'remaining' time, the PRIOR time is added back, and the total returned to the caller.

In this way, only time beyond the private epoch is adjusted.

If the system clock (or local NTP server) is retrained to Earth-UTC, there is a function to adjust the private epoch.

Major system parameters such as the private epoch are store in a file and reloaded after a restart.

## Installation:

If you don't have 'go' installed, please go to https://go.dev/ and follow the instructions.

After pulling the repository:
```
go mod tidy
go build .
./rtaps
```

If everything runs well:
```
go install
```

For the test client (in a separate terminal window):
```
cd rtapcli
go mod tidy
go run .
```

The test client should output some data and then terminate.

To exit the server on Windows, type Ctrl-C. You can do so on Linux too, but the server also outputs the kill command if you prefer to do it that way. I have no idea how to terminate it on a Mac, but if you're running one I'm sure you do.

## Functions

__All functions will return an error *to the RPC server code* in the event of an error.__

Clients can call two functions to get adjusted time:
```
ServerTime( arg *Args, reply *int64 ) // fetches adjusted server time
AdjustTime( arg *Args, reply *int64 ) // adjusts provided time
```
The `Args` structure is:
```
type Args struct {
	Moment string
}
```
Where `Moment` is a stringified int64 number. When calling `ServerTime`, `Moment` should be set to "0". When calling `AdjustTime`, it should be set to the timestamp (in nanoseconds) to be adjusted.

If the local timekeeper (NTP server) is retrained to Earth-UTC, the moment - expressed in nanoseconds - should be pushed to `rtaps` using the function:
```
SetPrvEpoch( arg *Args, reply *int64 )
```
This sets the private epoch to the value passed. This new private epoch becomes active immediately, and the configuration file is updated in case there is a restart.

There is an additional function that can be called:
```
SetCodeTime( arg *Args, reply *int64 )
```
This function is not recommended, but is included for experimentation. Its purpose is to take into consideration the turn-around time for calling the RPC functions and receiving a result by the caller. The intention is that when an adjusted timestamp is recieved, it is not stale (e.g. synched with Earth-UTC). There are too many variables involved (such as network latency, CPU utilisation on both machones etc.) to make this in any way accurate except by chance, and even then only on a request by request basis. However, if an additional *predictable* moderator is required, the `codeTime` variable can be leveraged to do something useful.

A better solution would be to snapshot the time at the moment the rseult is returned. The difference between this and the original time sent to the RPC server is the turnaround time. However, taking the new snapshot and comparing it to the original timestamp will distort the result.
