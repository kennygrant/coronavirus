package main

import (
	"crypto/tls"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kennygrant/coronavirus/covid"
	"golang.org/x/crypto/acme/autocert"
)

var development = false

// Store our templates globally, don't touch it after server start
var htmlTemplate *template.Template
var jsonTemplate *template.Template

// Main loads data, sets up a periodic fetch, and starts a web server to serve that data
func main() {

	if os.Getenv("COVID") == "dev" {
		development = true
	}

	if development {
		log.Printf("server: restarting in development mode")
	} else {
		log.Printf("server: restarting")
	}

	// Schedule a regular fetch of data at a specified time daily
	covid.ScheduleDataFetch()

	/*
		// For testing, test a fetch instead
		err := covid.FetchData()
		if err != nil {
			log.Fatalf("server: failed to load data:%s", err)
		}
	*/

	// Load the data
	err := covid.LoadData()
	if err != nil {
		log.Fatalf("server: failed to load data:%s", err)
	}

	// Load our template files into memory
	loadTemplates()

	// Set up the https server with the handler attached to serve this data in a template
	http.HandleFunc("/favicon.ico", handleFile)
	http.HandleFunc("/", handleHome)

	// Start a server on port 443 (or another port if dev specified)
	if development {
		// In development just serve with http on local port 3000
		err = http.ListenAndServe(":3000", nil)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		domains := []string{"coronavirus.projectpage.app"}
		StartTLSServer(development, domains)
	}

}

func loadTemplates() {
	var err error
	htmlTemplate, err = template.ParseFiles("index.html.got")
	if err != nil {
		log.Fatalf("template error:%s", err)
	}
	funcMap := map[string]interface{}{
		"e": escapeJSON,
	}
	jsonTemplate, err = template.New("index.json.got").Funcs(funcMap).ParseFiles("index.json.got")
	if err != nil {
		log.Fatalf("template error:%s", err)
	}
}

// handleHome shows our website
func handleHome(w http.ResponseWriter, r *http.Request) {

	log.Printf("request:%s", r.URL)

	// Get the parameters from the url
	country, province, period := parseParams(r)

	// Fetch the series concerned - if both are blank we'll get the global series
	series, err := covid.FetchSeries(country, province)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Limit by period if necessary
	if period > 0 {
		series = series.Days(period)
	}

	//log.Printf("request:%s country:%s province:%s period:%d", r.URL, country, province, period)

	// Set up context with data
	context := map[string]interface{}{
		"period":          strconv.Itoa(period),
		"country":         series.Key(series.Country),
		"province":        series.Key(series.Province),
		"series":          series,
		"periodOptions":   covid.PeriodOptions(),
		"countryOptions":  covid.CountryOptions(),
		"provinceOptions": covid.ProvinceOptions(series.Country),
	}

	// If in development reload templates each time
	if development {
		loadTemplates()
	}

	// Render the template, either html or json
	if strings.HasSuffix(r.URL.Path, ".json") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(200)
		err = jsonTemplate.Execute(w, context)
	} else {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(200)
		err = htmlTemplate.Execute(w, context)
	}

	// Check for errors on render
	if err != nil {
		log.Printf("template render error:%s", err)
		http.Error(w, err.Error(), 500)
	}

}

// parseParams parses the parts of the url path (if any) and params
func parseParams(r *http.Request) (country, province string, period int) {

	// Parse the path
	p := r.URL.Path

	// First remove .json if it exists
	p = strings.Replace(p, ".json", "", 1)

	// Now parse parts
	parts := strings.Split(strings.Trim(p, "/"), "/")

	if len(parts) > 0 {
		country = parts[0]
	}
	if len(parts) > 1 {
		province = parts[1]
	}

	// Add query string params from request  - accept all params this way
	queryParams := r.URL.Query()
	if len(queryParams["period"]) > 0 {
		var err error
		periodString := queryParams["period"][0]
		period, err = strconv.Atoi(periodString)
		if err != nil {
			period = 0
		}
	}
	if len(queryParams["country"]) > 0 {
		country = queryParams["country"][0]
	}

	if len(queryParams["province"]) > 0 {
		province = queryParams["province"][0]
	}

	// Allow some abreviations for urls
	if country == "uk" {
		country = "United Kingdom"
	}

	// Allow global for top level (for global.json)
	if country == "global" {
		country = ""
	}

	return country, province, period
}

// handleFile shows a file (if it exists)
func handleFile(w http.ResponseWriter, r *http.Request) {

	// Serve the local path
	localPath := "./public" + filepath.Clean(r.URL.Path)

	// Check it exists
	_, err := os.Stat(localPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Try to serve the file
	//log.Printf("file:%s", localPath)
	http.ServeFile(w, r, localPath)
}

// JSON escape function
func escapeJSON(t string) template.HTML {
	// Escape mandatory characters
	t = strings.Replace(t, "\r", " ", -1)
	t = strings.Replace(t, "\n", " ", -1)
	t = strings.Replace(t, "\t", " ", -1)
	t = strings.Replace(t, "\\", "\\\\", -1)
	t = strings.Replace(t, "\"", "\\\"", -1)
	// Because we use html/template escape as temlate.HTML
	return template.HTML(t)
}

// StartTLSServer starts a TLS server using lets encrypt
func StartTLSServer(dev bool, domains []string) {
	certManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Email:      "",                                 // Email for problems with certs
		HostPolicy: autocert.HostWhitelist(domains...), // Domains to request certs for
		Cache:      autocert.DirCache("secrets"),       // Cache certs in secrets folder
	}

	server := &http.Server{
		// Set the port in the preferred string format
		Addr: ":443",

		// The default server from net/http has no timeouts - set some limits
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       10 * time.Second, // IdleTimeout was introduced in Go 1.8

		// This TLS config follows recommendations in the above article
		TLSConfig: &tls.Config{
			// Pass in a cert manager if you want one set
			// this will only be used if the server Certificates are empty
			GetCertificate: certManager.GetCertificate,

			// VersionTLS11 or VersionTLS12 would exclude many browsers
			// inc. Android 4.x, IE 10, Opera 12.17, Safari 6
			// So unfortunately not acceptable as a default yet
			// Current default here for clarity
			MinVersion: tls.VersionTLS10,

			// Causes servers to use Go's default ciphersuite preferences,
			// which are tuned to avoid attacks. Does nothing on clients.
			PreferServerCipherSuites: true,
			// Only use curves which have assembly implementations
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519, // Go 1.8 only
			},
		},
	}

	// Handle all :80 traffic using autocert to allow http-01 challenge responses
	go func() {
		http.ListenAndServe(":80", certManager.HTTPHandler(nil))
	}()

	err := server.ListenAndServeTLS("", "")
	if err != nil {
		log.Printf("error: starting server %s", err)
	}
}
