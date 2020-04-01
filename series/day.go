package series

import (
	"fmt"
	"time"
)

// Day represents data for a day in a series
// Totals are cumulative deaths etc, not single day counts
type Day struct {
	Date      time.Time
	Deaths    int
	Confirmed int
	Recovered int
	Tested    int
}

// String returns a string representation of this Day
func (d *Day) String() string {
	return fmt.Sprintf("%s %d-%d-%d-%d", d.DateMachine(), d.Deaths, d.Confirmed, d.Recovered, d.Tested)
}

// DateMachine returns a string for machines
func (d *Day) DateMachine() string {
	return d.Date.Format("2006-01-02")
}

// DateDisplay returns a string for humans
func (d *Day) DateDisplay() string {
	return d.Date.Format("2 Jan, 2006")
}

// SetData sets data to this day for the given data kind
// the data replaces existing data
func (d *Day) SetData(dataKind, value int) error {
	switch dataKind {
	case DataDeaths:
		d.Deaths = value
	case DataConfirmed:
		d.Confirmed = value
	case DataRecovered:
		d.Recovered = value
	case DataTested:
		d.Tested = value
	default:
		return fmt.Errorf("invalid data kind:%d", dataKind)
	}

	return nil
}
