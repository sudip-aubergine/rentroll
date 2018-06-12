/* global
    RACompConfig, sliderContentDivLength, reassignGridRecids,
    getFullName, getTCIDName,
    hideSliderContent, appendNewSlider, showSliderContentW2UIComp,
    loadTargetSection, requiredFieldsFulFilled, getRAFlowPartTypeIndex, initRAFlowAjax,
    getRAFlowAllParts, saveActiveCompData, toggleHaveCheckBoxDisablity, getRAFlowCompData,
    openNewTransactantForm, getRAAddTransactantFormInitRec,
    acceptTransactant, loadRAPeopleForm,
    setRABGInfoFormHeader, showHideRABGInfoFormFields,
    setNotRequiredFields, getRATransanctantDetail, getRAPeopleGridRecord,
    updateRABGInfoFormCheckboxes, getRABGInfoFormInitRecord, loadRABGInfoForm, ReassignPeopleGridRecords,
    manageBGInfoFormFields, addDummyBackgroundInfo, savePeopleCompData, getPeopleLocalData, setPeopleLocalData,
    getPeopleLocalDataByTCID
*/

"use strict";

// -------------------------------------------------------------------------------
// Rental Agreement - People form, People Grid, Background information form
// -------------------------------------------------------------------------------
window.loadRAPeopleForm = function () {

    // if form is loaded then return
    if (!("RAPeopleForm" in w2ui)) {

        // people form
        $().w2form({
            name: 'RAPeopleForm',
            header: 'People',
            style: 'display: block; border: none;',
            formURL: '/webclient/html/formrapeople.html',
            focus: -1,
            fields: [
                {
                    name: 'Transactant', type: 'enum', required: true, html: {caption: "Transactant"},
                    options: {
                        url: '/v1/transactantstd/' + app.raflow.BID,
                        max: 1,
                        renderItem: function (item) {

                            // Enable Accept button
                            $(w2ui.RAPeopleForm.box).find("button[name=accept]").prop("disabled", false);

                            var s = getTCIDName(item);
                            w2ui.RAPeopleForm.record.TCID = item.TCID;
                            w2ui.RAPeopleForm.record.FirstName = item.FirstName;
                            w2ui.RAPeopleForm.record.LastName = item.LastName;
                            w2ui.RAPeopleForm.record.MiddleName = item.MiddleName;
                            w2ui.RAPeopleForm.record.Employer = item.Employer;
                            w2ui.RAPeopleForm.record.IsCompany = item.IsCompany;
                            return s;
                        },
                        renderDrop: function (item) {
                            return getTCIDName(item);
                        },
                        compare: function (item, search) {
                            var s = getTCIDName(item);
                            s = s.toLowerCase();
                            var srch = search.toLowerCase();
                            var match = (s.indexOf(srch) >= 0);
                            return match;
                        },
                        onRemove: function(event) {
                            event.onComplete = function() {
                                w2ui.RAPeopleForm.actions.reset();
                            };
                        }
                    }
                },
                {name: 'BID', type: 'int', required: true, html: {caption: "BID"}},
                {name: 'TCID', type: 'int', required: true, html: {caption: "TCID"}},
                {name: 'FirstName', type: 'text', required: false, html: {caption: "FirstName"}},
                {name: 'LastName', type: 'text', required: false, html: {caption: "LastName"}},
                {name: 'MiddleName', type: 'text', required: false, html: {caption: "MiddleName"}},
                {name: 'Employer', type: 'text', required: false, html: {caption: "Employer"}},
                {name: 'IsCompany', type: 'int', required: true, html: {caption: "IsCompany"}}
            ],
            actions: {
                reset: function () {
                    w2ui.RAPeopleForm.clear();
                    $(w2ui.RAPeopleForm.box).find("button[name=accept]").prop("disabled", true);
                }
            },
            onRefresh: function (event) {
                var f = this;
                event.onComplete = function () {
                    var BID = getCurrentBID(),
                        BUD = getBUDfromBID(BID);

                    f.record.BID = BID;
                };
            }
        });

        // transanctants/people list in grid
        $().w2grid({
            name: 'RAPeopleGrid',
            header: 'Background information',
            show: {
                toolbar: true,
                toolbarSearch: false,
                toolbarAdd: true,
                toolbarReload: true,
                toolbarInput: false,
                toolbarColumns: false,
                footer: true
            },
            style: 'border: 0px solid black; display: block;',
            multiSelect: false,
            columns: [
                {
                    field: 'recid',
                    hidden: true
                },
                {
                    field: 'TMPTCID',
                    hidden: true
                },
                {
                    field: 'TCID',
                    hidden: true
                },
                {
                    field: 'FullName',
                    caption: 'Name',
                    size: '100%',
                    style: 'text-align: left;',
                    render: function (record) {
                        if (record.IsCompany > 0) {
                            return record.Employer;
                        } else {
                            return getFullName(record);
                        }
                    }
                },
                {
                    field: 'IsRenter',
                    // caption: 'Renter',
                    // size: '100px',
                    hidden: true,
                    // render: function (record) {
                    //     if (record.IsRenter) {
                    //         return '<i class="fas fa-check" title="renter"></i>';
                    //     } else {
                    //         return '<i class="fas fa-times" title="renter"></i>';
                    //     }
                    // }
                },
                {
                    field: 'IsOccupant',
                    // caption: 'Occupant',
                    // size: '100px',
                    hidden: true,
                    // render: function (record) {
                    //     if (record.IsOccupant) {
                    //         return '<i class="fas fa-check" title="occupant"></i>';
                    //     } else {
                    //         return '<i class="fas fa-times" title="occupant"></i>';
                    //     }
                    // }
                },
                {
                    field: 'IsGuarantor',
                    // caption: 'Guarantor',
                    // size: '100px',
                    hidden: true,
                    // render: function (record) {
                    //     if (record.IsGuarantor) {
                    //         return '<i class="fas fa-check" title="guarantor"></i>';
                    //     } else {
                    //         return '<i class="fas fa-times" title="guarantor"></i>';
                    //     }
                    // }
                }
            ],
            onClick: function (event) {
                event.onComplete = function () {
                    var yes_args = [this, event.recid],
                        no_args = [this],
                        no_callBack = function (grid) {
                            grid.select(app.last.grid_sel_recid);
                            return false;
                        },
                        yes_callBack = function (grid, recid) {
                            var form = w2ui.RABGInfoForm;

                            app.last.grid_sel_recid = parseInt(recid);

                            // keep highlighting current row in any case
                            grid.select(app.last.grid_sel_recid);

                            var raBGInfoGridRecord = grid.get(event.recid); // record from the w2ui grid

                            // show slider content in w2ui comp
                            showSliderContentW2UIComp(form, RACompConfig.people.sliderWidth);

                            // show/hide list of fields based on role
                            manageBGInfoFormFields(raBGInfoGridRecord);

                            form.record = getPeopleLocalData(raBGInfoGridRecord.TMPTCID);
                            form.record.recid = raBGInfoGridRecord.recid;

                            // Set the form title
                            setRABGInfoFormHeader(form.record);

                            form.refresh(); // need to refresh for form changes
                        };

                    // warn user if form content has been changed
                    form_dirty_alert(yes_callBack, no_callBack, yes_args, no_args);
                };
            },
            onAdd: function () {
                openNewTransactantForm();
            }
        });

        // background info form
        $().w2form({
            name: 'RABGInfoForm',
            header: 'Background Information',
            style: 'border: 0px; background-color: transparent; display: block;',
            formURL: '/webclient/html/formrabginfo.html',
            toolbar: {
                items: [
                    {id: 'bt3', type: 'spacer'},
                    // {id: 'addInfo', type: 'button', icon: 'fas fa-plus-circle'}, // TODO: Remove this in production. This button is for development purpose
                    {id: 'btnClose', type: 'button', icon: 'fas fa-times'}
                ],
                onClick: function (event) {
                    switch (event.target) {
                        case 'btnClose':
                            hideSliderContent();
                            break;
                        case 'addInfo':
                            addDummyBackgroundInfo();
                            break;
                    }
                }
            },
            fields: [
                {name: 'recid',                     type: 'int',        required: true },
                {name: 'BID',                       type: 'int',        required: true,     html: {caption: 'BID', page: 0, column: 0}},
                {name: 'TMPTCID',                   type: 'int',        required: true },
                {name: 'TCID',                      type: 'int',        required: true,     html: {caption: 'TCID', page: 0, column: 0}},
                {name: 'IsRenter',                  type: 'checkbox',   required: false },  // will be responsible for paying rent
                {name: 'IsOccupant',                type: 'checkbox',   required: false },  // will reside in and/or use the items rented
                {name: 'IsGuarantor',               type: 'checkbox',   required: false },  // responsible for making sure all rent is paid
                {name: 'FirstName',                 type: 'text',       required: false },
                {name: 'MiddleName',                type: 'text',       required: false },
                {name: 'LastName',                  type: 'text',       required: false },
                {name: 'IsCompany',                 type: 'int',        required: true },
                {name: 'DateofBirth',               type: 'date',       required: false },  // Date of births of applicants
                {name: 'SSN',                       type: 'text',       required: false },  // Social security number of applicants
                {name: 'DriverLicNo',               type: 'text'},                          // Driving licence number of applicants
                {name: 'CellPhone',                 type: 'text',       required: false },  // Telephone no of applicants
                {name: 'PrimaryEmail',              type: 'email',      required: false },  // Email Address of applicants
                {name: 'CurrentAddress',            type: 'text',       required: false },  // Current Address
                {name: 'CurrentLandLordName',       type: 'text',       required: false },  // Current landlord's name
                {name: 'CurrentLandLordPhoneNo',    type: 'text',       required: false },  // Current landlord's phone number
                {name: 'CurrentLengthOfResidency',  type: 'int',        required: false },  // Length of residency at current address
                {name: 'CurrentReasonForMoving',    type: 'text',       required: false },  // Reason of moving from current address
                {name: 'PriorAddress',              type: 'text'},                          // Prior Address
                {name: 'PriorLandLordName',         type: 'text'},                          // Prior landlord's name
                {name: 'PriorLandLordPhoneNo',      type: 'text'},                          // Prior landlord's phone number
                {name: 'PriorLengthOfResidency',    type: 'int'},                           // Length of residency at Prior address
                {name: 'PriorReasonForMoving',      type: 'text'},                          // Reason of moving from Prior address
                {name: 'Evicted',                   type: 'checkbox',   required: false },  // have you ever been Evicted
                {name: 'EvictedDes',                type: 'text',       required: false },
                {name: 'Convicted',                 type: 'checkbox',   required: false },  // have you ever been Arrested or convicted of a crime
                {name: 'ConvictedDes',              type: 'text',       required: false },
                {name: 'Bankruptcy',                type: 'checkbox',   required: false },  // have you ever been Declared Bankruptcy
                {name: 'BankruptcyDes',             type: 'text',       required: false },
                {name: 'Employer',                  type: 'text',       required: false },
                {name: 'WorkPhone',                 type: 'text',       required: false },
                {name: 'Address',                   type: 'text',       required: false },
                {name: 'Address2',                  type: 'text',       required: false },
                {name: 'City',                      type: 'text',       required: false },
                {name: 'State',                     type: 'list',       required: false,    options: {items: app.usStateAbbr}, },
                {name: 'PostalCode',                type: 'text',       required: false },
                {name: 'Country',                   type: 'text',       required: false },
                {name: 'Position',                  type: 'text',       required: false },
                {name: 'GrossIncome',               type: 'money',      required: false },
                {name: 'Comment',                   type: 'text'},                          // In an effort to accommodate you, please advise us of any special needs
                {name: 'EmergencyContactName',      type: 'text',       required: false },  // Name of emergency contact
                {name: 'EmergencyContactPhone',     type: 'text',       required: false },  // Phone number of emergency contact
                {name: 'EmergencyContactAddress',   type: 'text',       required: false }   // Address of emergency contact
            ],
            actions: {
                save: function () {
                    var form = this,
                        TMPTCID = form.record.TMPTCID;

					var errors = form.validate();
					if (errors.length > 0) return;

					var peopleData = getFormSubmitData(form.record, true);

					// If transanctant role isn't selected than display error.
					if(!(peopleData.IsRenter || peopleData.IsOccupant || peopleData.IsGuarantor)){
						form.message("Please select transanctant role.");
						return;
					}

					// Convert integer to bool checkboxes fields
					updateRABGInfoFormCheckboxes(peopleData);

					setPeopleLocalData(TMPTCID, peopleData);

					// clean dirty flag of form
					app.form_is_dirty = false;

					// save this records in json Data
					savePeopleCompData()
						.done(function (data) {
							if (data.status === 'success') {

								form.clear();

								// update RAPeopleGrid
								ReassignPeopleGridRecords();

								// close the form
								hideSliderContent();
							} else {
								form.message(data.message);
							}
						})
						.fail(function (data) {
							console.log("failure " + data);
						});
				},
				delete: function () {
					var form = this;
					// get local data from TMPPETID
					var compData = getRAFlowCompData("people", app.raflow.activeFlowID) || [];
					var itemIndex = getPeopleLocalData(form.record.TMPTCID, true);
					compData.splice(itemIndex, 1);

					savePeopleCompData()
						.done(function (data) {
							if (data.status === 'success') {

								form.clear();

								// update RAPeopleGrid
								ReassignPeopleGridRecords();

								// close the form
								hideSliderContent();
							} else {
								form.message(data.message);
							}
						})
						.fail(function (data) {
							console.log("failure " + data);
						});
				},
				reset: function () {
					w2ui.RABGInfoForm.clear();
				}
			},
			onChange: function (event) {
				event.onComplete = function () {
					$("#EvictedDes").prop("disabled", !this.record.Evicted);
					$("#ConvictedDes").prop("disabled", !this.record.Convicted);
					$("#BankruptcyDes").prop("disabled", !this.record.Bankruptcy);

					manageBGInfoFormFields(this.record);

					this.refresh();

					// formRecDiffer: 1=current record, 2=original record, 3=diff object
					var diff = formRecDiffer(this.record, app.active_form_original, {});
					// if diff == {} then make dirty flag as false, else true
					if ($.isPlainObject(diff) && $.isEmptyObject(diff)) {
						app.form_is_dirty = false;
					} else {
						app.form_is_dirty = true;
					}
				};
			},
			onRefresh: function (event) {
				var form = this;
				event.onComplete = function() {
					// hide delete button if it is NewRecord
					var isNewRecord = (w2ui.RAPeopleGrid.get(form.record.recid, true) === null);
					if (isNewRecord) {
						$(form.box).find("button[name=delete]").addClass("hidden");
					} else {
						$(form.box).find("button[name=delete]").removeClass("hidden");
					}

					$("#EvictedDes").prop("disabled", !this.record.Evicted);
					$("#ConvictedDes").prop("disabled", !this.record.Convicted);
					$("#BankruptcyDes").prop("disabled", !this.record.Bankruptcy);
				};
			}
		});
    }

    // load form in div
    $('#ra-form #people .grid-container').w2render(w2ui.RAPeopleGrid);
    $('#ra-form #people .form-container').w2render(w2ui.RAPeopleForm);

    // load existing info in PeopleForm and PeopleGrid
    setTimeout(function () {
        // Operation on RAPeopleGrid
        ReassignPeopleGridRecords();
    }, 500);
};

