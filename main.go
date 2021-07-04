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

const APPVERSION = "0.0.3"
const CHECKINTOKEN = "sometoken"
const STATUSTOKEN = "statustoken"
const LISTENIP = "0.0.0.0"
const LISTENPORT = "54034"
const INDEXHTML = "index.html"

var allDevices = make(map[string]time.Time)

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

	// configure all devices to have a zero time
	for _, v := range viper.GetStringSlice("devices") {
		allDevices[strings.ToLower(v)] = time.Time{}
	}

	if viper.GetBool("displayconfig") {
		displayConfig()
		os.Exit(0)
	}

}

func main() {
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

	fmt.Println("Cannot start http server:", err)
}

func printFile(filename string, webprint http.ResponseWriter) {
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
	if CHECKINTOKEN != queries.Get("token") {
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
		if STATUSTOKEN != queries.Get("token") {
			fmt.Println("ERROR: Invalid API Token", queries.Get("token"))
			fmt.Fprintf(webprint, "%s", "ERROR: Invalid API Token")
			return
		}
	*/

	vars := mux.Vars(r)

	webprint.WriteHeader(http.StatusOK)

	if len(vars["device"]) > 0 {
		if value, ok := allDevices[strings.ToLower(vars["device"])]; ok {
			// fmt.Fprintf(webprint, "Device=%s\nLastCheckinTime=%s", strings.ToLower(vars["device"]), allDevices[strings.ToLower(vars["device"])])
			fmt.Fprintf(webprint, "Device=%s\nLastCheckinTime=%s", strings.ToLower(vars["device"]), value)
		} else {
			fmt.Fprintf(webprint, "Device doesn't exist")
		}
	} else {
		for _, v := range viper.GetStringSlice("devices") {
			fmt.Fprintf(webprint, "Device=%s\nLastCheckinTime=%s\n\n", strings.ToLower(v), allDevices[strings.ToLower(v)])
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
