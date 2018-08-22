package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"rentroll/bizlogic"
	"rentroll/rlib"
	"time"
)

// WriteHandlerContext contains context information for RA Write Handlers
type WriteHandlerContext struct {
	isNewOriginRaid bool  // true only if this is a new Rental Agreement, false otherwise
	oldRAID         int64 //
	newRAID         int64 //
	ra              rlib.RentalAgreement
	raOrig          rlib.RentalAgreement
	raf             rlib.RAFlowJSONData
	xbiz            rlib.XBusiness
}

// RAWriteHandler a handler function for part of the work of migrating
// RAFlow data into the permanent tables for a complete RentalAgreement
type RAWriteHandler struct {
	Name    string
	Handler func(context.Context, *WriteHandlerContext) error
}

// UpdateHandlers is the collection of routines to call to write a flow
// for an existing Rental Agreement back to the database as a RentalAgreement.
var ehandlers = []RAWriteHandler{
	{"Transactants", F2RAUpdatePeople},
	{"Pets", F2RAUpdatePets},
	{"Vehicles", F2RAUpdateVehicles},
	{"Rentables", FlowSaveRentables},
	{"Fees", Fees2RA},
}

// Flow2RA moves data from the Flow tabl into the permanent tables.
//
// INPUTS
//     ctx    - db context for transactions
//     flowid - the flow data to migrate into the permanent tables
//
// RETURNS
//     RAID of the newly created RentalAgreement
//     Any errors encountered
//-----------------------------------------------------------------------------
func Flow2RA(ctx context.Context, flowid int64) (int64, error) {
	rlib.Console("Entered Flow2RA\n")
	var x WriteHandlerContext
	var nraid int64

	//-------------------------------------------
	// Read the flow data into a data structure
	//-------------------------------------------
	flow, err := rlib.GetFlow(ctx, flowid)
	if err != nil {
		rlib.Console("\n\nERROR IN GetFlow: %s\n\n\n", err.Error())
		return nraid, err
	}
	err = json.Unmarshal(flow.Data, &x.raf)
	if err != nil {
		rlib.Console("\n\nERROR IN Unmarshal: %s\n\n\n", err.Error())
		return nraid, err
	}

	//----------------------------------------------------------------------------
	// If this is an update of an existing RAID, check to see if any changes
	// were made. Otherwise treat it as a new RAID
	//----------------------------------------------------------------------------
	x.isNewOriginRaid = x.raf.Meta.RAID == 0
	rlib.Console("isNewOriginRaid = %t\n", x.isNewOriginRaid)
	if !x.isNewOriginRaid { // update existing
		changes, err := rlib.RAFlowDataDiff(ctx, x.raf.Meta.RAID)
		if err != nil {
			rlib.Console("\n\nERROR IN FlowDataDIFF: %s\n\n\n", err.Error())
			return nraid, err
		}

		rlib.Console("\tData changes found = %t\n", changes)
		//-----------------------------------------------------------------------
		// If there were changes to the data, create an amended Rental Agreement
		//-----------------------------------------------------------------------
		if changes {
			x.newRAID, err = FlowSaveRA(ctx, &x) // this will update x.newRAID with the new raid
			if err != nil {
				rlib.Console("\n\nERROR IN FlowSaveRA: %s\n\n\n", err.Error())
				return x.newRAID, err
			}
		} else {
			err = fmt.Errorf("there are no data changes")
			return x.newRAID, err
		}
		//------------------------------------------------------------
		// if there are meta data changes, then updated existing RAID
		//------------------------------------------------------------
		rlib.Console("Just before call to FlowSaveMetaDataChanges: nraid = %d, x.newRAID = %d\n", nraid, x.newRAID)
		nraid, err = FlowSaveMetaDataChanges(ctx, &x)
		if err != nil {
			rlib.Console("\n\nERROR IN FlowSaveMetaDataChanges: %s\n\n\n", err.Error())
			return nraid, err
		}
		rlib.Console("\tMetaData data updated on RAID=%d\n", nraid)
	} else { // this is a new origin RA
		nraid, err = FlowSaveRA(ctx, &x)
		if err != nil {
			rlib.Console("\n\nERROR IN FlowSaveRA: %s\n\n\n", err.Error())
			return nraid, err
		}
		rlib.Console("New ORIGIN = %d\n", nraid)
		x.newRAID = nraid
		nraid, err = FlowSaveMetaDataChanges(ctx, &x)
		if err != nil {
			rlib.Console("\n\nERROR IN FlowSaveMetaDataChanges: %s\n\n\n", err.Error())
			return nraid, err
		}
		rlib.Console("\tMetaData data updated on RAID=%d\n", nraid)
	}

	// REMOVE FLOW IF MIGRATION DONE SUCCESSFULLY
	// Delete only if state is active or above active
	var state = x.ra.FLAGS & uint64(0xF)
	if state >= 4 && state <= 6 {
		err = rlib.DeleteFlow(ctx, flowid)
		if err != nil {
			return nraid, err
		}
	}

	rlib.Console("\tx.oldRAID = %d, x.newRAID = %d\n", x.oldRAID, x.newRAID)
	return x.newRAID, nil
}