// setRABGInfoFormHeader
// It set RABGInfoForm header title
window.setRABGInfoFormHeader = function (record) {
    if (record.IsCompany > 0) {
        w2ui.RABGInfoForm.header = 'Background Information - ' + record.Employer;
    } else {
        w2ui.RABGInfoForm.header = 'Background Information - ' + record.FirstName + ' ' + record.MiddleName + ' ' + record.LastName;
    }
};

// showHideRABGInfoFormFields
// hide fields if transanctant is only user
window.showHideRABGInfoFormFields = function (listOfHiddenFields, hidden) {
    if (hidden) {
        $("#cureentInfolabel").hide();
        $("#priorInfolabel").hide();
    } else {
        $("#cureentInfolabel").show();
        $("#priorInfolabel").show();
    }
    for (var fieldIndex = 0; fieldIndex < listOfHiddenFields.length; fieldIndex++) {
        w2ui.RABGInfoForm.get(listOfHiddenFields[fieldIndex]).hidden = hidden;
    }
};

// setNotRequiredFields
// define fields are not required if transanctant is only user
window.setNotRequiredFields = function (listOfNotRequiredFields, required) {
    for (var fieldIndex = 0; fieldIndex < listOfNotRequiredFields.length; fieldIndex++) {
        w2ui.RABGInfoForm.get(listOfNotRequiredFields[fieldIndex]).required = required;
    }
};

