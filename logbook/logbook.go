package logbook

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"strconv"
	"strings"
	"time"

	sm "github.com/flopp/go-staticmaps"
	"github.com/fogleman/gg"
	"github.com/golang/geo/s2"
	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type LogbookConfig struct {
	SourceType     string
	FileName       string
	APIKey         string
	SpreadsheetID  string
	StartRow       int
	LogbookOwner   string
	PageBrakes     []string
	Reverse        bool
	FilterNoRoutes bool
	FilterDate     string
}

// logbook time type, sort of a wrapper for time.Duration
type logbookTime struct {
	time time.Duration
}

func (t *logbookTime) SetTime(strTime string) {
	var err error

	if strTime == "" {
		strTime = "0:0"
	}

	strTime = fmt.Sprintf("%sm", strings.ReplaceAll(strTime, ":", "h"))

	t.time, err = time.ParseDuration(strTime)
	if err != nil {
		fmt.Printf("Error parsing time %s", strTime)
		t.time, _ = time.ParseDuration("0h0m")
	}
}

func (t *logbookTime) GetTime(params ...bool) string {
	d := t.time.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute

	if h == 0 && m == 0 {
		if len(params) > 0 { // means used to print totals
			return "0:00"
		} else {
			return ""
		}
	} else {
		return fmt.Sprintf("%01d:%02d", h, m)
	}
}

// times structure
type times struct {
	se         logbookTime
	me         logbookTime
	mcc        logbookTime
	night      logbookTime
	ifr        logbookTime
	pic        logbookTime
	copilot    logbookTime
	dual       logbookTime
	instructor logbookTime
	total      logbookTime
}

// location structure, contains place and time for departure or arrival
type location struct {
	place string
	time  string
}

// landings structure, days and night landings
type landing struct {
	day   int
	night int
}

// logbook record type structure
type logbookRecord struct {
	date      string
	departure location
	arrival   location

	aircraft struct {
		model string
		reg   string
	}

	time     times
	landings landing

	sim struct {
		name string
		time logbookTime
	}

	pic     string
	remarks string
}

// type structure to calculate totals
type logbookTotalRecord struct {
	time     times
	landings landing

	sim struct {
		time logbookTime
	}
}

// some global vars
var leftMargin = 10.0
var topMargin = 30.0
var logbookRows = 23
var bodyRowHeight = 5.0
var footerRowHeight = 6.0

var sheetName = "Flights"

var header1 = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
var header2 = []string{"DATE", "DEPARTURE", "ARRIVAL", "AIRCRAFT", "SINGLE PILOT TIME", "MULTI PILOT TIME", "TOTAL TIME", "PIC NAME", "LANDINGS", "OPERATIONAL CONDITION TIME", "PILOT FUNCTION TIME", "FSTD SESSION", "REMARKS AND ENDORSMENTS"}
var header3 = []string{"", "Place", "Time", "Place", "Time", "Type", "Reg", "SE", "ME", "", "", "", "Day", "Night", "Night", "IFR", "PIC", "COP", "DUAL", "INSTR", "Type", "Time", ""}

var w1 = []float64{12.2, 16.5, 16.5, 22.9, 33.6, 11.2, 22.86, 16.76, 22.4, 44.8, 22.4, 33.8}
var w2 = []float64{12.2, 16.5, 16.5, 22.9, 22.4, 11.2, 11.2, 22.86, 16.76, 22.4, 44.8, 22.4, 33.8}
var w3 = []float64{12.2, 8.25, 8.25, 8.25, 8.25, 10, 12.9, 11.2, 11.2, 11.2, 11.2, 22.86, 8.38, 8.38, 11.2, 11.2, 11.2, 11.2, 11.2, 11.2, 11.2, 11.2, 33.8}
var w4 = []float64{20.45, 47.65, 11.2, 11.2, 11.2, 11.2, 22.86, 8.38, 8.38, 11.2, 11.2, 11.2, 11.2, 11.2, 11.2, 11.2, 11.2, 33.8}

//go:embed  db/airports.json font/*
var content embed.FS