// FlowSaveMetaDataChanges saves any change to the meta data in the flow with
//     the existing RAID
//
// INPUTS
//     ctx - db context for transactions
//     x - all the contextual info we need for performing this operation
//         Note: this routine adds ra and raOrig to x
//
// RETURNS
//     RAID of the Rental Agreement in which meta-data was changed.
//     Any errors encountered
//-----------------------------------------------------------------------------
func FlowSaveMetaDataChanges(ctx context.Context, x *WriteHandlerContext) (int64, error) {
	var err error
	raid := x.newRAID // update this one if changes were found and a new amendment was written.
	if raid == 0 {
		raid = x.raf.Meta.RAID // update this one if no changes were found
	}
	x.ra, err = rlib.GetRentalAgreement(ctx, raid)
	if err != nil {
		rlib.Ulog("Could not read rental agreement %d, err: %s\n", raid, err.Error())
		return raid, err
	}
	x.newRAID = raid // we'll update this one

	//----------------------------------------------------
	// compare the meta data and update if necessary
	//----------------------------------------------------
	changes := 0
	bterminated := x.ra.FLAGS&0xf == rlib.RASTATETerminated
	if x.ra.FLAGS != x.raf.Meta.RAFLAGS {
		//---------------------------------------------------------------------
		// If the FLAGs have changed, check to see if state of the permanent
		// table copy is in the Terminated state. If it is, do not change it
		// or the reason...
		//---------------------------------------------------------------------
		if bterminated {
			// DO NOTHING IF RA IS ALREADY TERMINATED
			// flags := x.ra.FLAGS
			// if (x.ra.FLAGS & ^uint64(0xf)) != (x.raf.Meta.RAFLAGS & ^uint64(0xf)) { // flags other than
			// 	state := x.ra.FLAGS & 0xf
			// 	x.ra.FLAGS = x.raf.Meta.RAFLAGS
			// 	x.ra.FLAGS &= ^unit64(0xf)
			// 	x.ra.FLAGS |= state
			// 	changes++
			// }
		} else {
			x.ra.FLAGS = x.raf.Meta.RAFLAGS
			changes++
		}
	}
	if x.ra.ApplicationReadyUID != x.raf.Meta.ApplicationReadyUID {
		changes++
		x.ra.ApplicationReadyUID = x.raf.Meta.ApplicationReadyUID
	}
	if !x.ra.ApplicationReadyDate.Equal(time.Time(x.raf.Meta.ApplicationReadyDate)) {
		changes++
		x.ra.ApplicationReadyDate = time.Time(x.raf.Meta.ApplicationReadyDate)
	}
	if x.ra.Approver1 != x.raf.Meta.Approver1 {
		changes++
		x.ra.Approver1 = x.raf.Meta.Approver1
	}
	if !x.ra.DecisionDate1.Equal(time.Time(x.raf.Meta.DecisionDate1)) {
		changes++
		x.ra.DecisionDate1 = time.Time(x.raf.Meta.DecisionDate1)
	}
	if x.ra.DeclineReason1 != x.raf.Meta.DeclineReason1 {
		changes++
		x.ra.DeclineReason1 = x.raf.Meta.DeclineReason1
	}
	if x.ra.Approver2 != x.raf.Meta.Approver2 {
		changes++
		x.ra.Approver2 = x.raf.Meta.Approver2
	}
	if !x.ra.DecisionDate2.Equal(time.Time(x.raf.Meta.DecisionDate2)) {
		changes++
		x.ra.DecisionDate2 = time.Time(x.raf.Meta.DecisionDate2)
	}
	if x.ra.DeclineReason2 != x.raf.Meta.DeclineReason2 {
		changes++
		x.ra.DeclineReason2 = x.raf.Meta.DeclineReason2
	}
	if x.ra.MoveInUID != x.raf.Meta.MoveInUID {
		changes++
		x.ra.MoveInUID = x.raf.Meta.MoveInUID
	}
	if !x.ra.MoveInDate.Equal(time.Time(x.raf.Meta.MoveInDate)) {
		changes++
		x.ra.MoveInDate = time.Time(x.raf.Meta.MoveInDate)
	}
	if x.ra.ActiveUID != x.raf.Meta.ActiveUID {
		changes++
		x.ra.ActiveUID = x.raf.Meta.ActiveUID
	}
	if !x.ra.ActiveDate.Equal(time.Time(x.raf.Meta.ActiveDate)) {
		changes++
		x.ra.ActiveDate = time.Time(x.raf.Meta.ActiveDate)
	}
	if x.ra.TerminatorUID != x.raf.Meta.TerminatorUID {
		changes++
		x.ra.TerminatorUID = x.raf.Meta.TerminatorUID
	}
	if !x.ra.TerminationDate.Equal(time.Time(x.raf.Meta.TerminationDate)) {
		changes++
		x.ra.TerminationDate = time.Time(x.raf.Meta.TerminationDate)
	}
	if x.ra.LeaseTerminationReason != x.raf.Meta.LeaseTerminationReason {
		changes++
		x.ra.LeaseTerminationReason = x.raf.Meta.LeaseTerminationReason
	}
	if !x.ra.DocumentDate.Equal(time.Time(x.raf.Meta.DocumentDate)) {
		changes++
		x.ra.DocumentDate = time.Time(x.raf.Meta.DocumentDate)
	}
	if x.ra.NoticeToMoveUID != x.raf.Meta.NoticeToMoveUID {
		changes++
		x.ra.NoticeToMoveUID = x.raf.Meta.NoticeToMoveUID
	}
	if !x.ra.NoticeToMoveDate.Equal(time.Time(x.raf.Meta.NoticeToMoveDate)) {
		changes++
		x.ra.NoticeToMoveDate = time.Time(x.raf.Meta.NoticeToMoveDate)
	}
	if !x.ra.NoticeToMoveReported.Equal(time.Time(x.raf.Meta.NoticeToMoveReported)) {
		changes++
		x.ra.NoticeToMoveReported = time.Time(x.raf.Meta.NoticeToMoveReported)
	}

	//---------------------------------------------------------
	// If there were any changes, update the Rental Agreement
	//---------------------------------------------------------
	rlib.Console("Metadata change count = %d\n", changes)
	if changes > 0 {
		rlib.Console("Updating RAID = %d\n", x.ra.RAID)
		err = rlib.UpdateRentalAgreement(ctx, &x.ra)
		if err != nil {
			return x.raf.Meta.RAID, err
		}
	}

	return x.newRAID, err
}