// getRATransanctantDetail
// get Transanctant detail from the server
window.getRATransanctantDetail = function (TCID) {
    var bid = getCurrentBID();

    // temporary data
    var data = {
        "TCID": TCID,
        "FlowID": app.raflow.activeFlowID
    };


    return $.ajax({
        url: "/v1/raflow-persondetails/" + bid.toString() + "/" + app.raflow.activeFlowID.toString(),
        method: "POST",
        contentType: "application/json",
        dataType: "json",
        data: JSON.stringify(data),
        success: function (data) {
            if (data.status != "error") {
                // update the local copy of flow for the active one
                app.raflow.data[data.record.FlowID] = data.record;
            } else {
                console.error(data.message);
            }
        },
        error: function () {
            console.log("Error:" + JSON.stringify(data));
        }
    });
};

// getRAPeopleGridRecord
// get record from the list which match with TCID
window.getRAPeopleGridRecord = function (records, TCID) {
    var raBGInfoGridrecord;
    for (var recordIndex = 0; recordIndex < records.length; recordIndex++) {
        if (records[recordIndex].TCID === TCID) {
            raBGInfoGridrecord = records[recordIndex];
            break;
        }
    }
    return raBGInfoGridrecord;
};

// updateRABGInfoFormCheckboxes
// Convert checkboxes w2ui int(1/0) value to bool(true/false)
window.updateRABGInfoFormCheckboxes = function (record) {
    record.IsRenter = int_to_bool(record.IsRenter);
    record.IsOccupant = int_to_bool(record.IsOccupant);
    record.IsGuarantor = int_to_bool(record.IsGuarantor);

    // record.IsCompany = int_to_bool(record.IsCompany);

    record.Evicted = int_to_bool(record.Evicted);
    record.Bankruptcy = int_to_bool(record.Bankruptcy);
    record.Convicted = int_to_bool(record.Convicted);
};

