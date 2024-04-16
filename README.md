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