// getLogbookDump reads the logbook from the source (google spreadsheet, local xlsx file)
func getLogbookDump(logbookConfig LogbookConfig) (values [][]interface{}, err error) {

	if logbookConfig.SourceType == "google" {
		// get data from google spreadsheet
		ctx := context.Background()

		srv, err := sheets.NewService(ctx, option.WithAPIKey(logbookConfig.APIKey))
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve Google Spreadsheets client: %v", err)
		}

		response, err := srv.Spreadsheets.Values.Get(logbookConfig.SpreadsheetID, fmt.Sprintf("%s!A%d:W", sheetName, logbookConfig.StartRow)).Do()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
		}

		if len(response.Values) == 0 {
			return nil, fmt.Errorf("no data found in the sheet")
		}

		values = response.Values

	} else {
		// get data from local xlsx file
		xls, err := excelize.OpenFile(logbookConfig.FileName)
		if err != nil {
			return nil, fmt.Errorf("error opening xlsx file: %v", err)
		}

		rows, err := xls.GetRows(sheetName)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
		}

		for i, row := range rows {
			// skip first rows (headers)
			if i < logbookConfig.StartRow-1 {
				continue
			}

			var appendRow []interface{}

			for _, colCell := range row {
				if strings.HasPrefix(colCell, ":") {
					// somehow in case the time equals 0:mm it returns :mm only
					colCell = "0" + colCell
				}
				appendRow = append(appendRow, colCell)
			}

			values = append(values, appendRow)
		}

	}

	return values, err

}

// parseRecord returns a formed and parsed logbookRecord
//
// row []interface{} - element in *sheets.ValueRange from getLogbookDump function
func parseRecord(row []interface{}) logbookRecord {
	var record logbookRecord

	record.date = row[0].(string)
	record.departure.place = row[1].(string)
	record.departure.time = row[2].(string)
	record.arrival.place = row[3].(string)
	record.arrival.time = row[4].(string)
	record.aircraft.model = row[5].(string)
	record.aircraft.reg = row[6].(string)
	record.time.se.SetTime(row[7].(string))
	if row[9].(string) == "" && row[8].(string) != "" {
		record.time.me.SetTime(row[8].(string))
	} else {
		record.time.me.SetTime("")
	}
	record.time.mcc.SetTime(row[9].(string))
	record.time.total.SetTime(row[10].(string))
	record.landings.day, _ = strconv.Atoi(row[11].(string))
	record.landings.night, _ = strconv.Atoi(row[12].(string))
	record.time.night.SetTime(row[13].(string))
	record.time.ifr.SetTime(row[14].(string))
	record.time.pic.SetTime(row[15].(string))
	record.time.copilot.SetTime(row[16].(string))
	record.time.dual.SetTime(row[17].(string))
	record.time.instructor.SetTime(row[18].(string))
	record.sim.name = row[19].(string)
	record.sim.time.SetTime(row[20].(string))
	record.pic = row[21].(string)
	if len(row) > 22 {
		record.remarks = row[22].(string)
	}

	return record
}

// calculateTotals sums the provided logbookTotalRecord variable with logbook record.
// This is sort of append function for the custom type
//
// totals logbookTotalRecord - one of the totals which will be printed in the footer
//
// record logbookRecord - logbook record
func calculateTotals(totals logbookTotalRecord, record logbookRecord) logbookTotalRecord {

	totals.time.se.time += record.time.se.time
	totals.time.me.time += record.time.me.time
	totals.time.mcc.time += record.time.mcc.time
	totals.time.night.time += record.time.night.time
	totals.time.ifr.time += record.time.ifr.time
	totals.time.pic.time += record.time.pic.time
	totals.time.copilot.time += record.time.copilot.time
	totals.time.dual.time += record.time.dual.time
	totals.time.instructor.time += record.time.instructor.time
	totals.time.total.time += record.time.total.time

	totals.landings.day += record.landings.day
	totals.landings.night += record.landings.night

	totals.sim.time.time += record.sim.time.time

	return totals
}

// printLogbookHeader creates the header of the logbook page
//
// pdf *gofpdf.Fpdf - pdf object
func printLogbookHeader(pdf *gofpdf.Fpdf) {

	pdf.SetFillColor(217, 217, 217)
	pdf.SetFont("LiberationSansNarrow-Bold", "", 8)

	pdf.SetX(leftMargin)
	pdf.SetY(topMargin)

	// First header
	x, y := pdf.GetXY()
	for i, str := range header1 {
		width := w1[i]
		pdf.Rect(x, y-1, width, 5, "FD")
		pdf.MultiCell(width, 1, str, "", "C", false)
		x += width
		pdf.SetXY(x, y)
	}
	pdf.Ln(-1)

	// Second header
	x, y = pdf.GetXY()
	y += 2
	pdf.SetY(y)
	for i, str := range header2 {
		width := w2[i]
		pdf.Rect(x, y-1, width, 12, "FD")
		pdf.MultiCell(width, 3, str, "", "C", false)
		x += width
		pdf.SetXY(x, y)
	}
	pdf.Ln(-1)

	// Header inside header
	x, y = pdf.GetXY()
	y += 5
	pdf.SetY(y)
	for i, str := range header3 {
		width := w3[i]
		if str != "" {
			pdf.Rect(x, y-1, width, 4, "FD")
			pdf.MultiCell(width, 2, str, "", "C", false)
		}
		x += width
		pdf.SetXY(x, y)
	}
	pdf.Ln(-1)

	// Align the logbook body
	_, y = pdf.GetXY()
	y += 1
	pdf.SetY(y)
}