// FlowSaveRA saves a new Rental Agreement from the supplied flow. This
//     function assumes that a check has already been made to verify that
//     the RentalAgreement is either new or, if it is replacing an existing
//     rental agreement, that the data has actually been changed.
//
// INPUTS
//     ctx - db context for transactions
//     x - all the contextual info we need for performing this operation
//         Note: this routine adds ra and raOrig to x
//
// RETURNS
//     RAID of newly created Rental Agreement or updated Rental Agreement
//         if only meta-data was changed.
//     Any errors encountered
//-----------------------------------------------------------------------------
func FlowSaveRA(ctx context.Context, x *WriteHandlerContext) (int64, error) {
	// rlib.Console("Entered FlowSaveRA\n")
	var err error
	var nraid int64

	if err = rlib.InitBizInternals(x.raf.Meta.BID, &x.xbiz); err != nil {
		return nraid, err
	}

	if x.raf.Meta.RAID > 0 {
		//------------------------------------------------------------
		// Get the rental agreement that will be superceded by the
		// one we're creating here. Update its stop dates accordingly
		//------------------------------------------------------------
		x.oldRAID = x.raf.Meta.RAID
		x.raOrig, err = rlib.GetRentalAgreement(ctx, x.oldRAID)
		if err != nil {
			return nraid, err
		}

		// if err = rlib.InitBizInternals(x.raOrig.BID, &x.xbiz); err != nil {
		// 	return nraid, err
		// }
		// saveFlags := x.raOrig.FLAGS
		chgs := 0
		AStart := time.Time(x.raf.Dates.AgreementStart)
		RStart := time.Time(x.raf.Dates.RentStart)
		PStart := time.Time(x.raf.Dates.PossessionStart)
		if x.raOrig.AgreementStop.After(AStart) {
			x.raOrig.AgreementStop = AStart
			chgs++
		}
		if x.raOrig.RentStop.After(RStart) {
			x.raOrig.RentStop = RStart
			chgs++
		}
		if x.raOrig.PossessionStop.After(PStart) {
			x.raOrig.PossessionStop = PStart
			chgs++
		}

		//------------------------------------------------------------------
		// If there are changes, then we stop the old Rental Agreement and
		// create a new one linked to x.raOrig
		//------------------------------------------------------------------
		if chgs > 0 {
			x.raOrig.FLAGS &= ^uint64(0x7)           // clear the status
			x.raOrig.FLAGS |= rlib.RAActionTerminate // set the state to Terminated
			x.raOrig.LeaseTerminationReason =
				rlib.RRdb.BizTypes[x.raOrig.BID].Msgs.S[rlib.MSGRAUPDATED].SLSID // "Rental Agreement was updated"

			//--------------------------------------------------------------------------
			// In noauth mode, it still have tester session, get it from the context
			//--------------------------------------------------------------------------
			sess, ok := rlib.SessionFromContext(ctx)
			if !ok {
				return nraid, rlib.ErrSessionRequired
			}

			x.raOrig.TerminatorUID = sess.UID
			x.raOrig.TerminationDate = time.Now()

			err = rlib.UpdateRentalAgreement(ctx, &x.raOrig)
			if err != nil {
				return nraid, err
			}
		}

		//------------------------------------------------------------
		// Now start the new RAID.  Link it to x.raOrig
		//------------------------------------------------------------
		initRA(ctx, x)
		x.ra.PRAID = x.raOrig.RAID
		x.ra.ORIGIN = x.raOrig.ORIGIN
		x.ra.BID = x.raOrig.BID
		if x.raOrig.ORIGIN == 0 {
			x.ra.ORIGIN = x.raOrig.RAID
		}
		x.ra.RATID = x.raOrig.RATID
		x.ra.RentCycleEpoch = x.raOrig.RentCycleEpoch

	} else {
		//-------------------------------------
		// This is a new Rental Agreement...
		//-------------------------------------
		initRA(ctx, x)
	}

	nraid, err = rlib.InsertRentalAgreement(ctx, &x.ra)
	if err != nil {
		return nraid, err
	}
	x.newRAID = nraid
	//-----------------------------------------------------
	// Create a RentalAgreement Ledger marker
	//-----------------------------------------------------
	var lm = rlib.LedgerMarker{
		BID:     x.ra.BID,
		RAID:    x.newRAID,
		RID:     0,
		Dt:      x.ra.AgreementStart,
		Balance: float64(0),
		State:   rlib.LMINITIAL,
	}
	_, err = rlib.InsertLedgerMarker(ctx, &lm)
	if err != nil {
		return nraid, err
	}

	//---------------------------------------------------------------
	// Now spin through the series of handlers that move the data
	// into the permanent tables...
	//---------------------------------------------------------------
	for i := 0; i < len(ehandlers); i++ {
		// rlib.Console("FlowSaveRA: running handler %s\n", ehandlers[i].Name)
		if err = ehandlers[i].Handler(ctx, x); err != nil {
			rlib.Console("error returned from handler %s: %s\n", ehandlers[i].Name, err.Error())
			return nraid, err
		}
	}

	return nraid, nil
}