//
window.getRABGInfoFormInitRecord = function (BID, TCID, RECID) {

    return {
        recid: RECID,
        TMPTCID: 0,
        TCID: TCID,
        BID: BID,
        IsRenter: false,
        IsOccupant: true,
        IsGuarantor: false,
        FirstName: "",
        MiddleName: "",
        LastName: "",
        IsCompany: 0,
        DateofBirth: "",
        SSN: "",
        DriverLicNo: "",
        CellPhone: "",
        PrimaryEmail: "",
        CurrentAddress: "",
        CurrentLandLordName: "",
        CurrentLandLordPhoneNo: "",
        CurrentLengthOfResidency: 0,
        CurrentReasonForMoving: "",
        PriorAddress: "",
        PriorLandLordName: "",
        PriorLandLordPhoneNo: "",
        PriorLengthOfResidency: 0,
        PriorReasonForMoving: "",
        Evicted: false,
        EvictedDes: "",
        Convicted: false,
        ConvictedDes: "",
        Bankruptcy: false,
        BankruptcyDes: "",
        Employer: "",
        WorkPhone: "",
        Address: "",
        Address2: "",
        City: "",
        State: "",
        PostalCode: "",
        Country: "",
        Position: "",
        GrossIncome: 0,
        Comment: "",
        EmergencyContactName: "",
        EmergencyContactPhone: "",
        EmergencyContactAddress: ""
    };
};

