package series

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// loadCSV loads the given CSV file into memory
func loadCSV(p string) ([][]string, error) {
	p = filepath.Clean(p)
	log.Printf("data: loading file at path:%v", p)

	// Open the file at path
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func readStateRow(row []string) (time.Time, int, int, error) {

	// Dates are, remarkably, in two different formats in one file
	// Try first in the one true format
	var updated time.Time
	var err error
	if row[3] != "" {
		// Ignore blank dates
		updated, err = time.Parse("2006-01-02 15:04:05", row[3])
		if err != nil {
			// Then try the US format  3/13/2020 22:22
			updated, err = time.Parse("1/2/2006 15:04", row[3])
			if err != nil {
				return updated, 0, 0, fmt.Errorf("load: error reading updated at series:%s error:%s", row[1], err)
			}
		}
	}

	confirmed, err := strconv.Atoi(row[6])
	if err != nil {
		return updated, 0, 0, fmt.Errorf("load: error reading confirmed series:%s error:%s", row[1], err)
	}

	deaths, err := strconv.Atoi(row[7])
	if err != nil {
		return updated, 0, 0, fmt.Errorf("load: error reading deaths series:%s error:%s", row[1], err)
	}
	/*
		recovered, err := strconv.Atoi(row[8])
		if err != nil {
			return updated, 0, 0, 0, fmt.Errorf("load: error reading recovered series:%s error:%s", row[1], err)
		}
	*/
	return updated, confirmed, deaths, nil
}

func readCountryRow(row []string) (time.Time, int, int, error) {

	// Dates are, remarkably, in two different formats in one file
	// Try first in the one true format
	updated, err := time.Parse("2006-01-02 15:04:05", row[1])
	if err != nil {
		// Then try the US format  3/13/2020 22:22
		updated, err = time.Parse("1/2/2006 15:04", row[1])
		if err != nil {
			return updated, 0, 0, fmt.Errorf("load: error reading updated at series:%s error:%s", row[0], err)
		}
	}

	confirmed, err := strconv.Atoi(row[4])
	if err != nil {
		return updated, 0, 0, fmt.Errorf("load: error reading confirmed series:%s error:%s", row[0], err)
	}

	deaths, err := strconv.Atoi(row[5])
	if err != nil {
		return updated, 0, 0, fmt.Errorf("load: error reading deaths series:%s error:%s", row[0], err)
	}
	/*
		recovered, err := strconv.Atoi(row[6])
		if err != nil {
			return updated, 0, 0, 0, fmt.Errorf("load: error reading recovered series:%s error:%s", row[0], err)
		}
	*/
	return updated, confirmed, deaths, nil
}
