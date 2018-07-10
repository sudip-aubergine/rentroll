package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"rentroll/rlib"
	"rentroll/rtags"
)

// RAFlowDetailRequest is a struct to hold info for Flow which is going to be validate
type RAFlowDetailRequest struct {
	FlowID    int64
	UserRefNo string
}

// ValidateRAFlowResponse is struct to hold ErrorList for Flow
type ValidateRAFlowResponse struct {
	Total     int                `json:"total"`
	ErrorType string             `json:"errortype"`
	Errors    RAFlowFieldsErrors `json:"errors"`
}

// DatesFieldsError is struct to hold Errorlist for Dates section
type DatesFieldsError struct {
	Total  int                 `json:"total"`
	Errors map[string][]string `json:"errors"`
}

// PeopleFieldsError is struct to hold Errorlist for People section
type PeopleFieldsError struct {
	TMPTCID int64
	Total   int                 `json:"total"`
	Errors  map[string][]string `json:"errors"`
}

// PetFieldsError is struct to hold Errorlist for Pet section
type PetFieldsError struct {
	TMPPETID   int64
	Total      int                 `json:"total"`
	Errors     map[string][]string `json:"errors"`
	FeesErrors []RAFeesError       `json:"fees"`
}

// VehicleFieldsError is struct to hold Errorlist for Vehicle section
type VehicleFieldsError struct {
	TMPVID     int64
	Total      int                 `json:"total"`
	Errors     map[string][]string `json:"errors"`
	FeesErrors []RAFeesError       `json:"fees"`
}

// RentablesFieldsError is to hold Errorlist for Rentables section
type RentablesFieldsError struct {
	RID        int64
	Total      int                 `json:"total"`
	Errors     map[string][]string `json:"errors"`
	FeesErrors []RAFeesError       `json:"fees"`
}

// RAFeesError is struct to hold Errolist for Fees of vehicles
type RAFeesError struct {
	TMPASMID int64
	Total    int                 `json:"total"`
	Errors   map[string][]string `json:"errors"`
}

// ParentChildFieldsError is to hold Errorlist for Parent/Child section
type ParentChildFieldsError struct {
	PRID   int64               // parent rentable ID
	CRID   int64               // child rentable ID
	Total  int                 `json:"total"`
	Errors map[string][]string `json:"errors"`
}

// TiePeopleFieldsError is to hold Errorlist for TiePeople section
type TiePeopleFieldsError struct {
	TMPTCID int64
	Total   int                 `json:"total"`
	Errors  map[string][]string `json:"errors"`
}

// TieFieldsError is to hold Errorlist for Tie section
type TieFieldsError struct {
	TiePeople []TiePeopleFieldsError `json:"people"`
}

// RAFlowFieldsErrors is to hold Errorlist for each section of RAFlow
type RAFlowFieldsErrors struct {
	Dates       DatesFieldsError         `json:"dates"`
	People      []PeopleFieldsError      `json:"people"`
	Pets        []PetFieldsError         `json:"pets"`
	Vehicle     []VehicleFieldsError     `json:"vehicle"`
	Rentables   []RentablesFieldsError   `json:"rentables"`
	ParentChild []ParentChildFieldsError `json:"parentchild"`
	Tie         TieFieldsError           `json:"tie"`
}

// SvcValidateRAFlow is used to check/validate RAFlow's struct
//------------------------------------------------------------------------------
func SvcValidateRAFlow(w http.ResponseWriter, r *http.Request, d *ServiceData) {
	const funcname = "SvcValidateRAFlow"
	var (
		err error
	)
	fmt.Printf("Entered %s\n", funcname)
	fmt.Printf("Request: %s:  BID = %d,  FlowID = %d\n", d.wsSearchReq.Cmd, d.BID, d.ID)

	switch d.wsSearchReq.Cmd {
	case "get":
		ValidateRAFlow(w, r, d)
		break
	default:
		err = fmt.Errorf("Unhandled command: %s", d.wsSearchReq.Cmd)
		SvcErrorReturn(w, err, funcname)
		return
	}
}