//--------------------------------------------------------------------
// ReassignPeopleGridRecords
//--------------------------------------------------------------------
window.ReassignPeopleGridRecords = function () {
    var compData = getRAFlowCompData("people", app.raflow.activeFlowID);
    var grid = w2ui.RAPeopleGrid;

    if (compData) {
        grid.records = compData;
        reassignGridRecids(grid.name);

        // Operation on RAPeopleForm
        w2ui.RAPeopleForm.refresh();
    } else {
        // Operation on RAPeopleForm
        w2ui.RAPeopleForm.actions.reset();

        // Operation on RAPeopleGrid
        grid.clear();
    }
};

//-----------------------------------------------------------------------------
// openNewTransactantForm - popup new transactant form
//-----------------------------------------------------------------------------
window.openNewTransactantForm = function () {
    var BID = getCurrentBID(),
        BUD = getBUDfromBID(BID);

    // For new form TCID is 0
    var TCID = 0;
    var recid = w2ui.RAPeopleGrid.records.length + 1;

    w2ui.RABGInfoForm.header = 'Background Information';
    w2ui.RABGInfoForm.record = getRABGInfoFormInitRecord(BID, TCID, recid);

	setTransactantDefaultRole(w2ui.RABGInfoForm.record);

    showSliderContentW2UIComp(w2ui.RABGInfoForm, RACompConfig.people.sliderWidth);

    w2ui.RABGInfoForm.refresh(); // need to refresh for header changes
};