// initRA initializes a rental agreement structure with information from flow
// data. upon completion, x.ra will be filled out with basic information that
// can be pulled from x.raf
//
// INPUTS
//     ctx - db context for transactions
//     x - all the contextual info we need for performing this operation
//         Note: this routine adds ra and raOrig to x
//
// RETURNS
//     nothing at this time
//-----------------------------------------------------------------------------
func initRA(ctx context.Context, x *WriteHandlerContext) {
	x.ra.PRAID = int64(0)
	x.ra.ORIGIN = int64(0)
	x.ra.BID = x.raf.Meta.BID
	x.ra.AgreementStart = time.Time(x.raf.Dates.AgreementStart)
	x.ra.AgreementStop = time.Time(x.raf.Dates.AgreementStop)
	x.ra.RentStart = time.Time(x.raf.Dates.RentStart)
	x.ra.RentStop = time.Time(x.raf.Dates.RentStop)
	x.ra.PossessionStart = time.Time(x.raf.Dates.PossessionStart)
	x.ra.PossessionStop = time.Time(x.raf.Dates.PossessionStop)
	x.ra.CSAgent = x.raf.Dates.CSAgent
	x.ra.FLAGS = x.raf.Meta.RAFLAGS
	x.ra.Approver1 = x.raf.Meta.Approver1
	x.ra.DeclineReason1 = x.raf.Meta.DeclineReason1
	x.ra.DecisionDate1 = time.Time(x.raf.Meta.DecisionDate1)
	x.ra.Approver2 = x.raf.Meta.Approver2
	x.ra.DeclineReason2 = x.raf.Meta.DeclineReason2
	x.ra.DecisionDate2 = time.Time(x.raf.Meta.DecisionDate2)
	x.ra.CSAgent = x.raf.Dates.CSAgent
	x.ra.NoticeToMoveDate = time.Time(x.raf.Meta.NoticeToMoveDate)
	x.ra.NoticeToMoveReported = time.Time(x.raf.Meta.NoticeToMoveReported)
	x.ra.TerminatorUID = x.raf.Meta.TerminatorUID
	x.ra.TerminationDate = time.Time(x.raf.Meta.TerminationDate)
	// x.ra.FollowUpDate = time.Time(x.raf.Meta.FollowUpDate)
	// x.ra.Outcome = x.raf.Meta.Outcome
	// x.ra.NoticeToMoveUID = x.raf.Meta.NoticeToMoveUID
	// x.ra.OtherPreferences = x.raf.Meta.OtherPreferences
}

