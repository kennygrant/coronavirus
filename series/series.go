package series

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// seriesStartDate is our default start date
var seriesStartDate = time.Date(2020, 1, 22, 0, 0, 0, 0, time.UTC)

// NewData returns a new Data series based on the row values
// We expect the cols
func NewData(row []string) (*Data, error) {

	country := row[0]
	province := row[1]
	latitude, err := strconv.ParseFloat(row[2], 64)
	if err != nil {
		return nil, fmt.Errorf("areas: invalid latitude at row:%s", row)
	}
	longitude, err := strconv.ParseFloat(row[3], 64)
	if err != nil {
		return nil, fmt.Errorf("areas: invalid longitude at row:%s", row)
	}
	population, err := strconv.Atoi(row[4])
	if err != nil {
		return nil, fmt.Errorf("areas: invalid population at row:%s", row)
	}

	var lockdown time.Time
	if row[5] != "" {
		lockdown, err = time.Parse("2006-01-02", row[5])
		if err != nil {
			return nil, fmt.Errorf("areas: invalid lockdown at row:%s", row)
		}
	}

	color := row[6]

	// NB updated at left at zero time
	s := &Data{
		Country:    country,
		Province:   province,
		Latitude:   latitude,
		Longitude:  longitude,
		Population: population,
		Color:      color,
		LockdownAt: lockdown,
		Days:       make([]*Day, 0),
	}

	return s, nil
}

// Data stores data for one country or province within a country
type Data struct {

	// The Country or Region
	Country string

	// The Province or State - blank for countries
	Province string

	// The population of the area (if known)
	Population int

	// Coordinates for this area
	Latitude, Longitude float64

	// An rgb color/colour for plotting charts
	Color string

	// UTC Date data last updated
	UpdatedAt time.Time

	// UTC Date full area lockdown started
	LockdownAt time.Time

	// Days containing all our data - each day holds cumulative totals
	Days []*Day
}

// Global returns true if this is the global series
func (d *Data) String() string {
	if d.Global() {
		return fmt.Sprintf("%s (%d)", "Global", len(d.Days))
	} else if d.Province == "" {
		return fmt.Sprintf("%s (%d)", d.Country, len(d.Days))
	}
	return fmt.Sprintf("%s, %s (%d)", d.Province, d.Country, len(d.Days))
}

// SetUpdated updates UpdatedAt if it is before this new time
func (d *Data) SetUpdated(updated time.Time) {
	if d.UpdatedAt.Before(updated) {
		d.UpdatedAt = updated
	}
}

// Global returns true if this is the global series
func (d *Data) Global() bool {
	return d.Country == "" && d.Province == ""
}

// Valid returns true if this series is valid
// a series without days is considered invalid
func (d *Data) Valid() bool {
	return len(d.Days) == 0
}

// Key converts a value into one suitable for use in urls
func (d *Data) Key(v string) string {
	return strings.Replace(strings.ToLower(v), " ", "-", -1)
}

// Match returns true if this series matches country and province
// performs a case insensitive match
func (d *Data) Match(country string, province string) bool {
	return d.MatchCountry(country) && d.MatchProvince(province)
}

// MatchCountry return true if this series matches country
// performs a case insensitive match
func (d *Data) MatchCountry(country string) bool {
	return d.Key(d.Country) == d.Key(country)
}

// MatchProvince return true if this series matches province
// performs a case insensitive match
func (d *Data) MatchProvince(province string) bool {
	return d.Key(d.Province) == d.Key(province)
}

// FetchDate returns the datapoint for a given date and dataKind
func (d *Data) FetchDate(date time.Time, dataKind int) int {

	for _, d := range d.Days {
		if d.Date.Equal(date) {
			switch dataKind {
			case DataDeaths:
				return d.Deaths
			case DataConfirmed:
				return d.Confirmed
			case DataRecovered:
				return d.Recovered
			case DataTested:
				return d.Tested
			}
		}
	}

	return 0
}

// LastDay returns the last day in the series
// a blank day is returned if no days
func (d *Data) LastDay() *Day {
	if len(d.Days) == 0 {
		return &Day{}
	}
	return d.Days[len(d.Days)-1]
}

// TotalDeaths returns the cumulative death due to COVID-19 for this series
func (d *Data) TotalDeaths() int {
	return d.LastDay().Deaths
}

// TotalConfirmed returns the cumulative confirmed cases of COVID-19 for this series
func (d *Data) TotalConfirmed() int {
	return d.LastDay().Confirmed
}

// TotalRecovered returns the cumulative recovered cases of COVID-19 for this series
func (d *Data) TotalRecovered() int {
	return d.LastDay().Recovered
}

// TotalTested returns the cumulative tested cases of COVID-19 for this series
func (d *Data) TotalTested() int {
	return d.LastDay().Tested
}

// AddData adds the given series of data to this series
// existing data for that dataKind will be replaced
func (d *Data) AddData(startDate time.Time, dataKind int, values []int) error {

	log.Printf("data: add data of kind:%d data:%v", dataKind, values)

	// If we don't have enough days, add some
	if len(d.Days) < len(values) {
		log.Printf("addDays:%d %d", len(d.Days), len(values))
		d.AddDays(len(values) - len(d.Days))
	}

	// Now set the values for this datakind on each day we have
	for i, day := range d.Days {

		// Check date on first day matches
		if i == 0 && !day.Date.Equal(startDate) {
			return fmt.Errorf("series: mismatch on start date for data:%v %v", startDate, day.Date)
		}
		//log.Printf("day:%d", values[i])
		// Fill in the value on each day from values
		err := day.SetData(dataKind, values[i])
		if err != nil {
			return fmt.Errorf("series: failed to add day:%v error:%s", day, err)
		}
	}

	return nil
}

// AddDays adds the given number of days to the end of our series
func (d *Data) AddDays(count int) {
	// Get the last day (if any), and start a day after, otherwie start afresh
	date := seriesStartDate
	if len(d.Days) > 0 {
		date = d.LastDay().Date.AddDate(0, 0, 1)
	}

	for i := 0; i < count; i++ {
		day := &Day{
			Date: date,
		}
		d.Days = append(d.Days, day)
		date = date.AddDate(0, 0, 1)
	}
}

// FIXME - I think this won't be required

// AddDay adds a day to this series
// an error is returned if the date is not at the end of the series
func (d *Data) AddDay(date time.Time, deaths, confirmed, recovered, tested int) error {
	// Check data is valid
	if date.IsZero() {
		return fmt.Errorf("series: invalid zero date in AddDay")
	}

	// Check date is more than the last date in series
	if len(d.Days) > 0 {
		if !d.LastDay().Date.Before(date) {
			return fmt.Errorf("series: invalid date added")
		}
	}

	// What about updating an existing day, do we ever do that?
	// Different function for that.

	day := &Day{
		Date:      date,
		Deaths:    deaths,
		Confirmed: confirmed,
		Recovered: recovered,
		Tested:    tested,
	}

	d.Days = append(d.Days, day)
	return nil
}