//-----------------------------------------------------------------------------
// acceptTransactant - add transactant to the list of payor/user/guarantor
//
// @params
//   item = an object assumed to have a FirstName, MiddleName, LastName,
//          IsCompany, and Employer.
// @return - the name to render
//-----------------------------------------------------------------------------
window.acceptTransactant = function () {

    var compData = getRAFlowCompData("people", app.raflow.activeFlowID) || [];

    var peopleForm = w2ui.RAPeopleForm;

    var transactantRec = $.extend(true, {}, peopleForm.record);
    delete transactantRec.Transactant;
    var TCID = transactantRec.TCID;

    var tcidIndex = getPeopleLocalDataByTCID(TCID, true);

    // if not found then push it in the data
    if (tcidIndex < 0) {

        // get transanctant information from the server
        getRATransanctantDetail(TCID)
        .done(function (data) {

            if (data.status === 'success') {

                // load item in the RAPeopleGrid grid
                ReassignPeopleGridRecords();

                // clear the form
                w2ui.RAPeopleForm.actions.reset();

            } else {
                console.log(data.message);
            }
        })
        .fail(function (data) {
            console.log("failure" + data);
        });
    } else {
        var recid = compData[tcidIndex].recid;

        // Show selected row for existing transanctant record
        w2ui.RAPeopleGrid.select(recid);

        // clear the form
        w2ui.RAPeopleForm.actions.reset();
    }

};

// manageBGInfoFormFields
window.manageBGInfoFormFields = function (record) {
    // Hide these all fields when transanctant is only user.
    var listOfHiddenFields = ["CurrentAddress", "CurrentLandLordName",
        "CurrentLandLordPhoneNo", "CurrentLengthOfResidency", "CurrentReasonForMoving",
        "PriorAddress", "PriorLandLordName", "PriorLandLordPhoneNo",
        "PriorLengthOfResidency", "PriorReasonForMoving"];

    // Display/Required field based on transanctant type
    var haveToHide = record.IsOccupant && !record.IsRenter && !record.IsGuarantor; // true: hide fields, false: show fields
    // hide/show fields
    showHideRABGInfoFormFields(listOfHiddenFields, haveToHide);
};

