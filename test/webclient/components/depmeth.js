"use strict";

var GRID = "depmethGrid";
var SIDEBAR_ID = "depmeth";
var FORM = "depmethForm";

exports.gridConf = {
    grid: "depmethGrid",
    sidebarID: "depmeth",
    capture: "depmethGridRequest.png"
};

exports.formConf = {
    grid: "depmethGrid",
    form: "depmethForm",
    sidebarID: "depmeth",
    row: "0",
    capture: "depmethFormRequest.png",
    captureAfterClosingForm: "depmethFormRequestAfterClosingForm.png",
    buttonName: ["save", "saveadd", "delete"],
    testCount: 5
};

exports.addNewConf = {
    grid: GRID,
    form: FORM,
    sidebarID: SIDEBAR_ID,
    capture: "depmethAddNewButton.png",
    buttonName: ["save", "saveadd"],
    inputSelectField: [],
    checkboxes: [],
    testCount: 8
};
