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

const APPVERSION = "0.0.9"

var allDevices = make(map[string]time.Time)

var CHECKINTOKEN string
var STATUSTOKEN string
var LISTENIP string
var LISTENPORT string
var INDEXHTML string

func init() {
	fmt.Println("Simple-canary v" + APPVERSION)
	flag.Bool("help", false, "Display help")
	flag.String("config", "config.yaml", "Configuration file: /path/to/file.yaml, default = ./config.yaml")
	flag.Bool("version", false, "Display version")
	flag.Bool("verbose", false, "Be verbose")
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

	// assign configuration loaded from file to global variables
	CHECKINTOKEN = viper.GetString("checkintoken")
	STATUSTOKEN = viper.GetString("statustoken")
	LISTENIP = viper.GetString("listenip")
	LISTENPORT = viper.GetString("listenport")
	INDEXHTML = viper.GetString("indexhtml")

	// configure all devices to have a "zero" time
	for _, v := range viper.GetStringSlice("devices") {
		allDevices[strings.ToLower(v)] = time.Time{}
	}

	// display configuration
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

	// enable verbose logging
	if viper.GetBool("verbose") {
		r.Use(loggingMiddleware)
	}

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

	fmt.Fprintf(webprint, "%s", string(texttoprint))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("MIDDLEWARE: ", r.RemoteAddr, " ", r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func handlerIndex(w http.ResponseWriter, r *http.Request) {
	printFile(INDEXHTML, w)
}

func handlerCheckin(webprint http.ResponseWriter, r *http.Request) {
	// check if checkintoken is valid
	queries := r.URL.Query()
	if CHECKINTOKEN != queries.Get("token") {
		webprint.WriteHeader(http.StatusUnauthorized)
		log.Printf("Error:Invalid Checkin Token Received")
		fmt.Fprintf(webprint, "%s", "ERROR: Invalid Checkin Token")
		return
	}

	vars := mux.Vars(r)

	if len(vars["device"]) > 0 {
		if _, ok := allDevices[strings.ToLower(vars["device"])]; ok {
			webprint.WriteHeader(http.StatusOK)
			allDevices[strings.ToLower(vars["device"])] = time.Now()
			fmt.Fprintf(webprint, "Device=%s\nLastCheckinTime=%s", strings.ToLower(vars["device"]), allDevices[strings.ToLower(vars["device"])])
		} else {
			webprint.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(webprint, "Device doesn't exist")
		}
	}

}

func handlerStatus(webprint http.ResponseWriter, r *http.Request) {
	// check if statustoken is valid
	if viper.GetBool("statustokencheck") {
		queries := r.URL.Query()
		if STATUSTOKEN != queries.Get("token") {
			webprint.WriteHeader(http.StatusUnauthorized)
			log.Printf("Error:Invalid Status Token Received")
			fmt.Fprintf(webprint, "%s", "ERROR: Invalid Status Token")
			return
		}
	}

	vars := mux.Vars(r)

	if len(vars["device"]) > 0 {
		if value, ok := allDevices[strings.ToLower(vars["device"])]; ok {
			webprint.WriteHeader(http.StatusOK)
			// fmt.Fprintf(webprint, "Device=%s\nLastCheckinTime=%s", strings.ToLower(vars["device"]), allDevices[strings.ToLower(vars["device"])])
			//fmt.Fprintf(webprint, "Device=%s\nLastCheckinTime=%s State=%s", strings.ToLower(vars["device"]), value, checkTTL(value))
			fmt.Fprintf(webprint, "%s", checkTTL(value))
		} else {
			webprint.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(webprint, "Device doesn't exist")
		}
	} else {
		webprint.WriteHeader(http.StatusOK)
		/*
			for _, value := range viper.GetStringSlice("devices") {
				fmt.Fprintf(webprint, "Device=%s\nLastCheckinTime=%s SecondsSinceLastCheckin=%s State=%s\n\n", strings.ToLower(value), allDevices[strings.ToLower(value)], timeSinceLastCheckin(allDevices[strings.ToLower(value)]), checkTTL(allDevices[strings.ToLower(value)]))
			}
		*/

		var template = "      <tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n"

		header := `<!DOCTYPE HTML>
<html>
<head>
  <title>Status Page</title>
  <link rel="stylesheet" href="https://unpkg.com/purecss@2.0.6/build/pure-min.css" integrity="sha384-Uu6IeWbM+gzNVXJcM9XV3SohHtmWE+3VGi496jvgX1jyvDTXfdK+rfZc8C1Aehk5" crossorigin="anonymous">
</head>
<body>
`

		footer := `</body>
</html>`

		fmt.Fprintf(webprint, header)
		fmt.Fprintf(webprint, "  <table class=\"pure-table pure-table-bordered\">\n")
		fmt.Fprintf(webprint, "    <thead><tr><th>Device</th><th>Last Checkin</th><th>Seconds Since Checkin</th><th>State</th></tr></thead>\n")
		fmt.Fprintf(webprint, "    <tbody>\n")
		datelayout := "Mon Jan _2 15:04:05 MST 2006"

		var lastcheckindate string
		var timesincelastcheckin string
		var status string

		for _, value := range viper.GetStringSlice("devices") {

			if allDevices[strings.ToLower(value)].IsZero() {
				lastcheckindate = "Never"
				timesincelastcheckin = "Never"
				status = "Offline"
			} else {
				lastcheckindate = allDevices[strings.ToLower(value)].Format(datelayout)
				timesincelastcheckin = timeSinceLastCheckin(allDevices[strings.ToLower(value)])
				status = checkTTL(allDevices[strings.ToLower(value)])
			}

			fmt.Fprintf(webprint, template, strings.ToLower(value), lastcheckindate, timesincelastcheckin, status)

		}
		fmt.Fprintf(webprint, "    </tbody>\n")
		fmt.Fprintf(webprint, "  </table>\n")
		fmt.Fprintf(webprint, footer)

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

func checkTTL(checkthis time.Time) string {
	if time.Now().Sub(checkthis).Seconds() >= float64(viper.GetInt("ttl")) {
		return "Offline"
	} else {
		return "Online"
	}
}

func timeSinceLastCheckin(checkthis time.Time) string {
	temptime := time.Now().Sub(checkthis)
	tempseconds := fmt.Sprintf("%.0f", temptime.Seconds())
	return tempseconds
}