window.addDummyBackgroundInfo = function () {
    var form = w2ui.RABGInfoForm;
    var record = form.record;
    record.FirstName = Math.random().toString(32).slice(2);
    record.MiddleName = Math.random().toString(32).slice(2);
    record.LastName = Math.random().toString(32).slice(2);
    record.DateofBirth = "8/30/1990";
    record.SSN = Math.random().toString(32).slice(4);
    record.DriverLicNo = Math.random().toString(32).slice(2);
    record.CellPhone = Math.random().toString(32).slice(2);
    record.PrimaryEmail = Math.random().toString(32).slice(2) + "@yopmail.com";
    record.CurrentAddress = Math.random().toString(32).slice(2);
    record.CurrentLandLordName = Math.random().toString(32).slice(2);
    record.CurrentLandLordPhoneNo = Math.random().toString(32).slice(2);
    record.CurrentLengthOfResidency = 56;
    record.CurrentReasonForMoving = Math.random().toString(32).slice(2);
    record.PriorAddress = Math.random().toString(32).slice(2);
    record.PriorLandLordName = Math.random().toString(32).slice(2);
    record.PriorLandLordPhoneNo = Math.random().toString(32).slice(2);
    record.PriorLengthOfResidency = 36;
    record.PriorReasonForMoving = Math.random().toString(32).slice(2);
    record.Employer = Math.random().toString(32).slice(2);
    record.WorkPhone = Math.random().toString(32).slice(2);
    record.Address = Math.random().toString(32).slice(2);
    record.Position = Math.random().toString(32).slice(2);
    record.GrossIncome = Math.random() * 100;
    record.EmergencyContactName = Math.random().toString(32).slice(2);
    record.EmergencyContactPhone = Math.random().toString(32).slice(2);
    record.EmergencyContactAddress = Math.random().toString(32).slice(2);
    form.refresh();
};

//------------------------------------------------------------------------------
// savePetsCompData - saves the data on server side
//------------------------------------------------------------------------------
window.savePeopleCompData = function() {
	var compData = getRAFlowCompData("people", app.raflow.activeFlowID);
	return saveActiveCompData(compData, "people");
};

//-----------------------------------------------------------------------------
// getPeopleLocalDataByTCID - returns the clone of people data
//                            for requested TCID
//-----------------------------------------------------------------------------
window.getPeopleLocalDataByTCID = function(TCID, returnIndex) {
    var cloneData = {};
    var foundIndex = -1;
    var compData = getRAFlowCompData("people", app.raflow.activeFlowID) || [];
    compData.forEach(function(item, index) {
        if (item.TCID === TCID) {
            if (returnIndex) {
                foundIndex = index;
            } else {
                cloneData = $.extend(true, {}, item);
            }
            return false;
        }
    });
    if (returnIndex) {
        return foundIndex;
    }
    return cloneData;
};

//-----------------------------------------------------------------------------
// getPeopleLocalData - returns the clone of people data for requested TMPTCID
//-----------------------------------------------------------------------------
window.getPeopleLocalData = function(TMPTCID, returnIndex) {
	var cloneData = {};
	var foundIndex = -1;
	var compData = getRAFlowCompData("people", app.raflow.activeFlowID) || [];
	compData.forEach(function(item, index) {
		if (item.TMPTCID === TMPTCID) {
			if (returnIndex) {
				foundIndex = index;
			} else {
				cloneData = $.extend(true, {}, item);
			}
			return false;
		}
	});
	if (returnIndex) {
		return foundIndex;
	}
	return cloneData;
};

//-----------------------------------------------------------------------------
// setPeopleLocalData - save the data for requested a TMPTCID in local data
//-----------------------------------------------------------------------------
window.setPeopleLocalData = function(TMPTCID, peopleData) {
	var compData = getRAFlowCompData("people", app.raflow.activeFlowID) || [];
	var dataIndex = -1;
	compData.forEach(function(item, index) {
		if (item.TMPTCID === TMPTCID) {
			dataIndex = index;
			return false;
		}
	});
	if (dataIndex > -1) {
		compData[dataIndex] = peopleData;
	} else {
		compData.push(peopleData);
	}
};

//-----------------------------------------------------------------------------
// setTransactantDefaultRole - Assign default role for new transanctant.
//-----------------------------------------------------------------------------
window.setTransactantDefaultRole = function (transactantRec) {
	var compData = getRAFlowCompData("people", app.raflow.activeFlowID) || [];
	// If first record in the grid than transanctant will be renter by default
	if (compData.length === 0) {
		transactantRec.IsRenter = true;
	}

	// Each transactant must be occupant by default. It can be change via BGInfo detail form
	transactantRec.IsOccupant = true;
};
