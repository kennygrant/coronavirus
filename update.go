package main

import (
	"encoding/csv"
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

	// As a test just try calling update first
	go updateFrequent()
}

// I think for manual updates just hit the reload endpont

// updateFrequent updaes data frequently (every 30 minutes say)
// Called in a goroutine
// the dataset is somewhat inconsistent and therefore requires some massaging
// for example not all countries have global data
func updateFrequent() {

	// Pull the repo first with git to be sure we're up to date
	err := gitPull()
	if err != nil {
		log.Printf("server: failed to pull repo:%s", err)
		return
	}

	// Download the country cases file into csv rows
	filePath := "https://raw.githubusercontent.com/CSSEGISandData/COVID-19/web-data/data/cases_country.csv"
	rows, err := downloadCSV(filePath)
	if err != nil {
		log.Printf("server: failed to download JHU csv:%s", err)
		return
	}

	// This data has a specific format, ask the series to decode
	// and update the changed series in memory
	err = series.UpdateFromJHUCountryCases(rows)
	if err != nil {
		log.Printf("server: failed to update from JHU data :%s", err)
		return
	}

	// Update from the cases_states file for US states data

	// Download the US states cases file into csv rows
	filePath = "https://raw.githubusercontent.com/CSSEGISandData/COVID-19/web-data/data/cases_state.csv"
	rows, err = downloadCSV(filePath)
	if err != nil {
		log.Printf("server: failed to download JHU states csv:%s", err)
		return
	}

	// This data has a specific format, ask the series to decode
	// and update the changed series in memory
	err = series.UpdateFromJHUStatesCases(rows)
	if err != nil {
		log.Printf("server: failed to update from JHU data :%s", err)
		return
	}

	// Now update our global series which are unfortunteley not contained in this data
	err = series.CalculateGlobalSeriesData()
	if err != nil {
		log.Printf("server: failed to calculate global series :%s", err)
		return
	}

	/*
		// Now save the series file to disk
		err = series.Save("data/series.csv")
		if err != nil {
			log.Printf("server: failed to save series data:%s", err)
			return
		}

		// Finally attempt to commit the change to the report with a suitable commit message
		err = gitCommit("Updated from JHU data")
		if err != nil {
			log.Printf("server: failed to commit change:%s", err)
			return
		}*/

}

//
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