// FlowSaveRentables adds/updates rentables from the flow data.  This means
// that we update or add the RentalAgreementRentables list.  Update means
// that we set the stop date for the existing RentalAgreementRentables RAID.
// Then we add the Rentables in x.raf.Rentables[] into a
// RentalAgreementRentables for the new RAID
//
// INPUTS
//     ctx - db context for transactions
//     x - all the contextual info we need for performing this operation
//         Note: this routine adds ra and raOrig to x
//
// RETURNS
//     RAID of newly created Rental Agreement
//     Any errors encountered
//-----------------------------------------------------------------------------
func FlowSaveRentables(ctx context.Context, x *WriteHandlerContext) error {
	// rlib.Console("Entered FlowSaveRentables\n")
	//----------------------------------------------------------------
	// Update the stop date on any existing RentalAgreementRentables
	//----------------------------------------------------------------
	if x.raf.Meta.RAID > 0 {
		rarl, err := rlib.GetAllRentalAgreementRentables(ctx, x.raf.Meta.RAID)
		if err != nil {
			return err
		}
		for _, v := range rarl {
			v.RARDtStop = time.Time(x.raf.Dates.AgreementStart)
			if err = rlib.UpdateRentalAgreementRentable(ctx, &v); err != nil {
				return err
			}
			//----------------------------------------------------------------
			// Fix up the users
			//----------------------------------------------------------------
			rul, err := rlib.GetRentableUsersInRange(ctx, v.RID, &x.raOrig.PossessionStart, &x.ra.PossessionStop)
			if err != nil {
				return err
			}
			for _, ru := range rul {
				ru.DtStop = x.raOrig.PossessionStop
				if err = rlib.UpdateRentableUser(ctx, &ru); err != nil {
					return err
				}
			}
		}
		//----------------------------------------------------------------
		// Fix up the payors
		//----------------------------------------------------------------
		t, err := rlib.GetRentalAgreementPayorsByRAID(ctx, x.raf.Meta.RAID)
		if err != nil {
			return err
		}
		for _, v := range t {
			if v.DtStop.After(x.ra.RentStart) {
				v.DtStop = x.ra.RentStart
				if err = rlib.UpdateRentalAgreementPayor(ctx, &v); err != nil {
					return err
				}
			}
		}
	}

	//----------------------------------------------------------------
	// Add a RentalAgreementRentable entry for each Rentable
	//----------------------------------------------------------------
	for _, v := range x.raf.Rentables {
		var rar = rlib.RentalAgreementRentable{
			RAID:         x.ra.RAID,
			BID:          x.ra.BID,
			RID:          v.RID,
			CLID:         0, // TODO:
			ContractRent: 0, // TODO:
			RARDtStart:   time.Time(x.raf.Dates.PossessionStart),
			RARDtStop:    time.Time(x.raf.Dates.PossessionStop),
		}
		_, err := rlib.InsertRentalAgreementRentable(ctx, &rar)
		if err != nil {
			return err
		}

		//----------------------------------------------------------------
		// Add the users
		//----------------------------------------------------------------
		for _, v1 := range x.raf.People {
			var a = rlib.RentableUser{
				RID:     v.RID,
				BID:     x.ra.BID,
				TCID:    v1.TCID,
				DtStart: x.ra.PossessionStart,
				DtStop:  x.ra.PossessionStop,
			}
			if _, err := rlib.InsertRentableUser(ctx, &a); err != nil {
				return err
			}
		}
	}
	//----------------------------------------------------------------
	// Add the payers
	//----------------------------------------------------------------
	for _, v := range x.raf.People {
		var a = rlib.RentalAgreementPayor{
			RAID:    x.ra.RAID,
			BID:     x.ra.BID,
			TCID:    v.TCID,
			DtStart: x.ra.PossessionStart,
			DtStop:  x.ra.PossessionStop,
			FLAGS:   0,
		}
		if _, err := rlib.InsertRentalAgreementPayor(ctx, &a); err != nil {
			return err
		}
	}

	return nil
}

