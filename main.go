package main

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/kennygrant/coronavirus/series"
)

var development = false

// Store our templates globally, don't touch them after server start
var htmlTemplate *template.Template
var jsonTemplate *template.Template

// Main loads data, sets up a periodic fetch, and starts a web server to serve that data
func main() {

	if os.Getenv("COVID") == "dev" {
		development = true
	}

	if development {
		log.Printf("server: starting in development mode")
	} else {
		log.Printf("server: starting in production mode")
	}

	// Load our data
	err := series.LoadData("./data")
	if err != nil {
		log.Fatalf("server: failed to load new data:%s", err)
	}

	// Schedule a regular data update/reload - don't bother in development except when testing
	if !development {
		ScheduleUpdates()
	}

	// Load our template files into memory
	loadTemplates()

	// Set up the https server with the handler attached to serve this data in a template
	http.HandleFunc("/favicon.ico", handleFile)
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/reload", handleReload)

	// Start a server on port 443 (or another port if dev specified)
	if development {
		// In development just serve with http on local port 3000
		// reload templates on each page load
		err := http.ListenAndServe(":3000", nil)
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
		"e":  escapeJSON,
		"l":  outputList,
		"ls": outputStringList,
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
	country, province, period, startDeaths := parseParams(r)

	// Fetch the series concerned - if both are blank we'll get the global series
	s, err := series.FetchSeries(country, province)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Get the total counts first for the page
	allTimeDeaths := s.TotalDeaths()
	allTimeConfirmed := s.TotalConfirmed()
	allTimeRecovered := s.TotalRecovered() // unreliable as yet
	allTimeTested := s.TotalTested()

	mobile := strings.Contains(strings.ToLower(r.UserAgent()), "mobile")

	if startDeaths == 0 {
		startDeaths = 5
	}

	// Use a default period depending on device if none selected
	if period == 0 {
		// Default to last 56 days
		period = 56

		// Default to 28 days later for phones
		if mobile {
			period = 28
		}
	}

	// Limit by period if applied
	if period > 0 {
		s = s.Period(period)
	}

	scale := "linear"
	scaleURL := r.URL.Path + "?scale=log#growth"
	if param(r, "scale") == "log" {
		scale = "logarithmic"
		scaleURL = r.URL.Path + "#growth"
	}

	// For global compare growth rate of top 20 series
	var comparisons series.Slice
	if s.IsGlobal() {
		comparisons = series.TopSeries(country, 10)
	} else if s.IsEuropean() {
		comparisons = series.SelectedEuropeanSeries(country, 10)
	} else if s.HasProvinces() {
		log.Printf("home: comparing provinces for:%s", country)
		comparisons = series.TopSeries(country, 10)
	} else {
		// Else fetch a selection of copmarative series (for example nearby countries)
		comparisons = series.SelectedSeries(country, 10)
	}

	log.Printf("comparisons:%d", len(comparisons))

	// Set up context with data
	context := map[string]interface{}{
		"period":           strconv.Itoa(period),
		"country":          s.Key(s.Country),
		"province":         s.Key(s.Province),
		"comparisons":      comparisons,
		"series":           s,
		"allTimeDeaths":    allTimeDeaths,
		"allTimeConfirmed": allTimeConfirmed,
		"allTimeRecovered": allTimeRecovered,
		"allTimeTested":    allTimeTested,
		"periodOptions":    series.PeriodOptions(),
		"countryOptions":   series.CountryOptions(),
		"provinceOptions":  series.ProvinceOptions(s.Country),
		"jsonURL":          fmt.Sprintf("%s.json?period=%d", r.URL.Path, period),
		"scale":            scale,
		"scaleURL":         scaleURL,
		"mobile":           mobile,
		"startDeaths":      startDeaths, // Deaths to start comparison chart from
	}

	// If in development reload templates each time - no mutex as in dev only
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

// handleReload
// FIXME - require authentication to avoid DOS
func handleReload(w http.ResponseWriter, r *http.Request) {

	log.Printf("reload:%s", r.URL)

	err := series.LoadData("./data")

	// Check for errors on reload
	if err != nil {
		log.Printf("reload error:%s", err)
		http.Error(w, err.Error(), 500)
	} else {
		http.Redirect(w, r, "/", 302)
	}

	// Also reload today from online data
	go updateFrequent()
}

// param returns one param string value
func param(r *http.Request, key string) string {
	queryParams := r.URL.Query()
	if len(queryParams[key]) > 0 {
		return queryParams[key][0]
	}

	return ""
}

// parseParams parses the parts of the url path (if any) and params
func parseParams(r *http.Request) (country, province string, period, startDeaths int) {

	var err error

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

	// Read period if any
	if len(queryParams["period"]) > 0 {
		paramString := queryParams["period"][0]
		period, err = strconv.Atoi(paramString)
		if err != nil {
			period = 0
		}
	}

	// Read start deaths (to start charts at death n)
	if len(queryParams["start_deaths"]) > 0 {
		paramString := queryParams["start_deaths"][0]
		startDeaths, err = strconv.Atoi(paramString)
		if err != nil {
			startDeaths = 0
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

	return country, province, period, startDeaths
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

// outputList outputs a comma separated list with no trailing comma
func outputList(ints []int) template.HTML {
	result := ""
	for i, v := range ints {
		if i == 0 {
			result = fmt.Sprintf("%d", v)
		} else {
			result = fmt.Sprintf("%s, %d", result, v)
		}
	}
	return template.HTML("[" + result + "]")
}

// outputList outputs a comma separated list with no trailing comma
func outputStringList(strings []string) template.HTML {
	result := ""
	for i, v := range strings {
		if i == 0 {
			result = fmt.Sprintf("\"%s\"", v)
		} else {
			result = fmt.Sprintf("%s, \"%s\"", result, v)
		}
	}
	return template.HTML("[" + result + "]")
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
