"use strict";

const GRID = "expenseGrid";
const SIDEBAR_ID = "expense";
const FORM = "expenseForm";
const MODULE = "expense";

// Below configurations are in use while performing tests via roller_spec.js for AIR Roller application
// For Module: Deposit accounts
export let conf = {
    grid: GRID,
    form: FORM,
    sidebarID: SIDEBAR_ID,
    module: MODULE,
    capture: "expenseGridRequest.png",
    endPoint: "/{0}/expense/{1}",
    methodType: 'POST',
    requestData: JSON.stringify({"cmd": "get", "selected": [], "limit": 100, "offset": 0}),
    excludeGridColumns: [],
    buttonNamesInForm: ["save", "saveadd"],
    notVisibleButtonNamesInForm: ["close"],
    buttonNamesInDetailForm: ["save", "saveadd", "reverse"],
    skipColumns: ["Reversed"],
    skipFields: [],
    primaryId: "EXPID",
    haveDateValue: true,
    fromDate: new Date(2018, 1, 1), // year, month-1, day : 1st Feb 2018
    toDate: new Date(2018, 2, 1) // 1st March 2018
};

//TODO(Akshay): UI Test for Unpaid section(Green Color), White Color section, Find button in form
//TODO(Akshay): Handle From and To Date
