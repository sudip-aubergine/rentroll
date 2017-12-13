"use strict";

var GRID = "pmtsGrid";
var SIDEBAR_ID = "pmts";
var FORM = "pmtForm";

exports.gridConf = {
    grid: GRID,
    sidebarID: SIDEBAR_ID,
    tableName: "PaymentType",
    capture: "pmtsGridRequest.png",
    testCount: 20
};

exports.formConf = {
    grid: GRID,
    form: FORM,
    sidebarID: SIDEBAR_ID,
    row: "0",
    capture: "pmtsFormRequest.png",
    captureAfterClosingForm: "pmtsFormRequestAfterClosingForm.png",
    buttonName: ["save", "saveadd", "delete"],
    testCount: 5
};

exports.addNewConf = {
  grid: GRID,
  form: FORM,
  sidebarID: SIDEBAR_ID,
  capture: "pmtsAddNewButton.png",
  buttonName: ["save", "saveadd"],
  testCount: 12
};



