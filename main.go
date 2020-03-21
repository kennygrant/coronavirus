package main

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/kennygrant/covid-19/covid"
)

// Store our data globally, don't touch it after server start
var data covid.SeriesSlice

// Store our template globally, don't touch it after server start
//var template html.Template

// Main loads data, sets up a periodic fetch, and starts a web server to serve that data
func main() {
	log.Printf("server: restarting")

	// TODO - Set up a fetch to get the latest data later on

	// Load the data
	var err error
	start := time.Now()
	data, err = covid.LoadData("./data")
	if err != nil {
		log.Fatalf("server: failed to load data:%s", err)
	}

	log.Printf("server: loaded data in %s len:%d", time.Now().Sub(start), len(data))

	// Load our one template file for now into memory

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
	// Log the request
	log.Printf("request:%s", r.URL)

	// Get the parameters from the url
	country, province := parseURL(r.URL.Path)

	// Fetch the series concerned - if both are blank we'll get the global series
	series, err := data.FetchSeries(country, province)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	log.Printf("request: country:%s province:%s", country, province)

	// Read the template from our local file and render
	tmpl, err := template.ParseFiles("layout.html.got")
	if err != nil {
		log.Printf("template error:%s", err)
		http.Error(w, err.Error(), 500)
	}

	// Set up context with data
	context := map[string]interface{}{
		"series":          series,
		"countryOptions":  data.CountryOptions(),
		"provinceOptions": data.ProvinceOptions(""),
	}

	// Render the template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	err = tmpl.Execute(w, context)
	if err != nil {
		log.Printf("template render error:%s", err)
		http.Error(w, err.Error(), 500)
	}

}

// parseURL parses the parts of the url path (if any)
// it may return two empty strings if there is no path
func parseURL(path string) (country, province string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) > 0 {
		country = parts[0]
	}
	if len(parts) > 1 {
		province = parts[1]
	}

	// Allow some abreviations for urls
	if country == "uk" {
		country = "United Kingdom"
	}

	return country, province
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