// ValidateRAFlow validate RAFlow's fields section wise
//-------------------------------------------------------------------------
func ValidateRAFlow(w http.ResponseWriter, r *http.Request, d *ServiceData) {
	const funcname = "ValidateRAFlow"
	fmt.Printf("Entered %s\n", funcname)

	var (
		err                error
		foo                RAFlowDetailRequest
		raFlowData         RAFlowJSONData
		raFlowFieldsErrors RAFlowFieldsErrors
		g                  ValidateRAFlowResponse
	)

	// http method check
	if r.Method != "POST" {
		err = fmt.Errorf("Only POST method is allowed")
		return
	}

	// unmarshal data into request data struct
	if err = json.Unmarshal([]byte(d.data), &foo); err != nil {
		return
	}

	// Init RAFlowFields error list
	raFlowFieldsErrors = RAFlowFieldsErrors{
		Dates: DatesFieldsError{
			Errors: map[string][]string{},
		},
		People:      []PeopleFieldsError{},
		Pets:        []PetFieldsError{},
		Vehicle:     []VehicleFieldsError{},
		Rentables:   []RentablesFieldsError{},
		ParentChild: []ParentChildFieldsError{},
		Tie: TieFieldsError{
			TiePeople: []TiePeopleFieldsError{},
		},
	}

	// Get flow information from the table to validate fields value
	flow, err := rlib.GetFlow(r.Context(), foo.FlowID)
	if err != nil {
		SvcErrorReturn(w, err, funcname)
		return
	}

	// When flowId doesn't exists in database return and give error that flowId doesn't exists
	if flow.FlowID == 0 {
		err = fmt.Errorf("flowID %d - doesn't exists", foo.FlowID)
		SvcErrorReturn(w, err, funcname)
		return
	}

	// get unmarshalled raflow data into struct
	err = json.Unmarshal(flow.Data, &raFlowData)
	if err != nil {
		SvcErrorReturn(w, err, funcname)
		return
	}

	// ---------------------------------------
	// Perform basic validation on RAFlow
	// ---------------------------------------
	// TODO(Akshay): Enable basic validation check
	g = basicValidateRAFlow(raFlowData, raFlowFieldsErrors)

	if g.Total > 0 {
		// If RAFlow structure have more than 1 basic validation error than it return with the list of basic validation errors
		SvcWriteResponse(d.BID, &g, w)
		return
	}

	// --------------------------------------------
	// Perform Bizlogic check validation on RAFlow
	// --------------------------------------------
	// validateRAFlowBizLogic(r.Context(), &raFlowData)
	// g.ErrorType = "biz"

}

