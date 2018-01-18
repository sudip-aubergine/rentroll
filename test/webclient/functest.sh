#!/bin/bash

TOP=../..
RRBIN=${TOP}/tmp/rentroll
RSD="-rsd ${RRBIN}"

TESTNAME="Cypress UI Test"
TESTSUMMARY="UI Testing with cypress"

# do not create new db
CREATENEWDB=0

source ../share/base.sh

#--------------------------------------------------------------------
#  Use the testdb for these tests... (dbgen with db4.json, as of now)
#--------------------------------------------------------------------

# server with noauth
RENTROLLSERVERAUTH="-noauth"

# run cypress test with only roller_spec.js with videoRecording false as of now
doCypressUITest "a" "--config videoRecording=false --spec ./cypress/integration/roller_spec.js" "CypressUITesting"

logcheck
