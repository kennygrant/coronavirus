package series

import "sort"

// CalculateGlobalSeriesData adds some top level countries which are inexplicably missing from the original dataset
// presumably they calculate these on the fly
func CalculateGlobalSeriesData() error {

	// Lock during add operation
	mutex.Lock()
	defer mutex.Unlock()

	// Fetch series
	China, err := dataset.FetchSeries("China", "")
	if err != nil {
		return err
	}
	Australia, err := dataset.FetchSeries("Australia", "")
	if err != nil {
		return err
	}
	Canada, err := dataset.FetchSeries("Canada", "")
	if err != nil {
		return err
	}
	Global, err := dataset.FetchSeries("", "")
	if err != nil {
		return err
	}

	// Reset all these series as we're recalculating from scratch
	China.ResetDays()
	Australia.ResetDays()
	Canada.ResetDays()
	Global.ResetDays()

	// Add global country entries for countries with data broken down at province level
	// these are missing in the datasets from JHU for some reason, though US is now included
	for _, s := range dataset {

		// Build an overall China series
		if s.Country == "China" {
			err = China.MergeSeries(s)
			if err != nil {
				return err
			}
		}

		// Build an overall Australia series
		if s.Country == "Australia" {
			err = Australia.MergeSeries(s)
			if err != nil {
				return err
			}
		}

		// Build an overall Canada series
		if s.Country == "Canada" {
			err = Canada.MergeSeries(s)
			if err != nil {
				return err
			}
		}

		if s.ShouldIncludeInGlobal() {
			//	log.Printf("global:%s-%d", s.Country, s.TotalDeaths())
			err = Global.MergeSeries(s)
			if err != nil {
				return err
			}
		} else {
			if s.TotalDeaths() > 0 {
				//	log.Printf("ignore for global:%s deaths:%d", s, s.TotalDeaths())
			}
		}
	}

	// Sort entire dataset by deaths desc to get the right order
	sort.Stable(dataset)

	return nil
}
