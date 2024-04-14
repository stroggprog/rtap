/*
Rtaps creates an RPC server which provides functions for either:
	* providing adjusted timestamps collected from the server's own clock
	* adjusting timestamps sent to the server.
The adjustment corrects relativistic effects, retraining timestamps to be synchronous with UTC as experienced on Earth.

There are additional functions to:
	* provide an estimated real-time clock that attempts to negate request turn-around time (not recommended)
	* reset the private epoch - the time the system clock (or NTP server) was last retrained to UTC

Resetting the private epoch allows the clock to continue running instead of requiring a restart.
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
	"os/user"
	"time"
	"os/signal"
	"syscall"
	"strconv"
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

type TimeServer int64

func main() {
	mult = adjust
	fmt.Printf("Kill cmd: kill -s SIGINT %d\n",os.Getpid())
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

	fmt.Println("Setting unix epoch offset: " + fmt.Sprintf("%d", epochDiff))
	fmt.Println("Setting private epoch:", private_epoch)
	fmt.Printf("Granularity %d nanosecs (%d times per day)\n", adjust, divisor)
	fmt.Println("Listening on port:", serverPort)
}

func readIniFile() {
	serverPort = cfg.Section("Server").Key("http_port").MustInt(1234)

	private_epoch = cfg.Section("Epochs").Key("private").MustInt64(time.Now().UnixNano() + epochDiff)
	epochDiff = cfg.Section("Epochs").Key("epochDiff").MustInt64(0)

	divisor = cfg.Section("Ajustments").Key("DailyFractions").MustInt64(56)
	adjust = cfg.Section("Ajustments").Key("NanosecPerFraction").MustInt64(1000)
	codeTime = cfg.Section("Ajustments").Key("CodeTime").MustInt64(0)

	if !fileExists(iniFileName) {
		saveIniFile()
	}
}

func openIniFile() *ini.File {
	usr, _ := user.Current() // who are we?
	homedir = usr.HomeDir + "/.rtaps"
	os.MkdirAll(homedir, 0770) // make sure config data folder exists
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
	if err != nil {
		log.Fatal("listen error:", err)
	}
	fmt.Println("Ready")
	http.Serve(l, nil)
	fmt.Println("Bye")


	interruptSignal := make(chan os.Signal, 1)
    signal.Notify(interruptSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
    <-interruptSignal
    l.Close()
    fmt.Println("Bye.")
}

func fetchTime( inTime int64 ) int64 {
	t := (inTime + epochDiff) - private_epoch // get the current time less the period prior to prvt epoch
	e := t / (norm)                                          // divide by granularity to get number of granules
	t -= e * adjust                                          // subtract granules multiplied by adjustment per granule
	return t + private_epoch + codeTime                      // add the time prior to prvt epoch back and return to caller
}

// ============================
// RPC exported funcs

func (t *TimeServer) ServerTime(args *Args, reply *int64) error {
	*reply = fetchTime(time.Now().UnixNano())
	return nil
}

func (t *TimeServer) AdjustTime(args *Args, reply *int64) error {
	inTime, _ := strconv.ParseInt(args.Moment,10,64)
	*reply = fetchTime(inTime)
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

func (t *TimeServer) SetCodeTime(args *Args, reply *int64) error {
	v := args.Moment
	cfg.Section("Ajustments").Key("CodeTime").SetValue(v)
	saveIniFile()
	codeTime = cfg.Section("Ajustments").Key("CodeTime").MustInt64(0)
	*reply = codeTime
	fmt.Printf("Private epoch updated: %d\n", codeTime)
	return nil
}