// F2RAUpdatePets updates all pets. If the pet already exists then
// it just updates the pet. If the pet is
//
// INPUTS
//     ctx    - db context for transactions
//     x - all the contextual info we need for performing this operation
//
// RETURNS
//     Any errors encountered
//-----------------------------------------------------------------------------
func F2RAUpdatePets(ctx context.Context, x *WriteHandlerContext) (err error) {
	// rlib.Console("Entered F2RAUpdatePets\n")

	for i := 0; i < len(x.raf.Pets); i++ {
		// get contact person
		var petTCID int64
		petTCID, err = GetTCIDForTMPTCID(x, x.raf.Pets[i].TMPTCID)
		if err != nil {
			return err
		}

		// PET ENTRY
		var pet rlib.Pet
		var bind rlib.TBind

		// updated tbind will be from new raid start time and for all future time.
		bind.SourceElemType = rlib.ELEMPERSON
		bind.SourceElemID = petTCID
		bind.AssocElemType = rlib.ELEMPET
		bind.AssocElemID = x.raf.Pets[i].PETID
		bind.DtStart = x.ra.PossessionStart
		bind.DtStop = rlib.ENDOFTIME

		// If PET exists then update it
		if x.raf.Pets[i].PETID > 0 {
			pet, err = rlib.GetPet(ctx, x.raf.Pets[i].PETID)
			if err != nil {
				return err
			}

			//-----------------------------------------------------------------
			// update it if necessary
			//-----------------------------------------------------------------
			chgs := 0
			if pet.RAID != x.ra.RAID {
				pet.RAID = x.ra.RAID
				chgs++
			}
			if pet.Type != x.raf.Pets[i].Type {
				pet.Type = x.raf.Pets[i].Type
				chgs++
			}
			if pet.Breed != x.raf.Pets[i].Breed {
				pet.Breed = x.raf.Pets[i].Breed
				chgs++
			}
			if pet.Color != x.raf.Pets[i].Color {
				pet.Color = x.raf.Pets[i].Color
				chgs++
			}
			if pet.Weight != x.raf.Pets[i].Weight {
				pet.Weight = x.raf.Pets[i].Weight
				chgs++
			}
			if pet.Name != x.raf.Pets[i].Name {
				pet.Name = x.raf.Pets[i].Name
				chgs++
			}

			// TODO(Steve): ONCE WE HAVE THE WORKING FUNCTIONALITY OF "TBIND" TABLE
			//              WITH DB QUERIES, UPDATE PET ACCORDINGLY
			//
			//              IF TCID IS CHANGED THEN MARK THE PET'S STOP DATE
			//              AS CURRENT DATE IN "TBIND" ENTRY AND CREATE A NEW ENTRY
			//              IN "TBIND" WITH EXISTING PET AND TCID ASSOCIATION WITH START
			//              DATE SAME AS WE ASSIGNED IN STOP DATE (CURRENT DATE)
			//
			//              AS OF NOW, JUST UPDATE THE PET INFO, EVENTUALLY
			//              THIS UPDATE PART WILL BE REMOVED AND NEXT BLOCK OF
			//              COMMENTED CODE WILL BE MODIFIED.
			// if err = rlib.UpdatePet(ctx, &newpet); err != nil {
			// 	return err
			// }

			//----------------------------------------------------------
			// Is the new Agreement period AFTER the current period?
			//----------------------------------------------------------
			// newStart := time.Time(x.raf.Pets[i].DtStart)
			// //newStop := time.Time(x.raf.Pets[i].DtStop)
			// if pet.DtStart.Before(newStart) {
			// 	//----------------------------------------------------------
			// 	// stop the current Pet when the RA begins...
			// 	//----------------------------------------------------------
			// 	pet.DtStop = newStart
			// 	if pet.DtStop.Before(pet.DtStart) { // yes, this can happen
			// 		pet.DtStop = pet.DtStart
			// 	}
			// 	if err = rlib.UpdatePet(ctx, &pet); err != nil {
			// 		return err
			// 	}
			// 	//----------------------------------------------------------
			// 	// now create the new one...
			// 	//----------------------------------------------------------
			// 	x.raf.Pets[i].PETID, err = rlib.InsertPet(ctx, &newpet)
			// 	if err != nil {
			// 		return err
			// 	}
			// } else {
			// 	//---------------------------------------------------------------
			// 	// if dates are the same or newpet range precedes old pet range,
			// 	// then newpet overwrites pet.  Set the old one to no time...
			// 	//---------------------------------------------------------------
			// 	pet.DtStop = pet.DtStart // essentially sets it to no time
			// 	if err = rlib.UpdatePet(ctx, &pet); err != nil {
			// 		return err
			// 	}
			// }

			//----------------------------------------------------------------
			// 1. If the TCID did not change from the one in the TBind that
			//    overlaps the amend RAID start time, then no action is taken
			//    to the TBinds for this pet.
			//
			// 2. For the TBind that overlaps the amended RAID start time,
			//    update its DtStop to the start of the amended RAID.
			//
			// 3. Remove any other TBinds for this pet in the future.
			//
			// 4. Create a new TBind for this pet where DtStart = start of
			//    the amended RAID and DtStop is set to EOT (end of time).
			//----------------------------------------------------------------

			//----------------------------------------------------------
			// Get the TBinds for the pets impacted by this amendment.
			// Source type = PERSON,  Assoc type = PET
			//----------------------------------------------------------
			var tblist []rlib.TBind
			tblist, err = rlib.GetTBindAssocsByRange(ctx, pet.BID, rlib.ELEMPET, petTCID, rlib.ELEMPERSON, &bind.DtStart, &bind.DtStop)
			if err != nil {
				return err
			}
			//----------------------------------------------------
			// if only 1 and person hasn't changed, we're done
			//----------------------------------------------------
			if len(tblist) == 1 && tblist[0].SourceElemType == rlib.ELEMPERSON && tblist[0].SourceElemID == petTCID {
				return nil
			}
			//-------------------------------------------------------------
			// Spin throught the records, update the overlapping record,
			// and remove the rest.
			//-------------------------------------------------------------
			for _, tb := range tblist {
				if rlib.DateInRange(&bind.DtStart, &tb.DtStart, &tb.DtStop) { // overlaps amended RAID ??
					tb.DtStop = bind.DtStart // YES: update its stop date
					if err = rlib.UpdateTBind(ctx, &tb); err != nil {
						return err
					}
				} else {
					if err = rlib.DeleteTBind(ctx, tb.TBID); err != nil { // NO: delete it
						return err
					}
				}
			}
		} else {
			rlib.MigrateStructVals(&x.raf.Pets[i], &pet)
			pet.TCID = petTCID
			pet.BID = x.raf.Meta.BID

			// TODO: remove these soon, they are deprecated
			pet.DtStart = bind.DtStart
			pet.DtStop = bind.DtStop
			pet.RAID = x.ra.RAID

			x.raf.Pets[i].PETID, err = rlib.InsertPet(ctx, &pet)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// F2RAUpdateVehicles updates all pets
//
// INPUTS
//     ctx    - db context for transactions
//     x - all the contextual info we need for performing this operation
//
// RETURNS
//     Any errors encountered
//-----------------------------------------------------------------------------
func F2RAUpdateVehicles(ctx context.Context, x *WriteHandlerContext) error {
	// rlib.Console("Entered F2RAUpdateVehicles\n")
	for i := 0; i < len(x.raf.Vehicles); i++ {

		// get contact person
		vehicleTCID, err := GetTCIDForTMPTCID(x, x.raf.Vehicles[i].TMPTCID)
		if err != nil {
			return err
		}

		var vehicle rlib.Vehicle

		//-------------------------------
		// handle existing vehicles...
		//-------------------------------
		if x.raf.Vehicles[i].VID > 0 {
			err := rlib.GetVehicle(ctx, x.raf.Vehicles[i].VID, &vehicle)
			if err != nil {
				return err
			}

			newvehicle := vehicle

			rlib.MigrateStructVals(&x.raf.Vehicles[i], &newvehicle)
			newvehicle.TCID = vehicleTCID
			newvehicle.BID = x.raf.Meta.BID

			// TODO(Steve): ONCE WE HAVE THE WORKING FUNCTIONALITY OF "TBIND" TABLE
			//              WITH DB QUERIES, UPDATE VEHICLE ACCORDINGLY
			//
			//              IF TCID IS CHANGED THEN MARK THE VEHICLE'S STOP DATE
			//              AS CURRENT DATE IN "TBIND" ENTRY AND CREATE A NEW ENTRY
			//              IN "TBIND" WITH EXISTING VEHICLE AND TCID ASSOCIATION WITH START
			//              DATE SAME AS WE ASSIGNED IN STOP DATE (CURRENT DATE)
			//
			//              AS OF NOW, JUST UPDATE THE VEHICLE INFO, EVENTUALLY
			//              THIS UPDATE PART WILL BE REMOVED AND NEXT BLOCK OF
			//              COMMENTED CODE WILL BE MODIFIED.
			if err = rlib.UpdateVehicle(ctx, &newvehicle); err != nil {
				return err
			}

			/****** MODIFY CODE STARTS ******
			newStart := time.Time(x.raf.Vehicles[i].DtStart)
			//newStop := time.Time(x.raf.Pets[i].DtStop)
			if vehicle.DtStart.Before(newStart) {
				//----------------------------------------------------------
				// stop the current Vehicle when the RA begins...
				//----------------------------------------------------------
				vehicle.DtStop = newStart
				if vehicle.DtStop.Before(vehicle.DtStart) { // yes, this can happen
					vehicle.DtStop = vehicle.DtStart
				}
				if err = rlib.UpdateVehicle(ctx, &vehicle); err != nil {
					return err
				}
				//----------------------------------------------------------
				// now create the new one...
				//----------------------------------------------------------
				x.raf.Vehicles[i].VID, err = rlib.InsertVehicle(ctx, &newvehicle)
				if err != nil {
					return err
				}
			} else {
				//---------------------------------------------------------------
				// if dates are the same or newvehicle range precedes old vehicle range,
				// then newvehicle overwrites vehicle.  Set the old one to no time...
				//---------------------------------------------------------------
				vehicle.DtStop = vehicle.DtStart // essentially sets it to no time
				if err = rlib.UpdateVehicle(ctx, &vehicle); err != nil {
					return err
				}
				//---------------------------------------------------------------
				// now create the new one...
				//---------------------------------------------------------------
				x.raf.Vehicles[i].VID, err = rlib.InsertVehicle(ctx, &newvehicle)
				if err != nil {
					return err
				}
			}

			// rlib.Console("Just before UpdateVehicle: vehicles[j] = %#v\n", vehicles[j])
			if err = rlib.UpdateVehicle(ctx, &vehicle); err != nil {
				return err
			}
			//---------------------------------
			****** MODIFY CODE ENDS ******/
		} else {
			//-------------------------------
			// handle new vehicles...
			//-------------------------------
			rlib.MigrateStructVals(&x.raf.Vehicles[i], &vehicle)
			vehicle.TCID = vehicleTCID
			vehicle.BID = x.raf.Meta.BID
			// vehicle.RAID = x.ra.RAID

			x.raf.Vehicles[i].VID, err = rlib.InsertVehicle(ctx, &vehicle)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetTCIDForTMPTCID finds the TCID associated with the supplied tmptcid.
//
// INPUTS
//     ctx     - db context for transactions
//     x       - all the contextual info we need for performing this operation
//     TMPTCID - TMPTCID for person we want the associated RID
//
// RETURNS
//     TCID of associated Transactant, or -1 if not found
//-----------------------------------------------------------------------------
func GetTCIDForTMPTCID(x *WriteHandlerContext, TMPTCID int64) (TCID int64, err error) {

	for i := 0; i < len(x.raf.People); i++ {
		if x.raf.People[i].TMPTCID == TMPTCID {
			TCID = x.raf.People[i].TCID
			break
		}
	}

	if 0 == TCID {
		return TCID, fmt.Errorf("No TCID found for TMPTCID = %d", TMPTCID)
	}
	return TCID, nil
}

// F2RAUpdatePeople adds or updates all people information.
//
// INPUTS
//     ctx    - db context for transactions
//     x - all the contextual info we need for performing this operation
//
// RETURNS
//     Any errors encountered
//-----------------------------------------------------------------------------
func F2RAUpdatePeople(ctx context.Context, x *WriteHandlerContext) error {
	var err error
	// rlib.Console("Entered F2RAUpdatePeople\n")

	//-------------------------------------------------------------------
	// Spin through all the people and update or create as needed
	//-------------------------------------------------------------------
	for i := 0; i < len(x.raf.People); i++ {
		var xp rlib.XPerson
		tcid := x.raf.People[i].TCID
		// rlib.Console("Found persond: TMPTCID = %d, TCID = %d\n", x.raf.People[i].TMPTCID, tcid)
		if tcid > 0 {
			//---------------------------
			// Update existing...
			//---------------------------
			if err = rlib.GetXPerson(ctx, tcid, &xp); err != nil {
				return err
			}
			rlib.MigrateStructVals(&x.raf.People[i], &xp.Trn)
			rlib.MigrateStructVals(&x.raf.People[i], &xp.Usr)
			rlib.MigrateStructVals(&x.raf.People[i], &xp.Psp)
			rlib.MigrateStructVals(&x.raf.People[i], &xp.Pay)
			if err = rlib.UpdateTransactant(ctx, &xp.Trn); nil != err {
				return err
			}
			if err = rlib.UpdateUser(ctx, &xp.Usr); nil != err {
				return err
			}
			if err = rlib.UpdatePayor(ctx, &xp.Pay); nil != err {
				return err
			}
			if err = rlib.UpdateProspect(ctx, &xp.Psp); nil != err {
				return err
			}
		} else {
			//---------------------------
			// Create new Transactant...
			//---------------------------
			rlib.MigrateStructVals(&x.raf.People[i], &xp.Trn)
			rlib.MigrateStructVals(&x.raf.People[i], &xp.Usr)
			rlib.MigrateStructVals(&x.raf.People[i], &xp.Psp)
			rlib.MigrateStructVals(&x.raf.People[i], &xp.Pay)
			tcid, err := rlib.InsertTransactant(ctx, &xp.Trn)
			if nil != err {
				return err
			}
			if tcid == 0 {
				return fmt.Errorf("Insert returned a 0 id")
			}
			x.raf.People[i].TCID = tcid
			xp.Trn.TCID = tcid
			xp.Usr.TCID = tcid
			xp.Usr.BID = xp.Trn.BID
			xp.Pay.TCID = tcid
			xp.Pay.BID = xp.Trn.BID
			xp.Psp.TCID = tcid
			xp.Psp.BID = xp.Trn.BID
			_, err = rlib.InsertUser(ctx, &xp.Usr)
			if nil != err {
				return err
			}
			_, err = rlib.InsertPayor(ctx, &xp.Pay)
			if nil != err {
				return err
			}
			_, err = rlib.InsertProspect(ctx, &xp.Psp)
			if nil != err {
				return err
			}
			errlist := bizlogic.FinalizeTransactant(ctx, &xp.Trn)
			if len(errlist) > 0 {
				return bizlogic.BizErrorListToError(errlist)
			}
		}
	}
	return nil
}

//
// // F2RAUpdateRA creates a new rental agreement and links it to its
// // parent
// //
// // INPUTS
// //     ctx    - db context for transactions
// //     x - all the contextual info we need for performing this operation
// //
// // RETURNS
// //     Any errors encountered
// //-----------------------------------------------------------------------------
// func F2RAUpdateRA(ctx context.Context, x *WriteHandlerContext) error {
// 	// var err error
// 	rlib.Console("Entered F2RAUpdateRA\n")
// 	return nil
// }
