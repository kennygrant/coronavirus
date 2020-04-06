package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kennygrant/coronavirus/series"
)

// ScheduleUpdates schedules data updates from our data sources
// after each update data series is resaved and a data reload triggered
// the changes are also committed to the git repository
func ScheduleUpdates() {
	log.Printf("series: scheduling updates")

	// Call update frequent immediately on load to start loading data for todaay
	go updateFrequent()

	// Schedule calls daily and every 10 mins to update data
	now := time.Now()

	when := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 5, 0, time.UTC)
	regularly := 15 * time.Minute // every 15 minutes
	ScheduleAt(updateFrequent, when, regularly)

	when = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 1, 0, time.UTC)
	daily := time.Hour * 24 // daily
	ScheduleAt(updateDaily, when, daily)

}

// updateDaily adds a new day to all of our series for today (based on yesterday's figures)
// it should be run just after UTC zero hours
func updateDaily() {
	log.Printf("update: updating daily at:%s", time.Now().UTC())

	// Update the series to add today
	err := series.AddToday()
	if err != nil {
		log.Printf("update: failed to add today to series:%s", err)
		return
	}

}

// I think for manual updates just edit files and hit the reload endpont

// updateFrequent updates data frequently (every 30 minutes say)
// Called in a goroutine
// the dataset is somewhat inconsistent and therefore requires some massaging
// for example not all countries have global data
// NB after non-essential failures we just continue rather than log an error
// some datasources may be down for example, but some not
func updateFrequent() {

	// Pull the repo first with git to be sure we're up to date
	err := gitPull()
	if err != nil {
		log.Printf("update: failed to pull repo:%s", err)
	}

	err = updateUKCases()
	if err != nil {
		log.Printf("update: UK FAILED:%s", err)
	}

	err = updateJHUCases()
	if err != nil {
		log.Printf("update: JHU FAILED:%s", err)
	}

	// Now update our global series which are unfortunteley not contained in this data
	err = series.CalculateGlobalSeriesData()
	if err != nil {
		log.Printf("update: failed to calculate global series :%s", err)
		return
	}

	// Now save the series file to disk
	err = series.Save("data/series.csv")
	if err != nil {
		log.Printf("server: failed to save series data:%s", err)
		return
	}

	// Finally attempt to commit the change to the report with a suitable commit message
	message := fmt.Sprintf("Updated from external data for %s", time.Now().UTC().Format("2006-01-02"))
	err = gitCommit(message)
	if err != nil {
		log.Printf("server: failed to commit change:%s", err)
	}

}

// Update UK stats linked from gov.uk
// https://www.gov.uk/guidance/coronavirus-covid-19-information-for-the-public#number-of-cases-and-deaths
func updateUKCases() error {

	filePath := "https://services1.arcgis.com/0IrmI40n5ZYxTUrV/arcgis/rest/services/DailyIndicators/FeatureServer/0/query?where=TotalUKCases%3E0&objectIds=&time=&resultType=standard&outFields=*&returnIdsOnly=false&returnUniqueIdsOnly=false&returnCountOnly=false&returnDistinctValues=false&cacheHint=false&orderByFields=&groupByFieldsForStatistics=&outStatistics=&having=&resultOffset=&resultRecordCount=&sqlFormat=none&f=pgeojson&token="

	jsonData, err := downloadJSON(filePath)
	if err != nil {
		return fmt.Errorf("server: failed to download UK json:%s", err)
	}

	err = series.UpdateFromUKStats(jsonData)
	if err != nil {
		return fmt.Errorf("server: failed to parse UK json:%s", err)
	}

	return nil
}

func updateJHUCases() error {

	// Download the country cases file into csv rows
	filePath := "https://raw.githubusercontent.com/CSSEGISandData/COVID-19/web-data/data/cases_country.csv"
	rows, err := downloadCSV(filePath)
	if err != nil {
		return fmt.Errorf("server: failed to download JHU csv:%s", err)
	}

	// This data has a specific format, ask the series to decode
	// and update the changed series in memory
	err = series.UpdateFromJHUCountryCases(rows)
	if err != nil {
		return fmt.Errorf("server: failed to update from JHU data :%s", err)
	}

	// Update from the cases_states file for US states data

	// Download the US states cases file into csv rows
	filePath = "https://raw.githubusercontent.com/CSSEGISandData/COVID-19/web-data/data/cases_state.csv"
	rows, err = downloadCSV(filePath)
	if err != nil {
		return fmt.Errorf("server: failed to download JHU states csv:%s", err)
	}

	// This data has a specific format, ask the series to decode
	// and update the changed series in memory
	err = series.UpdateFromJHUStatesCases(rows)
	if err != nil {
		return fmt.Errorf("server: failed to update from JHU data :%s", err)
	}
	return nil

}

// downloadJSON downloads and parses the url as generic json
func downloadJSON(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	jsonData := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&jsonData)
	return jsonData, err
}

// downloadCSV downloads and parses the url as csv
func downloadCSV(url string) ([][]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// gitPull runs a git pull command
func gitPull() error {

	return nil
}

// gitCommit runs a git commit command
func gitCommit(message string) error {

	return nil
}

// ScheduleAt schedules execution for a particular time and at intervals thereafter.
// If interval is 0, the function will be called only once.
// Callers should call close(task) before exiting the app or to stop repeating the action.
func ScheduleAt(f func(), t time.Time, i time.Duration) chan struct{} {
	task := make(chan struct{})
	now := time.Now().UTC()

	// Check that t is not in the past, if it is increment it by interval until it is not
	for now.Sub(t) > 0 {
		t = t.Add(i)
	}

	// We ignore the timer returned by AfterFunc - so no cancelling, perhaps rethink this
	tillTime := t.Sub(now)
	time.AfterFunc(tillTime, func() {
		// Call f at the time specified
		go f()

		// If we have an interval, call it again repeatedly after interval
		// stopping if the caller calls stop(task) on returned channel
		if i > 0 {
			ticker := time.NewTicker(i)
			go func() {
				for {
					select {
					case <-ticker.C:
						go f()
					case <-task:
						ticker.Stop()
						return
					}
				}
			}()
		}
	})

	return task // call close(task) to stop executing the task for repeated tasks
}
