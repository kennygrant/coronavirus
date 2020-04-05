package series

import (
	"log"
)

// ScheduleUpdate schedules data updates from our data sources
// after each update data series is resaved and a data reload triggered
// the changes are also committed to the git repository
func ScheduleUpdate() {
	log.Printf("series: scheduling updates")

	// We could schedule 10 minute updates from data source?
	// Some data sources only update daily so perhaps best just to depend on maanual for those

	// We could schedule hourly update to catch any manual updates to repo?
}