// basicValidateRAFlow validate RAFlow's fields section wise
//-------------------------------------------------------------------------
func basicValidateRAFlow(raFlowData RAFlowJSONData, raFlowFieldsErrors RAFlowFieldsErrors) ValidateRAFlowResponse {

	var (
		datesFieldsErrors       DatesFieldsError
		peopleFieldsErrors      PeopleFieldsError
		petFieldsErrors         PetFieldsError
		vehicleFieldsErrors     VehicleFieldsError
		rentablesFieldsErrors   RentablesFieldsError
		raFeesErrors            RAFeesError
		parentChildFieldsErrors ParentChildFieldsError
		tieFieldsErrors         TieFieldsError
		tiePeopleFieldsErrors   TiePeopleFieldsError
		g                       ValidateRAFlowResponse
	)

	fmt.Println(raFlowData.Pets)

	//----------------------------------------------
	// validate RADatesFlowData structure
	// ----------------------------------------------
	// NOTE: Validation not require for the date type fields.
	// Because it handles while Unmarshalling string into rlib.JSONDate

	// call validation function
	errs := rtags.ValidateStructFromTagRules(raFlowData.Dates)

	// Modify error count for the response
	datesFieldsErrors.Total = len(errs)
	datesFieldsErrors.Errors = errs

	// Modify Total Error
	g.Total += datesFieldsErrors.Total

	// Assign dates fields error to
	raFlowFieldsErrors.Dates = datesFieldsErrors

	//----------------------------------------------
	// validate RAPeopleFlowData structure
	// ----------------------------------------------
	for _, people := range raFlowData.People {
		// call validation function
		errs := rtags.ValidateStructFromTagRules(people)

		// Modify error count for the response
		peopleFieldsErrors.Total = len(errs)
		peopleFieldsErrors.TMPTCID = people.TMPTCID
		peopleFieldsErrors.Errors = errs

		// Modify Total Error
		g.Total += peopleFieldsErrors.Total

		// Skip the row if it doesn't have error for the any fields
		if len(errs) == 0 {
			continue
		}

		raFlowFieldsErrors.People = append(raFlowFieldsErrors.People, peopleFieldsErrors)
	}

	// ----------------------------------------------
	// validate RAPetFlowData structure
	// ----------------------------------------------
	for _, pet := range raFlowData.Pets {

		// init raFeesErrors
		raFeesErrors := RAFeesError{
			Errors: map[string][]string{},
		}

		// call validation function
		errs := rtags.ValidateStructFromTagRules(pet)

		// Modify error count for the response
		petFieldsErrors.Total = len(errs)
		petFieldsErrors.TMPPETID = pet.TMPPETID
		petFieldsErrors.Errors = errs
		petFieldsErrors.FeesErrors = make([]RAFeesError, 0)

		fmt.Printf("Petfields error: %d\n", petFieldsErrors.Total)

		// ----------------------------------------------
		// validate RAPetFlowData.Fees structure
		// ----------------------------------------------
		for _, fee := range pet.Fees {
			// call validation function
			errs := rtags.ValidateStructFromTagRules(fee)

			raFeesErrors.Total = len(errs)
			raFeesErrors.TMPASMID = fee.TMPASMID
			raFeesErrors.Errors = errs

			// Modify pets error count
			petFieldsErrors.Total += raFeesErrors.Total

			// Skip the row if it doesn't have error for the any fields
			if len(errs) == 0 {
				continue
			}

			petFieldsErrors.FeesErrors = append(petFieldsErrors.FeesErrors, raFeesErrors)
		}

		// Modify total error
		g.Total += petFieldsErrors.Total

		// If there is no error in pet than skip that pet's error being added.
		if petFieldsErrors.Total == 0 {
			continue
		}

		raFlowFieldsErrors.Pets = append(raFlowFieldsErrors.Pets, petFieldsErrors)
	}

	// ----------------------------------------------
	// validate RAVehicleFlowData structure
	// ----------------------------------------------
	for _, vehicle := range raFlowData.Vehicles {

		// init raFeesErrors
		raFeesErrors := RAFeesError{
			Errors: map[string][]string{},
		}

		// call validation function
		errs := rtags.ValidateStructFromTagRules(vehicle)

		// Modify error count for the response
		vehicleFieldsErrors.Total = len(errs)
		vehicleFieldsErrors.TMPVID = vehicle.TMPVID
		vehicleFieldsErrors.Errors = errs
		vehicleFieldsErrors.FeesErrors = make([]RAFeesError, 0)

		// ----------------------------------------------
		// validate RAVehicleFlowData.Fees structure
		// ----------------------------------------------
		for _, fee := range vehicle.Fees {

			// call validation function
			errs := rtags.ValidateStructFromTagRules(fee)

			raFeesErrors.Total = len(errs)
			raFeesErrors.TMPASMID = fee.TMPASMID
			raFeesErrors.Errors = errs

			// Modify vehicle error count
			vehicleFieldsErrors.Total += raFeesErrors.Total

			// Skip the row if it doesn't have error for the any fields
			if len(errs) == 0 {
				continue
			}

			vehicleFieldsErrors.FeesErrors = append(vehicleFieldsErrors.FeesErrors, raFeesErrors)
		}

		// Modify Total Error
		g.Total += vehicleFieldsErrors.Total

		// If there is no error in vehicle than skip that vehicle's error being added.
		if vehicleFieldsErrors.Total == 0 {
			continue
		}

		raFlowFieldsErrors.Vehicle = append(raFlowFieldsErrors.Vehicle, vehicleFieldsErrors)
	}

	// ----------------------------------------------
	// validate RARentablesFlowData structure
	// ----------------------------------------------
	for _, rentable := range raFlowData.Rentables {
		// init raFeesErrors
		raFeesErrors = RAFeesError{
			Errors: map[string][]string{},
		}

		// call validation function
		errs := rtags.ValidateStructFromTagRules(rentable)

		// Modify error count for the response
		rentablesFieldsErrors.Total = len(errs)
		rentablesFieldsErrors.RID = rentable.RID
		rentablesFieldsErrors.Errors = errs
		rentablesFieldsErrors.FeesErrors = make([]RAFeesError, 0)

		// Modify Total Error
		g.Total += rentablesFieldsErrors.Total

		// ----------------------------------------------
		// validate RAVehicleFlowData.Fees structure
		// ----------------------------------------------
		for _, fee := range rentable.Fees {

			// call validation function
			errs := rtags.ValidateStructFromTagRules(fee)

			raFeesErrors.Total = len(errs)
			raFeesErrors.TMPASMID = fee.TMPASMID
			raFeesErrors.Errors = errs

			rentablesFieldsErrors.Total += raFeesErrors.Total

			// Skip the row if it doesn't have error for the any fields
			if len(errs) == 0 {
				continue
			}

			rentablesFieldsErrors.FeesErrors = append(rentablesFieldsErrors.FeesErrors, raFeesErrors)
		}

		// Modify Total Error
		g.Total += raFeesErrors.Total

		// If there is no error in vehicle than skip that rentable's error being added.
		if rentablesFieldsErrors.Total == 0 {
			continue
		}

		raFlowFieldsErrors.Rentables = append(raFlowFieldsErrors.Rentables, rentablesFieldsErrors)
	}

	// ----------------------------------------------
	// validate RAParentChildFlowData structure
	// ----------------------------------------------
	for _, parentChild := range raFlowData.ParentChild {
		// call validation function
		errs := rtags.ValidateStructFromTagRules(parentChild)

		// Skip the row if it doesn't have error for the any fields
		if len(errs) == 0 {
			continue
		}

		// Modify error count for the response
		parentChildFieldsErrors.Total = len(errs)
		parentChildFieldsErrors.PRID = parentChild.PRID
		parentChildFieldsErrors.Errors = errs

		// Modify Total Error
		g.Total += rentablesFieldsErrors.Total

		raFlowFieldsErrors.ParentChild = append(raFlowFieldsErrors.ParentChild, parentChildFieldsErrors)
	}

	// ----------------------------------------------
	// validate RATieFlowData.People structure
	// ----------------------------------------------
	for _, people := range raFlowData.Tie.People {
		// call validation function
		errs = rtags.ValidateStructFromTagRules(people)

		// Modify error count for the response
		tiePeopleFieldsErrors.Total = len(errs)
		tiePeopleFieldsErrors.TMPTCID = people.TMPTCID
		tiePeopleFieldsErrors.Errors = errs

		// Modify Total Error
		g.Total += tiePeopleFieldsErrors.Total

		tieFieldsErrors.TiePeople = append(tieFieldsErrors.TiePeople, tiePeopleFieldsErrors)
	}

	// Assign all(people/pet/vehicles) tie related error
	raFlowFieldsErrors.Tie = tieFieldsErrors

	//---------------------------------------
	// set the response
	//---------------------------------------
	g.Errors = raFlowFieldsErrors
	g.ErrorType = "basic"

	return g
}

// validateRAFlowBizLogic is to check RAFlow's business logic
func validateRAFlowBizLogic(ctx context.Context, a *RAFlowJSONData) error {
	const funcname = "ValidateRAFlowBizLogic"
	fmt.Printf("Entered %s\n", funcname)

	// ---------------------------------------------
	// Perform business logic check on date section
	// 1. Dates must be Jan 1, 2000 00:00:00 UTC or later.
	// ---------------------------------------------

	return nil
}
