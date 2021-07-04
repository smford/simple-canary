package main

import (
	//"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const APPVERSION = "0.0.1"
const APITOKEN = "sometoken"
const LISTENIP = "0.0.0.0"
const LISTENPORT = "54034"
const INDEXHTML = "index.html"

var (
	allDevices     []Device
	allDevicesTime []DeviceTime
)

var mapSteve = make(map[string]time.Time)

type Device struct {
	Name        string `json:"name"`
	Lastcheckin string `json:"lastcheckin,omitempty"`
}

type DeviceTime struct {
	Name        string    `json:"name"`
	Lastcheckin time.Time `json:"lastcheckin,omitempty"`
}

func init() {
	fmt.Println("Simple-canary v" + APPVERSION)
	flag.Bool("help", false, "Display help")
	flag.String("config", "config.yaml", "Configuration file: /path/to/file.yaml, default = ./config.yaml")
	flag.Bool("version", false, "Display version")
	flag.Bool("verbose", true, "Be verbose")
	flag.Bool("displayconfig", false, "Display configuration")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	if viper.GetBool("help") {
		displayHelp()
		os.Exit(0)
	}

	if viper.GetBool("version") {
		os.Exit(0)
	}

	configdir, configfile := filepath.Split(viper.GetString("config"))

	// set default configuration directory to current directory
	if configdir == "" {
		configdir = "."
	}

	if viper.GetBool("verbose") {
		fmt.Println("   DIR:", configdir)
		fmt.Println("  FILE:", configfile)
	}

	viper.SetConfigType("yaml")
	viper.AddConfigPath(configdir)

	config := strings.TrimSuffix(configfile, ".yaml")
	config = strings.TrimSuffix(config, ".yml")

	viper.SetConfigName(config)
	err := viper.ReadInConfig()
	if err != nil {
		if !viper.GetBool("silent") {
			fmt.Println("ERROR: No config file found")
			if viper.GetBool("verbose") {
				fmt.Printf("%s\n", err)
			}
			os.Exit(1)
		}
	}

	// configure in memory device tracking
	for _, v := range viper.GetStringSlice("devices") {
		fmt.Printf("value[%s]\n", v)

		d1 := Device{
			Name:        v,
			Lastcheckin: "poos",
		}

		d1time := DeviceTime{
			Name:        v,
			Lastcheckin: time.Now(),
		}

		allDevices = append(allDevices, d1)
		allDevicesTime = append(allDevicesTime, d1time)
	}

	fmt.Println("allDevices unformated:")
	fmt.Println(allDevices)
	fmt.Println("========")

	fmt.Println("allDevices=")
	for k, v := range allDevices {
		fmt.Printf("key[%d] value[%s]\n", k, v)
	}
	fmt.Println("-------------------------------------")
	fmt.Println("allDevicesTime unformated:")
	fmt.Println(allDevicesTime)
	fmt.Println("========")

	fmt.Println("allDevicesTime=")
	for k, v := range allDevicesTime {
		fmt.Printf("key[%d] value[%s]\n", k, v)
	}

	// make a map  device["name"]
	//mapSteve := make(map[string]time.Time)

	fmt.Println("=-=-=-=-=-=-=-=-=-=-=-=-=-=-=")
	for _, v := range viper.GetStringSlice("devices") {
		mapSteve[strings.ToLower(v)] = time.Now()
	}

	for _, v := range viper.GetStringSlice("devices") {
		fmt.Printf("v=%s time=%s\n", v, mapSteve[v].String())
	}
	fmt.Println("=-=-=-=-=-=-=-=-=-=-=-=-=-=-=")

}

func main() {
	if viper.GetBool("displayconfig") {
		displayConfig()
		os.Exit(0)
	}

	fmt.Println("Simple-canary is now running.  Press CTRL-C to exit.")
	startWeb(LISTENIP, LISTENPORT, false)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

}

func startWeb(listenip string, listenport string, usetls bool) {
	r := mux.NewRouter()

	// default index page
	r.HandleFunc("/", handlerIndex)

	// per device checkin
	r.HandleFunc("/checkin/{device}", handlerCheckin)

	// per device status
	r.HandleFunc("/status/{device}", handlerStatus)

	// all devices status
	r.HandleFunc("/status", handlerStatus)

	// enable logging
	r.Use(loggingMiddleware)

	log.Printf("Starting HTTP Webserver http://%s:%s\n", listenip, listenport)

	srv := &http.Server{
		Handler:      r,
		Addr:         LISTENIP + ":" + LISTENPORT,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	err := srv.ListenAndServe()

	fmt.Println("cannot start http server:", err)
}

func printFile(filename string, webprint http.ResponseWriter) {
	fmt.Println("Starting printFile")
	texttoprint, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("ERROR: cannot open ", filename)
		if webprint != nil {
			http.Error(webprint, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	}
	if webprint != nil {
		fmt.Fprintf(webprint, "%s", string(texttoprint))
	} else {
		fmt.Print(string(texttoprint))
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("MIDDLEWARE: ", r.RemoteAddr, " ", r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func handlerIndex(w http.ResponseWriter, r *http.Request) {
	log.Println("Starting handlerIndex")
	printFile(INDEXHTML, w)
}

func handlerCheckin(webprint http.ResponseWriter, r *http.Request) {
	fmt.Println("starting handlercheckin")
	queries := r.URL.Query()
	fmt.Printf("queries = %q\n", queries)

	// check if api token is valid
	if APITOKEN != queries.Get("token") {
		fmt.Println("ERROR: Invalid API Token", queries.Get("token"))
		fmt.Fprintf(webprint, "%s", "ERROR: Invalid API Token")
		return
	}
}

func handlerStatus(webprint http.ResponseWriter, r *http.Request) {
	fmt.Println("starting handlerstatus")
	queries := r.URL.Query()
	fmt.Printf("queries = %q\n", queries)

	// check if api token is valid
	/*
		if APITOKEN != queries.Get("token") {
			fmt.Println("ERROR: Invalid API Token", queries.Get("token"))
			fmt.Fprintf(webprint, "%s", "ERROR: Invalid API Token")
			return
		}
	*/

	vars := mux.Vars(r)

	webprint.WriteHeader(http.StatusOK)

	if len(vars["device"]) > 0 {
		fmt.Fprintf(webprint, "%s%s", "Device=", vars["device"])
		//fmt.Fprintf(webprint, "\n%s%s", "struct=", allDevices[0])
		//mapSteve[strings.ToLower(vars["device"])]
		fmt.Fprintf(webprint, "\n\nSteveDevice=%s\nSteveTime=%s", strings.ToLower(vars["device"]), mapSteve[strings.ToLower(vars["device"])])
	} else {
		//fmt.Fprintf(webprint, "%s", viper.GetStringSlice("devices"))
		fmt.Fprintf(webprint, "Devices:\n")
		for _, v := range viper.GetStringSlice("devices") {
			fmt.Printf("value[%s]\n", v)
			fmt.Fprintf(webprint, "v=%s\n", v)
		}

		for _, v := range viper.GetStringSlice("devices") {
			fmt.Printf("v=%s time=%s\n", v, mapSteve[v].String())
		}

	}

}

func displayHelp() {
	helpmessage :=
		`
  --config [config file]             Configuration file: /path/to/file.yaml, default = ./config.yaml
  --displayconfig                    Display configuration
  --help                             Display help
  --version                          Display version
`
	fmt.Printf("%s", helpmessage)
}

func displayConfig() {
	fmt.Println("CONFIG: file :", viper.ConfigFileUsed())
	allmysettings := viper.AllSettings()
	var keys []string
	for k := range allmysettings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Println("CONFIG:", k, ":", allmysettings[k])
	}
}
