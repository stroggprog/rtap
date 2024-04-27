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

Clients can call four functions to get adjusted time:
```
ServerTime( arg *Args, reply *int64 ) 		// fetches adjusted server time
AdjustTime( arg *Args, reply *int64 )		// adjusts provided timestamp
CalcRelativeTime(arg *Args, reply *int64)	// returns the difference between the provided timestamp and Earth-UTC
AddRelativeTime( arg *Args, reply *int64 )	// Earth-UTC converted to un-adjusted gravitational time
RelativeUnix(args *Args, reply *int64)		// returns the amount of adjustment between given timestamp and unix epoch
AdjustUTCTZ(args *Args, reply *int64)		// returns timestamp adjusted since TZ epoch (UTC @ 1900-01-01 00:00:00)
RelativeUTCTZ(args *Args, reply *int64)		// returns amount of adjustment on given timestamp since TZ epoch
FixDrift( args *Args, reply *int64)			// returns a value to fix clock drift
```
The `Args` structure is:
```
type Args struct {
	Moment string
}
```
Where `Moment` is a stringified int64 number. When calling `ServerTime`, `Moment` should be set to "0". When calling `AdjustTime`, it should be set to the timestamp (in nanoseconds) to be adjusted.

`CalcRelativeTime` returns the difference between a timestamp affected by relativity and Earth-UTC.

If the local timekeeper (NTP server) is retrained to Earth-UTC, the moment - expressed in nanoseconds - should be pushed to `rtaps` using the function:
```
SetPrvEpoch( arg *Args, reply *int64 )
```
This sets the private epoch to the value passed. This new private epoch becomes active immediately, and the configuration file is updated in case there is a restart.

`AddRelativeTime` is a function that is the opposite of `ServerTime` & `AdjustTime`, in that it is meant to be used on Earth to return the un-adjusted time on the other gravitational body (e.g. Earth's moon). This takes a timestamp expressed in nanoseconds, where 0 (zero) is the beginning of the unix epoch. Any offset from the epoch will be added, the amount of time that would have advanced is added (e.g. 1 nanosecond per 1/58.7th of 24hrs) and then returned to the caller.

`RelativeUnix` returns the amount of adjustment between the given timestamp and the Unix Epoch (1970-01-01 00:00:00). Note that if a different epoch is used for other calculations, this function's return value will differ by the difference between the used and the Unix epochs.

`AdjustUTCTZ` returns the timestamp adjusted using the internationally recognised Timezone epoch, which is 1900-01-01 00:00:00 UTC.

`RelativeUTCTZ` returns the amount of adjustment that would have been applied to a timestamp using the Timezone epoch instead of the Unix or user-defined (via the epochDiff setting in the ini file) epoch.

`FixDrift` accepts a timestamp and returns the amount of drift the clock would have experienced during that period. All software/hardware clocks - other than atomic clocks - drift because they're just not that accurate (if they were, we wouldn't need atomic clocks). When requesting `ServerTime` or calling `AdjustTime` with value `0`, drift is automatically calculated and applied to the return value. For all other calls, including calling `AdjustTime` with a non-zero value, drift is __not__ calculated or applied. If a drift value is required, the value of the server's drift can be returned via this method so it can be retrospectively applied to a client timestamp.