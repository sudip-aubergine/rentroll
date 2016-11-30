package rcsv

import (
	"fmt"
	"rentroll/rlib"
	"strconv"
	"strings"
)

// CSV file format:
//   0   1     2        3         4         5        6                  7                   8
// 	BUD, TCID, VehicleMake, VehicleModel, VehicleColor, VehicleYear, LicensePlateState, LicensePlateNumber, ParkingPermitNumber
// 	REX, 1
// 	REX, 1
// 	REX, 1
// 	REX, 1
// 	REX, 1
// 	REX, 1
// 	REX, 1

// CreateVehicleFromCSV reads a rental specialty type string array and creates a database record for the rental specialty type.
// If the return value is not 0, abort the csv load
func CreateVehicleFromCSV(sa []string, lineno int) (string, int) {
	funcname := "CreateVehicleFromCSV"

	var (
		err error
		tr  rlib.Transactant
		t   rlib.Vehicle
	)

	const (
		BUD                 = 0
		TCID                = iota
		VehicleType         = iota
		VehicleMake         = iota
		VehicleModel        = iota
		VehicleColor        = iota
		VehicleYear         = iota
		LicensePlateState   = iota
		LicensePlateNumber  = iota
		ParkingPermitNumber = iota
		DtStart             = iota
		DtStop              = iota
	)

	// csvCols is an array that defines all the columns that should be in this csv file
	var csvCols = []CSVColumn{
		{"BUD", BUD},
		{"User", TCID},
		{"VehicleType", VehicleType},
		{"VehicleMake", VehicleMake},
		{"VehicleModel", VehicleModel},
		{"VehicleColor", VehicleColor},
		{"VehicleYear", VehicleYear},
		{"LicensePlateState", LicensePlateState},
		{"LicensePlateNumber", LicensePlateNumber},
		{"ParkingPermitNumber", ParkingPermitNumber},
		{"DtStart", DtStart},
		{"DtStop", DtStop},
	}

	rs, y := ValidateCSVColumns(csvCols, sa, funcname, lineno)
	if y > 0 {
		return rs, 1
	}
	if lineno == 1 {
		return rs, 0
	}

	for i := 0; i < len(sa); i++ {
		s := strings.TrimSpace(sa[i])
		switch i {
		case BUD: // business
			des := strings.ToLower(strings.TrimSpace(sa[0])) // this should be BUD

			//-------------------------------------------------------------------
			// Make sure the rlib.Business is in the database
			//-------------------------------------------------------------------
			if len(des) > 0 { // make sure it's not empty
				b1 := rlib.GetBusinessByDesignation(des) // see if we can find the biz
				if len(b1.Designation) == 0 {
					rs += fmt.Sprintf("%s: line %d, Business with designation %s does not exist\n", funcname, lineno, sa[0])
					return rs, CsvErrorSensitivity
				}
				tr.BID = b1.BID
			}
		case TCID:
			tr = rlib.GetTransactantByPhoneOrEmail(tr.BID, s)
			if tr.TCID < 1 {
				rs += fmt.Sprintf("%s: line %d, no Transactant found with %s listed as a phone or email\n", funcname, lineno, s)
				return rs, CsvErrorSensitivity
			}
			t.TCID = tr.TCID
		case VehicleType:
			t.VehicleType = s
		case VehicleMake:
			t.VehicleMake = s
		case VehicleModel:
			t.VehicleModel = s
		case VehicleColor:
			t.VehicleColor = s
		case VehicleYear:
			if len(s) > 0 {
				i, err := strconv.Atoi(strings.TrimSpace(s))
				if err != nil {
					rs += fmt.Sprintf("%s: line %d - VehicleYear value is invalid: %s\n", funcname, lineno, s)
					return rs, CsvErrorSensitivity
				}
				t.VehicleYear = int64(i)
			}
		case LicensePlateState:
			t.LicensePlateState = s
		case LicensePlateNumber:
			t.LicensePlateNumber = s
		case ParkingPermitNumber:
			t.ParkingPermitNumber = s
		case DtStart:
			if len(s) > 0 {
				t.DtStart, err = rlib.StringToDate(s) // required field
				if err != nil {
					rs += fmt.Sprintf("%s: line %d - invalid start date.  Error = %s\n", funcname, lineno, err.Error())
					return rs, CsvErrorSensitivity
				}
			}
		case DtStop:
			if len(s) > 0 {
				t.DtStop, err = rlib.StringToDate(s) // required field
				if err != nil {
					rs += fmt.Sprintf("%s: line %d - invalid start date.  Error = %s\n", funcname, lineno, err.Error())
					return rs, CsvErrorSensitivity
				}
			}
		default:
			rs += fmt.Sprintf("i = %d, unknown field\n", i)
		}
	}

	//-------------------------------------------------------------------
	// Check for duplicate...
	//-------------------------------------------------------------------
	tm := rlib.GetVehiclesByLicensePlate(t.LicensePlateNumber)
	for i := 0; i < len(tm); i++ {
		if t.LicensePlateNumber == tm[i].LicensePlateNumber && t.LicensePlateState == tm[i].LicensePlateState {
			rs += fmt.Sprintf("%s: line %d - vehicle with License Plate %s in State = %s already exists\n", funcname, lineno, t.LicensePlateNumber, t.LicensePlateState)
			return rs, CsvErrorSensitivity
		}
	}

	//-------------------------------------------------------------------
	// OK, just insert the records and we're done
	//-------------------------------------------------------------------
	t.TCID = tr.TCID
	t.BID = tr.BID
	vid, err := rlib.InsertVehicle(&t)
	if nil != err {
		rs += fmt.Sprintf("%s: line %d - error inserting Vehicle = %v\n", funcname, lineno, err)
		return rs, CsvErrorSensitivity
	}

	if vid == 0 {
		rs += fmt.Sprintf("%s: line %d - after InsertVehicle vid = %d\n", funcname, lineno, vid)
		return rs, CsvErrorSensitivity
	}
	return rs, 0
}

// LoadVehicleCSV loads a csv file with vehicles
func LoadVehicleCSV(fname string) string {
	rs := ""
	t := rlib.LoadCSV(fname)
	for i := 0; i < len(t); i++ {
		s, err := CreateVehicleFromCSV(t[i], i+1)
		rs += s
		if err > 0 {
			break
		}
	}
	return rs
}