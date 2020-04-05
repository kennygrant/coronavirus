package series

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

// UpdateFromJHUCases updates from JHU case data
// Cols: FIPS,Admin2,Province_State,Country_Region,Last_Update,Lat,Long_,Confirmed,Deaths,Recovered,Active,Combined_Key
func UpdateFromJHUCases(rows [][]string) error {

	log.Printf("series: update from JHU %d rows", len(rows))

	// For each row in the input data, reject if admin2 completed
	for i, row := range rows {
		// Check format on row 0
		if i == 0 {
			if row[0] != "FIPS" || row[2] != "Province_State" || row[11] != "Combined_Key" {
				return fmt.Errorf("error reading JHU cases - format invalid for row:%s", row)
			}
			continue
		}

		// Reject rows with Admin2 completed
		if row[1] != "" {
			continue
		}

		province := row[2]
		country := row[3]

		// Read other rows which are are interested in, and ask series to update last day if changed
		series, err := dataset.FetchSeries(country, province)
		if err != nil || series == nil {
			continue
		}

		// If we reach here we have a valid row and series
		updated, deaths, confirmed, recovered, err := readJHURow(row)
		if err != nil {
			continue
		}

		// We don't hav etested data from JHU so leave it unchanged
		series.UpdateToday(updated, deaths, confirmed, recovered, 0)

		log.Printf("update: %s u:%v d:%d c:%d r:%d", series, updated, deaths, confirmed, recovered)

	}

	return nil
}

// Note row order is different from our standard order
//		 0 		1		2				3			4			5   6	    7		8		9		 10	    11
// Cols: FIPS,Admin2,Province_State,Country_Region,Last_Update,Lat,Long_,Confirmed,Deaths,Recovered,Active,Combined_Key
func readJHURow(row []string) (time.Time, int, int, int, error) {

	// Dates are, remarkably, in two different formats in one file
	// Try first in the one true format
	updated, err := time.Parse("2006-01-02 15:04:05", row[4])
	if err != nil {
		return updated, 0, 0, 0, fmt.Errorf("load: error reading updated at series:%s error:%s", row[0], err)
	}

	confirmed, err := strconv.Atoi(row[4])
	if err != nil {
		return updated, 0, 0, 0, fmt.Errorf("load: error reading confirmed series:%s error:%s", row[0], err)
	}

	deaths, err := strconv.Atoi(row[5])
	if err != nil {
		return updated, 0, 0, 0, fmt.Errorf("load: error reading deaths series:%s error:%s", row[0], err)
	}

	recovered, err := strconv.Atoi(row[6])
	if err != nil {
		return updated, 0, 0, 0, fmt.Errorf("load: error reading recovered series:%s error:%s", row[0], err)
	}

	return updated, deaths, confirmed, recovered, nil
}
