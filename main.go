package main

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/kennygrant/coronavirus/covid"
)

// Store our templates globally, don't touch it after server start
var htmlTemplate *template.Template
var jsonTemplate *template.Template

// Main loads data, sets up a periodic fetch, and starts a web server to serve that data
func main() {
	log.Printf("server: restarting")

	// Schedule a regular fetch of data at a specified time daily
	covid.ScheduleDataFetch()

	// Load the data
	err := covid.LoadData()
	if err != nil {
		log.Fatalf("server: failed to load data:%s", err)
	}

	// Load our template files into memory
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

	// Set up the https server with the handler attached to serve this data in a template
	http.HandleFunc("/favicon.ico", handleFile)
	http.HandleFunc("/", handleHome)
	err = http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

// handleHome shows our website
func handleHome(w http.ResponseWriter, r *http.Request) {

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

	log.Printf("request:%s country:%s province:%s period:%d", r.URL, country, province, period)

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

	// Get the URL path
	p := path.Clean(r.URL.Path)

	log.Printf("file:%s", p)

	// Construct a local path

	// For now just return not found
	http.NotFound(w, r)

	/*
		if r.URL.Path == "/favicon.ico" {
			http.NotFound(w, r)
			return
		}
	*/
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