// printLogbookFooter forms the logbook footer with calculated totals
//
// pdf *gofpdf.Fpdf - pdf object
//
// logbookOwner string - owner's name to print in the footer of the logbook
//
// totalPage logbookTotalRecord - contains totals on the page
// totalPrevious logbookTotalRecord - contains totals of the previous pages
// totalTime logbookTotalRecord - contains totals of all times
func printLogbookFooter(pdf *gofpdf.Fpdf, logbookOwner string, totalPage logbookTotalRecord, totalPrevious logbookTotalRecord, totalTime logbookTotalRecord) {

	printTotal := func(totalName string, total logbookTotalRecord) {
		pdf.SetFillColor(217, 217, 217)
		pdf.SetFont("LiberationSansNarrow-Bold", "", 8)

		pdf.SetX(leftMargin)

		if totalName == "TOTAL THIS PAGE" {
			pdf.CellFormat(w4[0], footerRowHeight, "", "LTR", 0, "", true, 0, "")
		} else if totalName == "TOTAL FROM PREVIOUS PAGES" {
			pdf.CellFormat(w4[0], footerRowHeight, "", "LR", 0, "", true, 0, "")
		} else {
			pdf.CellFormat(w4[0], footerRowHeight, "", "LBR", 0, "", true, 0, "")
		}
		pdf.CellFormat(w4[1], footerRowHeight, totalName, "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[2], footerRowHeight, total.time.se.GetTime(true), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[3], footerRowHeight, total.time.me.GetTime(true), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[4], footerRowHeight, total.time.mcc.GetTime(true), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[5], footerRowHeight, total.time.total.GetTime(true), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[6], footerRowHeight, "", "1", 0, "", true, 0, "")
		pdf.CellFormat(w4[7], footerRowHeight, fmt.Sprintf("%d", total.landings.day), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[8], footerRowHeight, fmt.Sprintf("%d", total.landings.night), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[9], footerRowHeight, total.time.night.GetTime(true), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[10], footerRowHeight, total.time.ifr.GetTime(true), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[11], footerRowHeight, total.time.pic.GetTime(true), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[12], footerRowHeight, total.time.copilot.GetTime(true), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[13], footerRowHeight, total.time.dual.GetTime(true), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[14], footerRowHeight, total.time.instructor.GetTime(true), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4[15], footerRowHeight, "", "1", 0, "", true, 0, "")
		pdf.CellFormat(w4[16], footerRowHeight, total.sim.time.GetTime(true), "1", 0, "C", true, 0, "")

		pdf.SetFont("LiberationSansNarrow-Regular", "", 6)
		if totalName == "TOTAL THIS PAGE" {
			pdf.CellFormat(w4[17], footerRowHeight, "I certify that the entries in this log are true.", "LTR", 0, "C", true, 0, "")
		} else if totalName == "TOTAL FROM PREVIOUS PAGES" {
			pdf.CellFormat(w4[17], footerRowHeight, "", "LR", 0, "", true, 0, "")
		} else {
			pdf.CellFormat(w4[17], footerRowHeight, logbookOwner, "LBR", 0, "C", true, 0, "")
		}

		pdf.Ln(-1)
	}

	printTotal("TOTAL THIS PAGE", totalPage)
	printTotal("TOTAL FROM PREVIOUS PAGES", totalPrevious)
	printTotal("TOTAL TIME", totalTime)

}

// printLogbookBody forms and prints the logbook row
//
// pdf *gofpdf.Fpdf - pdf object
//
// record logbookRecord - logbook record
//
// fill bool - identifies if the row will be filled with gray color
func printLogbookBody(pdf *gofpdf.Fpdf, record logbookRecord, fill bool) {

	pdf.SetFillColor(228, 228, 228)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("LiberationSansNarrow-Regular", "", 8)

	// 	Data

	pdf.SetX(leftMargin)
	pdf.CellFormat(w3[0], bodyRowHeight, record.date, "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[1], bodyRowHeight, record.departure.place, "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[2], bodyRowHeight, record.departure.time, "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[3], bodyRowHeight, record.arrival.place, "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[4], bodyRowHeight, record.arrival.time, "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[5], bodyRowHeight, record.aircraft.model, "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[6], bodyRowHeight, record.aircraft.reg, "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[7], bodyRowHeight, record.time.se.GetTime(), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[8], bodyRowHeight, record.time.me.GetTime(), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[9], bodyRowHeight, record.time.mcc.GetTime(), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[10], bodyRowHeight, record.time.total.GetTime(), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[11], bodyRowHeight, record.pic, "1", 0, "L", fill, 0, "")
	if record.landings.day != 0 {
		pdf.CellFormat(w3[12], bodyRowHeight, fmt.Sprintf("%d", record.landings.day), "1", 0, "C", fill, 0, "")
	} else {
		pdf.CellFormat(w3[12], bodyRowHeight, "", "1", 0, "C", fill, 0, "")

	}
	if record.landings.night != 0 {
		pdf.CellFormat(w3[13], bodyRowHeight, fmt.Sprintf("%d", record.landings.night), "1", 0, "C", fill, 0, "")
	} else {
		pdf.CellFormat(w3[13], bodyRowHeight, "", "1", 0, "C", fill, 0, "")

	}
	pdf.CellFormat(w3[14], bodyRowHeight, record.time.night.GetTime(), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[15], bodyRowHeight, record.time.ifr.GetTime(), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[16], bodyRowHeight, record.time.pic.GetTime(), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[17], bodyRowHeight, record.time.copilot.GetTime(), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[18], bodyRowHeight, record.time.dual.GetTime(), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[19], bodyRowHeight, record.time.instructor.GetTime(), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[20], bodyRowHeight, record.sim.name, "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[21], bodyRowHeight, record.sim.time.GetTime(), "1", 0, "C", fill, 0, "")
	pdf.CellFormat(w3[22], bodyRowHeight, record.remarks, "1", 0, "L", fill, 0, "")

	pdf.Ln(-1)

	pdf.SetX(leftMargin)
}

// fillLine returns if the logbook line should be filled with gray color
func fillLine(rowCounter int) bool {
	if (rowCounter+1)%3 == 0 { // fill every 3rd row only
		return true
	} else {
		return false
	}
}

// LoadFonts loads fonts for pdf object from embed fs
func LoadFonts(pdf *gofpdf.Fpdf) {

	fontRegularBytes, _ := content.ReadFile("font/LiberationSansNarrow-Regular.ttf")
	pdf.AddUTF8FontFromBytes("LiberationSansNarrow-Regular", "", fontRegularBytes)

	fontBoldBytes, _ := content.ReadFile("font/LiberationSansNarrow-Bold.ttf")
	pdf.AddUTF8FontFromBytes("LiberationSansNarrow-Bold", "", fontBoldBytes)

}

// Export reads the google spreadsheet and create pdf with logbook in EASA format
func Export(logbookConfig LogbookConfig) {

	// get data from the google spreadsheet
	response, err := getLogbookDump(logbookConfig)
	if err != nil {
		log.Fatalf("Cannot get logbook dump: %v", err)
	}

	// start forming the pdf file
	pdf := gofpdf.New("L", "mm", "A4", "")
	LoadFonts(pdf)

	pdf.SetLineWidth(.2)

	rowCounter := 0
	pageCounter := 1

	var totalPage logbookTotalRecord
	var totalPrevious logbookTotalRecord
	var totalTime logbookTotalRecord
	var totalEmpty logbookTotalRecord

	pdf.AddPage()
	printLogbookHeader(pdf)

	fill := false

	logBookRow := func(item int) {
		rowCounter += 1

		record := parseRecord(response[item])

		totalPage = calculateTotals(totalPage, record)
		totalTime = calculateTotals(totalTime, record)

		printLogbookBody(pdf, record, fill)

		if rowCounter >= logbookRows {
			printLogbookFooter(pdf, logbookConfig.LogbookOwner, totalPage, totalPrevious, totalTime)
			totalPrevious = totalTime
			totalPage = totalEmpty

			// print page number
			pdf.SetY(pdf.GetY() - 1)
			pdf.CellFormat(0, 10, fmt.Sprintf("page %d", pageCounter), "", 0, "L", false, 0, "")

			// check for the page brakes to separate logbooks
			if len(logbookConfig.PageBrakes) > 0 {
				if fmt.Sprintf("%d", pageCounter) == logbookConfig.PageBrakes[0] {
					pdf.AddPage()
					pageCounter = 0

					logbookConfig.PageBrakes = append(logbookConfig.PageBrakes[:0], logbookConfig.PageBrakes[1:]...)
				}
			}

			rowCounter = 0
			pageCounter += 1

			pdf.AddPage()
			printLogbookHeader(pdf)
		}
		fill = fillLine(rowCounter)

	}

	if logbookConfig.Reverse {
		for i := len(response) - 1; i >= 0; i-- {
			logBookRow(i)
		}
	} else {
		for i := 0; i < len(response); i++ {
			logBookRow(i)
		}
	}

	// check the last page for the proper format
	var emptyRecord logbookRecord
	for i := rowCounter + 1; i <= logbookRows; i++ {
		printLogbookBody(pdf, emptyRecord, fill)
		fill = fillLine(i)

	}
	printLogbookFooter(pdf, logbookConfig.LogbookOwner, totalPage, totalPrevious, totalTime)
	// print page number
	pdf.SetY(pdf.GetY() - 1)
	pdf.CellFormat(0, 10, fmt.Sprintf("page %d", pageCounter), "", 0, "L", false, 0, "")

	// save and close pdf
	err = pdf.OutputFileAndClose("logbook.pdf")
	if err != nil {
		log.Fatalf("Cannot export pdf: %v\n", err)
	} else {
		fmt.Println("Loogbook has been exported to logbook.pdf")
	}
}

// loadAirportsDB loads the airports data (location and so on)
func loadAirportsDB() (map[string]interface{}, error) {
	var airports map[string]interface{}

	byteValue, err := content.ReadFile("db/airports.json")

	if err != nil {
		return airports, err
	}

	err = json.Unmarshal([]byte(byteValue), &airports)
	if err != nil {
		return nil, err
	}

	return airports, nil
}

// RendersMap generates a PNG file with airports markers and routes between them
func RendersMap(logbookConfig LogbookConfig) {

	airportMarkers := make(map[string]struct{})
	routeLines := make(map[string]struct{})

	var totals logbookTotalRecord

	// load airports.json
	airports, err := loadAirportsDB()
	if err != nil {
		log.Fatalf("Cannot load airports.json file: %v", err)
	}

	// get data from the google spreadsheet
	response, err := getLogbookDump(logbookConfig)
	if err != nil {
		log.Fatalf("Cannot get logbook dump: %v", err)
	}

	// parsing
	for _, row := range response {
		record := parseRecord(row)

		if (logbookConfig.FilterDate != "" && strings.Contains(record.date, logbookConfig.FilterDate)) || logbookConfig.FilterDate == "" {
			// add to the list of the airport markers departure and arrival
			// it will be automatically a list of unique airports
			airportMarkers[record.departure.place] = struct{}{}
			airportMarkers[record.arrival.place] = struct{}{}

			// the same for the route lines
			if !logbookConfig.FilterNoRoutes {
				if record.departure.place != record.arrival.place {
					routeLines[fmt.Sprintf("%s-%s", record.departure.place, record.arrival.place)] = struct{}{}
				}
			}

			totals = calculateTotals(totals, record)
		}

	}

	fmt.Printf("Airports: %d\n", len(airportMarkers))
	fmt.Printf("Routes: %d\n", len(routeLines))
	fmt.Printf("Total time: %s\n", totals.time.total.GetTime())
	fmt.Printf("Landings: %d day, %d night\n", totals.landings.day, totals.landings.night)

	ctx := sm.NewContext()
	ctx.SetSize(1920, 1080)

	// generate routes lines
	for route := range routeLines {
		places := strings.Split(route, "-")

		if airport1, ok := airports[places[0]].(map[string]interface{}); ok {
			if airport2, ok := airports[places[1]].(map[string]interface{}); ok {

				ctx.AddObject(
					sm.NewPath(
						[]s2.LatLng{
							s2.LatLngFromDegrees(airport1["lat"].(float64), airport1["lon"].(float64)),
							s2.LatLngFromDegrees(airport2["lat"].(float64), airport2["lon"].(float64)),
						},
						color.Black,
						0.5),
				)

			}
		}
	}

	// generate airports markers
	for place := range airportMarkers {

		if airport, ok := airports[place].(map[string]interface{}); ok {
			ctx.AddObject(
				sm.NewMarker(
					s2.LatLngFromDegrees(airport["lat"].(float64), airport["lon"].(float64)),
					color.RGBA{0xff, 0, 0, 0xff},
					16.0,
				),
			)
		}

	}

	img, err := ctx.Render()
	if err != nil {
		log.Fatalf("Cannot render a map %v", err)
	}

	if err := gg.SavePNG("map.png", img); err != nil {
		log.Fatalf("Cannot save a map %v", err)
	} else {
		fmt.Printf("Map has been saved to map.png\n")
	}
}
