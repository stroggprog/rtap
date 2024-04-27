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
*/
/*
Rtaps creates an RPC server which provides functions for either:
  - providing adjusted timestamps collected from the server's own clock
  - adjusting timestamps sent to the server.

The adjustment corrects relativistic effects, retraining timestamps to be synchronous with UTC as experienced on Earth.

There are additional functions to:
  - provide an estimated real-time clock that attempts to negate request turn-around time (not recommended)
  - reset the private epoch - the time the system clock (or NTP server) was last retrained to UTC

Resetting the private epoch allows the clock to continue running instead of requiring a restart.
Settings are saved in $HOME/.rtaps/rtaps.ini

This code is released under the GNU GPL v3.0 licence. Any third-party modules or software this code
relies on in order to run, may have their own licencing requirements and stipulations. A copy of the licence
should always accompany this code.

Alternate licencing may be available on request.

Code originally written by Philip Ide, and is based on code by atanda nafiu, from the code
example at https://dev.to/atanda0x/a-beginners-guide-to-rpc-in-golang-understanding-the-basics-4eeb
*/
package main

import (
	"fmt"
	"gopkg.in/ini.v1"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

var norm int64 = (86400 * 1e9) // one day in nanoseconds
var adjust int64 = 7 * 1e3     // 1/8th of 56 microseconds, the dilation, normalised to an integer in nanoseconds
var epochDiff int64 = 0        // offset in nanoseconds if epoch is not 1970-01-01 00:00:00. Positive if earlier, negative if later.
var mult int64 = 0
var private_epoch int64 = 0
var homedir string
var serverPort = 1234
var divisor int64
var codeTime int64
var err error
var cfg *ini.File
var iniFileName string
var tznano int64

var driftFraction int64
var driftAdjust int64

type TimeServer int64

func main() {
	mult = adjust
	fmt.Printf("Kill cmd: kill -s SIGINT %d\n", os.Getpid())
	configure()
	clock()
}

func chkErr(err error, s string, fatal bool) {
	if err != nil && fatal {
		log.Panicf(s, err)
	} else if err != nil {
		log.Printf(s, err)
	}
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	return false
}

func configure() {
	openIniFile()
	readIniFile()
	norm = norm / divisor

	tznano = -time.Date(1900, 01, 01, 0, 0, 0, 0, time.Local).UTC().UnixNano()
	//utc := tznano.UTC().UnixNano()

	fmt.Println("TZ Epoch:", tznano)
	fmt.Println("Setting unix epoch offset: " + fmt.Sprintf("%d", epochDiff))
	fmt.Println("Setting private epoch:", private_epoch)
	fmt.Printf("Granularity %d nanosecs (%d times per day)\n", adjust, divisor)
	fmt.Println("Listening on port:", serverPort)
}

func readIniFile() {
	serverPort = cfg.Section("Server").Key("http_port").MustInt(1234)

	private_epoch = cfg.Section("Epochs").Key("private").MustInt64(time.Now().UnixNano() + epochDiff)
	epochDiff = cfg.Section("Epochs").Key("epochDiff").MustInt64(0)

	divisor = cfg.Section("Ajustments").Key("DailyFractions").MustInt64(587)
	adjust = cfg.Section("Ajustments").Key("NanosecPerFraction").MustInt64(100)

	// if drift isn't catered for, set
	// driftFraction = 1
	// driftAdjust = 0
	//
	// driftFraction is the granularity (1 = everything)
	// driftAdjust is how much drift to account for per granule (0 = no drift)
	//
	driftFraction = cfg.Section("Adjustments").Key("driftFractions").MustInt64(1)
	driftAdjust = cfg.Section("Adjustments").Key("driftFractions").MustInt64(0)
	codeTime = cfg.Section("Ajustments").Key("CodeTime").MustInt64(0)

	if !fileExists(iniFileName) {
		saveIniFile()
	}
}

func openIniFile() *ini.File {
	usr, err := user.Current() // who are we?
	chkErr(err, "failed checking user", true)

	homedir = usr.HomeDir + "/.rtaps"
	err = os.MkdirAll(homedir, 0770) // make sure config data folder exists
	chkErr(err, "failed to create folder ~/.rtaps", true)

	iniFileName = homedir + "/rtaps.ini"
	cfg, err = ini.LooseLoad(iniFileName)
	chkErr(err, "Couldn't load ini file: %v", false)
	return cfg
}

func saveIniFile() {
	cfg.SaveTo(iniFileName)
}

func clock() {
	timeserver := new(TimeServer)
	rpc.Register(timeserver)
	rpc.HandleHTTP()
	sport := fmt.Sprintf(":%d", serverPort)
	l, err := net.Listen("tcp", sport)
	chkErr(err, "listen error", true)

	fmt.Println("Ready")
	go http.Serve(l, nil)

	interruptSignal := make(chan os.Signal, 1)
	signal.Notify(interruptSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-interruptSignal

	l.Close()
	fmt.Println("\nBye.")
}

func fixDrift(inTime int64) int64 {
	e := inTime / driftFraction
	return t - (e * driftAdjust)
}

func fetchTime(inTime int64, fixdrift bool) int64 {
	t := (inTime + epochDiff) - private_epoch // get the current time less the period prior to prvt epoch
	if fixdrift {
		t = fixDrift(t)
	}
	e := t / (norm)          // divide by granularity to get number of granules
	t -= (e * adjust)        // subtract granules multiplied by adjustment per granule
	return t + private_epoch // add the time prior to prvt epoch back and return to caller
}

// ============================
// RPC exported funcs

func (t *TimeServer) FixDrift(args *Args, reply *int64) error {
	inTime, err := strconv.ParseInt(args.Moment, 10, 64)
	if err != nil {
		inTime = time.Now().UnixNano()
	}
	*reply = fixDrift(inTime)
	return nil
}

// returns the current adjusted time as reported by the local clock
func (t *TimeServer) ServerTime(args *Args, reply *int64) error {
	*reply = fetchTime(time.Now().UnixNano(), true)
	return nil
}

// if AdjustTime doesn't receive a parameter, it behaves exactly
// like ServerTime(), otherwise adjusts given timestamp
func (t *TimeServer) AdjustTime(args *Args, reply *int64) error {
	fixdrift := false
	inTime, err := strconv.ParseInt(args.Moment, 10, 64)
	if err != nil || inTime == 0 {
		inTime = time.Now().UnixNano()
		fixdrift = true
	}
	*reply = fetchTime(inTime, fixdrift)
	return nil
}

// given a timestamp, it returns the difference between the adjusted timestamp and Earth-UTC
// never fix drift when a timestamp is provided
func (t *TimeServer) CalcRelativeTime(args *Args, reply *int64) error {
	inTime, err := strconv.ParseInt(args.Moment, 10, 64)
	if err != nil || inTime == 0 {
		inTime = time.Now().UnixNano()
	}
	t0 := (inTime + epochDiff)
	e := t0 / norm
	*reply = (e * adjust)
	return nil
}

// when used on Earth, adjusts a given timestamp to show local time on other body (e.g. moon)
func (t *TimeServer) AddRelativeTime(args *Args, reply *int64) error {
	inTime, err := strconv.ParseInt(args.Moment, 10, 64)
	if err != nil || inTime == 0 {
		inTime = time.Now().UnixNano()
	}
	t0 := (inTime + epochDiff)
	e := t0 / norm
	*reply = t0 + (e * adjust)
	return nil
}

// returns the amount of adjustment between given timestamp and unix epoch
// timestamp defaults to current time on server
func (t *TimeServer) RelativeUnix(args *Args, reply *int64) error {
	inTime, err := strconv.ParseInt(args.Moment, 10, 64)
	if err != nil || inTime == 0 {
		inTime = time.Now().UnixNano()
	}
	e := inTime / norm
	*reply = e * adjust
	return nil
}

// returns given timestamp adjusted since TZ epoch (UTC @ 1900-01-01 00:00:00)
func (t *TimeServer) AdjustUTCTZ(args *Args, reply *int64) error {
	inTime, err := strconv.ParseInt(args.Moment, 10, 64)
	if err != nil || inTime == 0 {
		inTime = time.Now().UnixNano()
	}
	inTime = tznano + inTime
	e := inTime / norm
	*reply = (inTime - tznano) + (e * adjust)
	return nil
}

// returns amount of adjustment on given timestamp since TZ epoch (UTC @ 1900-01-01 00:00:00)
func (t *TimeServer) RelativeUTCTZ(args *Args, reply *int64) error {
	inTime, err := strconv.ParseInt(args.Moment, 10, 64)
	if err != nil || inTime == 0 {
		inTime = time.Now().UnixNano()
	}
	inTime = tznano + inTime
	e := inTime / norm
	*reply = (e * adjust)
	return nil
}

func (t *TimeServer) SetPrvEpoch(args *Args, reply *int64) error {
	v := args.Moment
	cfg.Section("Epochs").Key("private").SetValue(v)
	saveIniFile()
	private_epoch = cfg.Section("Epochs").Key("private").MustInt64(0)
	*reply = private_epoch
	fmt.Printf("Private epoch updated: %d\n", private_epoch)
	return nil
}

/*
func (t *TimeServer) SetCodeTime(args *Args, reply *int64) error {
	v := args.Moment
	cfg.Section("Ajustments").Key("CodeTime").SetValue(v)
	saveIniFile()
	codeTime = cfg.Section("Ajustments").Key("CodeTime").MustInt64(0)
	*reply = codeTime
	fmt.Printf("Private epoch updated: %d\n", codeTime)
	return nil
}
*/
