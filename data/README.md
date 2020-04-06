# Data Formats 

Data is stored in csv files, with a simpler format based on a series of days starting on 2020-01-22 (the first day with data from our data sources). These files are read once on load, and updated periodically. All timestamps are treated as starting at 0 hours UTC. Series data is updated once a day after the end of the day at UTC 0 hours to add the latest day for all areas with data. Daily data is updated periodically throughout the day from the sources below. 

## Area data 

Area data which is relatively unchanging is stored in the areas.csv file, including location, population etc. Each area has a numeric id which is used to refer to it as area_id in other files. 

## Series data 

Series data is stored in a file with an row per day per area_id (where data is non-zero). Areas with all 0 data for a given day are ommitted to save space.


# Data sources

* US data is available from data compiled by (John Hopkins)[https://github.com/CSSEGISandData/COVID-19]
* US testing data from (CDC)[https://www.cdc.gov/coronavirus/2019-ncov/cases-updates/testing-in-us.html] and (covidtracking.com)[https://covidtracking.com/data/]
* UK data sourced from (gov.uk)[https://www.gov.uk/government/publications/covid-19-track-coronavirus-cases] 
* Test Data from (worldometers)[https://www.worldometers.info/coronavirus/]
* Japan data from (mhlw.go.jp)[https://www.mhlw.go.jp/stf/seisakunitsuite/bunya/newpage_00032.html]