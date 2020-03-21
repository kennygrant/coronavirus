package covid

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Data types for imported series data
const (
	DataDeaths = iota
	DataConfirmed
	DataRecovered
)

// Series stores data for one country or province within a country
type Series struct {
	// The Country or Region
	Country string
	// The Province or State - may be blank for countries
	Province string
	// The date at which the series starts
	StartsAt time.Time
	// Total Deaths, Confirmed or Recovered by day (cumulative)
	Deaths    []int
	Confirmed []int
	Recovered []int
}

// Title returns a display title for this series
func (s *Series) Title() string {
	if s.Country == "" && s.Province == "" {
		return "Global"
	} else if s.Province == "" {
		return s.Country
	}

	return fmt.Sprintf("%s > %s", s.Country, s.Province)
}

// FetchDate retusn the data for the given data from datum
func (s *Series) FetchDate(datum int, date time.Time) int {

	// Calculate index in series given StartsAt
	days := date.Sub(s.StartsAt)
	i := int(days.Hours() / 24)

	// Bounds check index
	if i < 0 || i > len(s.Deaths)-1 {
		return 0
	}

	// Fetch the data at index
	switch datum {
	case DataDeaths:
		return s.Deaths[i]
	case DataConfirmed:
		return s.Confirmed[i]
	case DataRecovered:
		return s.Recovered[i]
	}
	return 0
}

// Valid returns true if this series is valid
// a series without a start date set is considered invalid
func (s *Series) Valid() bool {
	return !s.StartsAt.IsZero()
}

// Match returns true if this series matches data from a row
// performs a case insensitive match
func (s *Series) Match(country string, province string) bool {
	return strings.ToLower(s.Country) == strings.ToLower(country) && strings.ToLower(s.Province) == strings.ToLower(province)
}

// Merge the data from the incoming series with ours
func (s *Series) Merge(series *Series) {
	// If we are len 0 just replace series
	if len(s.Deaths) == 0 {
		s.Deaths = series.Deaths
		s.Confirmed = series.Confirmed
		s.Recovered = series.Recovered
		return // replace and return
	}

	// Else we have data already, so add to it
	// we assume the same number of dates for all series
	for i, d := range series.Deaths {
		s.Deaths[i] += d
	}
	for i, d := range series.Confirmed {
		s.Confirmed[i] += d
	}
	for i, d := range series.Recovered {
		s.Recovered[i] += d
	}
}

// TotalDeaths returns the cumulative death due to COVID-19 for this series
// (the last entry)
func (s *Series) TotalDeaths() int {
	return s.Deaths[len(s.Deaths)-1]
}

// TotalConfirmed returns the cumulative confirmed cases of COVID-19 for this series
// (the last entry)
func (s *Series) TotalConfirmed() int {
	return s.Confirmed[len(s.Confirmed)-1]
}

// TotalRecovered returns the cumulative confirmed cases of COVID-19 for this series
// (the last entry)
func (s *Series) TotalRecovered() int {
	return s.Recovered[len(s.Recovered)-1]
}

// SLICE OF Series

// SeriesSlice is a collection of Series
type SeriesSlice []*Series

// FetchDate fetches the datapiont for a given datum and date
func (slice SeriesSlice) FetchDate(country, province string, datum int, date time.Time) (int, error) {
	// Find the series, if none found return 0
	series, err := slice.FetchSeries(country, province)
	if err != nil {
		return 0, err
	}
	if !series.Valid() {
		return 0, fmt.Errorf("series: no such series")
	}

	return series.FetchDate(datum, date), nil
}

// FetchSeries returns a series (if found) for this combination of country and province
func (slice SeriesSlice) FetchSeries(country string, province string) (*Series, error) {

	for _, s := range slice {
		if s.Match(country, province) {
			return s, nil
		}
	}

	return &Series{}, fmt.Errorf("series: not found")
}

// MergeCSV merges the data in this CSV with data
func (slice SeriesSlice) MergeCSV(records [][]string, dataType int) (SeriesSlice, error) {

	// Make an assumption about the starting date (checked below on header row)
	startDate := time.Date(2020, 1, 22, 0, 0, 0, 0, time.UTC)

	for i, row := range records {
		// Check header to see this is the file we expect, if not skip
		if i == 0 {
			// We just check a few cols - we assume the start date of the data won't change
			if row[0] != "Province/State" || row[1] != "Country/Region" || row[2] != "Lat" || row[4] != "1/22/20" {
				return slice, fmt.Errorf("load: error loading file - csv data format invalid")
			}

		} else {

			// Fetch data to match series
			country := row[1]
			province := row[0]

			// We ignore rows which match ,CA etc
			// these are US sub-state level data which is no longer included in the dataset and is zeroed out
			if country == "US" && strings.Contains(province, ", ") {
				fmt.Printf("ignoring series:%s %s\n", country, province)
				continue
			}

			// Fetch the series
			var series *Series
			series, _ = slice.FetchSeries(country, province)

			// If we don't have one yet, create one
			if !series.Valid() {
				series = &Series{
					Country:  country,
					Province: province,
					StartsAt: startDate,
				}
				slice = append(slice, series)
			}

			// Walk through row, reading days data after col 3 (longitude)
			for i, d := range row {
				if i < 4 {
					continue
				}
				v, err := strconv.Atoi(d)
				if err != nil {
					return slice, fmt.Errorf("load: error loading row %d - csv day data invalid:%s", i, err)
				}
				switch dataType {
				case DataDeaths:
					series.Deaths = append(series.Deaths, v)
				case DataConfirmed:
					series.Confirmed = append(series.Confirmed, v)
				case DataRecovered:
					series.Recovered = append(series.Recovered, v)
				}
			}

		}

	}
	return slice, nil
}
