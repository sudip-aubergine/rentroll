package ws

// These are general utilty routines to support w2ui grid components.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"rentroll/bizlogic"
	"rentroll/rlib"
	"strings"
	"time"
	"tws"
)

// SvcStatus is the generalized error structure to return errors to the grid widget
type SvcStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// SvcStatusResponse is the response to return status when no other data
// needs to be returned
type SvcStatusResponse struct {
	Status string `json:"status"` // typically "success"
	Recid  int64  `json:"recid"`  // set to id of newly inserted record
}

// ServiceHandler describes the handler for all services
type ServiceHandler struct {
	Cmd         string
	Handler     func(http.ResponseWriter, *http.Request, *ServiceData)
	NeedBiz     bool // true if the command requires a BID
	NeedSession bool // true if this command requires a session
}

// GenSearch describes a search condition
type GenSearch struct {
	Field    string `json:"field"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	Operator string `json:"operator"`
}

// ColSort is what the UI uses to indicate how the return values should be sorted
type ColSort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// WebGridSearchRequestJSON is a struct suitable for describing a webservice operation.
// It is the wire format data. It will be merged into another object where JSONDate values
// are converted to time.Time
type WebGridSearchRequestJSON struct {
	Cmd           string        `json:"cmd"`           // get, save, delete
	Limit         int           `json:"limit"`         // max number to return
	Offset        int           `json:"offset"`        // solution set offset
	Selected      []int         `json:"selected"`      // selected rows
	SearchLogic   string        `json:"searchLogic"`   // OR | AND
	Search        []GenSearch   `json:"search"`        // what fields and what values
	Sort          []ColSort     `json:"sort"`          // sort criteria
	SearchDtStart rlib.JSONDate `json:"searchDtStart"` // for time-sensitive searches
	SearchDtStop  rlib.JSONDate `json:"searchDtStop"`  // for time-sensitive searches
	Bool1         bool          `json:"Bool1"`         // a general purpose bool flag for postData from client
}

// WebGridSearchRequest is a struct suitable for describing a webservice operation.
type WebGridSearchRequest struct {
	Cmd           string      `json:"cmd"`           // get, save, delete
	Limit         int         `json:"limit"`         // max number to return
	Offset        int         `json:"offset"`        // solution set offset
	Selected      []int       `json:"selected"`      // selected rows
	SearchLogic   string      `json:"searchLogic"`   // OR | AND
	Search        []GenSearch `json:"search"`        // what fields and what values
	Sort          []ColSort   `json:"sort"`          // sort criteria
	SearchDtStart time.Time   `json:"searchDtStart"` // for time-sensitive searches
	SearchDtStop  time.Time   `json:"searchDtStop"`  // for time-sensitive searches
	Bool1         bool        `json:"Bool1"`         // a general purpose bool flag for postData from client
}

// WebFormRequest is a struct suitable for describing a webservice operation.
type WebFormRequest struct {
	Cmd      string      `json:"cmd"`    // get, save, delete
	Recid    int         `json:"recid"`  // max number to return
	FormName string      `json:"name"`   // solution set offset
	Record   interface{} `json:"record"` // selected rows
}

// WebTypeDownRequest is a search call made by a client while the user is
// typing in something to search for and the expecation is that the solution
// set will be sent back in realtime to aid the user.  Search is a string
// to search for -- it's what the user types in.  Max is the maximum number
// of matches to return.
type WebTypeDownRequest struct {
	Search string `json:"search"`
	Max    int    `json:"max"`
}

// WebGridDelete is a generic command structure returned when records are
// deleted from a grid. the Selected struct will contain the list of ids
// (recids which should map to the record type unique identifier) that are
// to be deleted.
type WebGridDelete struct {
	Cmd      string  `json:"cmd"`
	Selected []int64 `json:"selected"`
	Limit    int     `json:"limit"`
	Offset   int     `json:"offset"`
}

// ServiceData is the generalized data gatherer for svcHandler. It allows all
// the common data to be centrally parsed and passed to a handler, which may
// need to parse further to get its unique data.  It includes fields for
// common data elements in web svc requests
type ServiceData struct {
	Service       string               // the service requested (position 1)
	BID           int64                // which business (position 2)
	ID            int64                // the numeric id parsed from position 3
	DetVal        string               // value of 3rd path element if present (it is not always a number)
	UID           int64                // user id of requester
	TCID          int64                // TCID if supplied
	RAID          int64                // RAID if supplied
	RID           int64                // RAID if supplied
	RCPTID        int64                // RCPTID if supplied
	ASMID         int64                // ASMID if supplied
	ARID          int64                // ARID if supplied
	pathElements  []string             // the parts of the uri
	Dt            time.Time            // for cmds that need a single date
	D1            time.Time            // start of date range
	D2            time.Time            // end of date range
	wsSearchReq   WebGridSearchRequest // what did the search requester ask for
	wsTypeDownReq WebTypeDownRequest   // fast for typedown
	data          string               // the raw unparsed data
	sess          *rlib.Session        // the caller's session
	QueryParams   map[string][]string  // parameters when HTTP GET is used
	Files         map[string][]*multipart.FileHeader
	MFValues      map[string][]string
}

// Svcs is the table of all service handlers
var Svcs = []ServiceHandler{
	{"exportaccounts", SvcExportGLAccounts, true, true},
	{"importaccounts", SvcImportGLAccounts, true, true},
	{"account", SvcFormHandlerGLAccounts, true, true},
	{"accountlist", SvcAccountsList, true, true},
	{"accounts", SvcSearchHandlerGLAccounts, true, true},
	{"allocfunds", SvcSearchHandlerAllocFunds, true, true},
	{"ar", SvcFormHandlerAR, true, true},
	{"ars", SvcSearchHandlerARs, true, true},
	{"asm", SvcFormHandlerAssessment, true, true},
	{"asms", SvcSearchHandlerAssessments, true, true},
	{"authn", SvcAuthenticate, false, false},
	{"dep", SvcHandlerDepository, true, true},
	{"depmeth", SvcHandlerDepositMethod, true, true},
	{"deposit", SvcHandlerDeposit, true, true},
	{"depositlist", SvcHandlerDepositList, true, true},
	{"discon", SvcDisableConsole, false, true},
	{"encon", SvcEnableConsole, false, true},
	{"expense", SvcHandlerExpense, false, true},
	{"ledgers", getLedgerGrid, true, true},
	{"logoff", SvcLogoff, false, false},
	{"parentaccounts", SvcParentAccountsList, true, true},
	{"payorfund", SvcHandlerTotalUnallocFund, true, true},
	{"payorstmt", SvcPayorStmtDispatch, true, true},
	{"payorstmtinfo", SvcGetPayorStmInfo, true, true},
	{"person", SvcFormHandlerXPerson, true, true},
	{"ping", SvcHandlerPing, false, false},
	{"pmts", SvcHandlerPaymentType, true, true},
	{"postaccounts", SvcPostAccountsList, true, true},
	{"rapayor", SvcRAPayor, true, true},
	{"rapets", SvcRAPets, true, true},
	{"rar", SvcRARentables, true, true},
	{"receipt", SvcFormHandlerReceipt, true, true},
	{"receipts", SvcSearchHandlerReceipts, true, true},
	{"rentable", SvcFormHandlerRentable, true, true},
	{"rentables", SvcSearchHandlerRentables, true, true},
	{"rentablestd", SvcRentableTypeDown, true, true},
	{"rentalagr", SvcFormHandlerRentalAgreement, true, true},
	{"rentalagrs", SvcSearchHandlerRentalAgr, true, true},
	{"rentalagrtd", SvcRentalAgreementTypeDown, true, true},
	{"resetpw", SvcResetPW, false, false},
	{"rr", SvcRR, true, true},
	{"rt", SvcHandlerRentableType, true, true},
	{"rmr", SvcHandlerRentableMarketRates, true, true},
	{"rtlist", SvcRentableTypesTD, true, true},
	{"ruser", SvcRUser, true, true},
	{"stmt", SvcStatement, true, true},
	{"stmtdetail", SvcStatementDetail, true, true},
	{"stmtinfo", SvcGetStatementInfo, true, true},
	{"transactants", SvcSearchHandlerTransactants, true, true},
	{"transactantstd", SvcTransactantTypeDown, true, true},
	{"tws", SvcTWS, true, true},
	{"uilists", SvcUILists, false, false},
	{"uival", SvcUIVal, false, false},
	{"unpaidasms", SvcHandlerGetUnpaidAsms, true, true},
	{"userprofile", SvcUserProfile, false, true},
	{"version", SvcHandlerVersion, false, false},
}

// SvcCtx contains information global to the Svc handlers
var SvcCtx struct {
	NoAuth bool
}

// SvcInit initializes the service subsystem
func SvcInit(noauth bool) {
	SvcCtx.NoAuth = noauth
	rlib.RRdb.NoAuth = noauth // TODO(sudip): needs to be changed to some internal app struct
}

// V1ServiceHandler is the main dispatch point for WEB SERVICE requests
//
// The expected input is of the form:
//		request=%7B%22cmd%22%3A%22get%22%2C%22selected%22%3A%5B%5D%2C%22limit%22%3A100%2C%22offset%22%3A0%7D
// This is exactly what the w2ui grid sends as a request.
//
// Decoded, this message looks something like this:
//		request={"cmd":"get","selected":[],"limit":100,"offset":0}
//
// The leading "request=" is optional. This routine parses the basic
// information, then contacts an appropriate handler for more detailed
// processing.  It will set the Cmd member variable.
//
// W2UI sometimes sends requests that look like this: request=%7B%22search%22%3A%22s%22%2C%22max%22%3A250%7D
// using HTTP GET (rather than its more typical POST).  The command decodes to
// this: request={"search":"s","max":250}
//
//-----------------------------------------------------------------------------
func V1ServiceHandler(w http.ResponseWriter, r *http.Request) {
	funcname := "V1ServiceHandler"
	svcDebugTxn(funcname, r)
	var err error
	var d ServiceData

	d.ID = -1  // indicates it has not been set
	d.BID = -1 // indicates it has not been set

	if !SvcCtx.NoAuth {
		d.sess, err = rlib.GetSession(w, r)
		if err != nil {
			SvcErrorReturn(w, err, funcname)
			return
		}
		if d.sess != nil {
			d.sess.Refresh(w, r) // they actively tried to use the session, extend timeout
		}

		// get session in the request context
		ctx := rlib.SetSessionContextKey(r.Context(), d.sess)
		r = r.WithContext(ctx)
	}

	//-----------------------------------------------------------------------
	// pathElements:  0   1            2     3
	//               /v1/{subservice}/{BUI}/{ID} into an array of strings
	// BID is common to nearly all commands
	//-----------------------------------------------------------------------
	ss := strings.Split(r.RequestURI[1:], "?") // it could be GET command
	d.pathElements = strings.Split(ss[0], "/")
	d.Service = d.pathElements[1]
	if d.Service != "uilists" && len(d.pathElements) >= 3 {
		d.BID, err = getBIDfromBUI(d.pathElements[2])
		if err != nil {
			e := fmt.Errorf("Could not determine business from %s", d.pathElements[2])
			SvcErrorReturn(w, e, funcname)
			return
		}
		if d.BID < 0 {
			e := fmt.Errorf("Invalid business id: %s", d.pathElements[2])
			SvcErrorReturn(w, e, funcname)
			return
		}
	}
	if len(d.pathElements) >= 4 {
		d.DetVal = d.pathElements[3]
		d.ID, err = rlib.IntFromString(d.DetVal, "bad request integer value") // assume it's a BID
		if err != nil {
			d.ID = 0
		}
	}

	svcDebugURL(r, &d)
	showRequestHeaders(r)

	switch r.Method {
	case "POST":
		if nil != getPOSTdata(w, r, &d) {
			return
		}
	case "GET":
		if nil != getGETdata(w, r, &d) {
			return
		}
	}

	showWebRequest(&d)

	//-----------------------------------------------------------------------
	//  Now call the appropriate handler to do the rest
	//-----------------------------------------------------------------------
	found := false
	for i := 0; i < len(Svcs); i++ {
		if Svcs[i].Cmd == d.Service {
			if Svcs[i].NeedBiz && d.BID == 0 {
				var sbid = "<missing>"
				if len(d.pathElements) > 3 {
					sbid = d.pathElements[3]
				}
				e := fmt.Errorf("Could not identify business: %s", sbid)
				rlib.Console("***ERROR IN URL***  %s\n", e.Error())
				SvcErrorReturn(w, err, funcname)
				return
			}
			if !SvcCtx.NoAuth && Svcs[i].NeedSession && d.sess == nil || (d.sess != nil && d.sess.UID == 0) {
				e := fmt.Errorf("session required, please log in")
				rlib.Console("*** ERROR ***  command %s requires a session. SvcCtx.NoAuth = %t\n", Svcs[i].Cmd, SvcCtx.NoAuth)
				SvcErrorReturn(w, e, funcname)
				return
			}
			Svcs[i].Handler(w, r, &d)
			found = true
			break
		}
	}
	if !found {
		rlib.Console("**** YIPES! **** %s - Handler not found\n", r.RequestURI)
		e := fmt.Errorf("Service not recognized: %s", d.Service)
		rlib.Console("***ERROR IN URL***  %s", e.Error())
		SvcErrorReturn(w, e, funcname)
		return
	}
	svcDebugTxnEnd()
}

// SvcHandlerPing is the most basic test that you can run against the server
// see if it is alive and taking requests. It will return its version number.
func SvcHandlerPing(w http.ResponseWriter, r *http.Request, d *ServiceData) {
	fmt.Fprintf(w, "Accord Rentroll - Version %s\n", GetVersionNo())
}

// SvcHandlerVersion returns the server version number
//  @Title Verrsion
//  @URL /v1/version
//  @Method  POST or GET
//  @Synopsis Get the current server version
//  @Description Returns the server build number appended to the major/minor
//  @Description version number.
//  @Input
//  @Response version number
// wsdoc }
func SvcHandlerVersion(w http.ResponseWriter, r *http.Request, d *ServiceData) {
	fmt.Fprintf(w, "%s", GetVersionNo())
}

// SvcTWS returns a grid representation of the TWS table
// wsdoc {
//  @Title Timed Work Schedule
//  @URL /v1/tws
//  @Method  POST
//  @Synopsis Get the contents of the TWS table
//  @Description The TWS table shows all timed work currently scheduled
//  @Input WebGridSearchRequest
//  @Response tws.GridTable
// wsdoc }
func SvcTWS(w http.ResponseWriter, r *http.Request, d *ServiceData) {
	funcname := "SvcTWS"
	g, err := tws.WSGridData(d.wsSearchReq.Limit, d.wsSearchReq.Offset)
	if err != nil {
		SvcErrorReturn(w, err, funcname)
		return
	}
	SvcWriteResponse(&g, w)
}

func getBIDfromBUI(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return int64(0), nil
	}
	d, err := rlib.IntFromString(s, "bad request integer value") // assume it's a BID
	if err != nil {
		var ok bool // OK, let's see if it's a BUD
		err = nil   // clear the slate
		d, ok = rlib.RRdb.BUDlist[s]
		if !ok {
			d = 0
			err = fmt.Errorf("Could not find Business for %q", s)
		}
	}
	return d, err
}

// SvcErrorReturn formats an error return to the grid widget and sends it
func SvcErrorReturn(w http.ResponseWriter, err error, funcname string) {
	// rlib.Console("<Function>: %s | <Error>: %s\n", funcname, err.Error())
	rlib.Console("%s: %s\n", funcname, err.Error())
	var e SvcStatus
	e.Status = "error"
	e.Message = fmt.Sprintf("Error: %s\n", err.Error())
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(e)
	SvcWrite(w, b)
}

// SvcErrListReturn formats an error return to the grid widget and sends it
func SvcErrListReturn(w http.ResponseWriter, errlist []bizlogic.BizError, funcname string) {
	err := bizlogic.BizErrorListToError(errlist)
	SvcErrorReturn(w, err, funcname)
}

// SvcGetInt64 tries to read an int64 value from the supplied string.
// If it fails for any reason, it sends writes an error message back
// to the caller and returns the error.  Otherwise, it returns an
// int64 and returns nil
func SvcGetInt64(s, errmsg string, w http.ResponseWriter) (int64, error) {
	i, err := rlib.IntFromString(s, "not an integer number")
	if err != nil {
		err = fmt.Errorf("%s: %s", errmsg, err.Error())
		SvcErrorReturn(w, err, "SvcGetInt64")
		return i, err
	}
	return i, nil
}

// SvcExtractIDFromURI extracts an int64 id value from position pos of the supplied uri.
// The URI is of the form returned by http.Request.RequestURI .  In particular:
//
//	pos:     0    1      2  3
//  uri:    /v1/rentable/34/421
//
// So, in the example uri above, a call where pos = 3 would return int64(421). errmsg
// is a string that will be used in the error message if the requested position had an
// error during conversion to int64. So in the example above, pos 3 is the RID, so
// errmsg would probably be set to "RID"
func SvcExtractIDFromURI(uri, errmsg string, pos int, w http.ResponseWriter) (int64, error) {
	var ID = int64(0)
	var err error
	var funcname = "SvcExtractIDFromURI"

	sa := strings.Split(uri[1:], "/")
	// rlib.Console("uri parts:  %v\n", sa)
	if len(sa) < pos+1 {
		err = fmt.Errorf("Expecting at least %d elements in URI: %s, but found only %d", pos+1, uri, len(sa))
		// rlib.Console("err = %s\n", err)
		SvcErrorReturn(w, err, funcname)
		return ID, err
	}
	// rlib.Console("sa[pos] = %s\n", sa[pos])
	ID, err = SvcGetInt64(sa[pos], errmsg, w)
	return ID, err
}

func getPOSTdata(w http.ResponseWriter, r *http.Request, d *ServiceData) error {
	funcname := "getPOSTdata"
	var err error

	const _1MB = (1 << 20) * 1024

	// if content type is form data then
	ct := r.Header.Get("Content-Type")
	ct, _, err = mime.ParseMediaType(ct)
	if err != nil {
		e := fmt.Errorf("%s: Error while parsing content type: %s", funcname, err.Error())
		SvcErrorReturn(w, e, funcname)
		return e
	}
	if ct == "multipart/form-data" {
		// parse multipart form first
		err = r.ParseMultipartForm(_1MB)
		if err != nil {
			e := fmt.Errorf("%s: Error while parsing multipart form: %s", funcname, err.Error())
			SvcErrorReturn(w, e, funcname)
			return e
		}

		// check for headers
		for _, fheaders := range r.MultipartForm.File {
			for _, fh := range fheaders {
				cd := "Content-Disposition"
				if _, ok := fh.Header["Content-Disposition"]; !ok {
					e := fmt.Errorf("%s: Header missing (%s)", funcname, cd)
					SvcErrorReturn(w, e, funcname)
					return e
				}
				ct := "Content-Type"
				if _, ok := fh.Header["Content-Type"]; !ok {
					e := fmt.Errorf("%s: Header missing (%s)", funcname, ct)
					SvcErrorReturn(w, e, funcname)
					return e
				}
			}
		}

		d.Files = r.MultipartForm.File
		d.MFValues = r.MultipartForm.Value
	}

	htmlData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		e := fmt.Errorf("%s: Error reading message Body: %s", funcname, err.Error())
		SvcErrorReturn(w, e, funcname)
		return e
	}
	rlib.Console("\t- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -\n")
	rlib.Console("\thtmlData = %s\n", htmlData)
	if len(htmlData) == 0 {
		d.wsSearchReq.Cmd = "?"
		return nil
	}
	u, err := url.QueryUnescape(string(htmlData))
	if err != nil {
		e := fmt.Errorf("%s: Error with QueryUnescape: %s", funcname, err.Error())
		SvcErrorReturn(w, e, funcname)
		return e
	}
	rlib.Console("\tUnescaped htmlData = %s\n", u)

	u = strings.TrimPrefix(u, "request=") // strip off "request=" if it is present
	d.data = u
	var wjs WebGridSearchRequestJSON
	err = json.Unmarshal([]byte(u), &wjs)
	if err != nil {
		e := fmt.Errorf("%s: Error with json.Unmarshal:  %s", funcname, err.Error())
		SvcErrorReturn(w, e, funcname)
		return e
	}
	rlib.MigrateStructVals(&wjs, &d.wsSearchReq)
	return err
}

func getGETdata(w http.ResponseWriter, r *http.Request, d *ServiceData) error {
	funcname := "getGETdata"
	s, err := url.QueryUnescape(strings.TrimSpace(r.URL.String()))
	if err != nil {
		e := fmt.Errorf("%s: Error with url.QueryUnescape:  %s", funcname, err.Error())
		SvcErrorReturn(w, e, funcname)
		return e
	}
	rlib.Console("Unescaped query = %s\n", s)
	d.QueryParams = r.URL.Query()
	rlib.Console("Query Parameters: %v\n", d.QueryParams)
	w2uiPrefix := "request="
	n := strings.Index(s, w2uiPrefix)
	rlib.Console("n = %d\n", n)
	if n > 0 {
		rlib.Console("Will process as Typedown\n")
		d.data = s[n+len(w2uiPrefix):]
		rlib.Console("%s: will unmarshal: %s\n", funcname, d.data)
		if err = json.Unmarshal([]byte(d.data), &d.wsTypeDownReq); err != nil {
			e := fmt.Errorf("%s: Error with json.Unmarshal:  %s", funcname, err.Error())
			SvcErrorReturn(w, e, funcname)
			return e
		}
		d.wsSearchReq.Cmd = "typedown"
	} else {
		rlib.Console("Will process as web search command\n")
		d.wsSearchReq.Cmd = r.URL.Query().Get("cmd")
	}
	return nil
}

func showRequestHeaders(r *http.Request) {
	rlib.Console("Headers:\n")
	for k, v := range r.Header {
		rlib.Console("\t%s: ", k)
		for i := 0; i < len(v); i++ {
			rlib.Console("%q  ", v[i])
		}
		rlib.Console("\n")
	}
}

func showWebRequest(d *ServiceData) {
	if d.wsSearchReq.Cmd == "typedown" {
		rlib.Console("Typedown:\n")
		rlib.Console("\tSearch  = %q\n", d.wsTypeDownReq.Search)
		rlib.Console("\tMax     = %d\n", d.wsTypeDownReq.Max)
	} else {
		rlib.Console("\tSearchReq:\n")
		rlib.Console("\t\tCmd           = %s\n", d.wsSearchReq.Cmd)
		rlib.Console("\t\tLimit         = %d\n", d.wsSearchReq.Limit)
		rlib.Console("\t\tOffset        = %d\n", d.wsSearchReq.Offset)
		rlib.Console("\t\tsearchLogic   = %s\n", d.wsSearchReq.SearchLogic)
		rlib.Console("\t\tsearchDtStart = %s\n", time.Time(d.wsSearchReq.SearchDtStart).Format(rlib.RRDATEFMT4))
		rlib.Console("\t\tsearchDtStop  = %s\n", time.Time(d.wsSearchReq.SearchDtStop).Format(rlib.RRDATEFMT4))
		for i := 0; i < len(d.wsSearchReq.Search); i++ {
			rlib.Console("\t\tsearch[%d] - Field = %s,  Type = %s,  Value = %s,  Operator = %s\n", i, d.wsSearchReq.Search[i].Field, d.wsSearchReq.Search[i].Type, d.wsSearchReq.Search[i].Value, d.wsSearchReq.Search[i].Operator)
		}
		for i := 0; i < len(d.wsSearchReq.Sort); i++ {
			rlib.Console("\t\tsort[%d] - Field = %s,  Direction = %s\n", i, d.wsSearchReq.Sort[i].Field, d.wsSearchReq.Sort[i].Direction)
		}
	}
}

func svcDebugTxn(funcname string, r *http.Request) {
	rlib.Console("\n%s\n", rlib.Mkstr(80, '-'))
	rlib.Console("URL:      %s\n", r.URL.String())
	rlib.Console("METHOD:   %s\n", r.Method)
	rlib.Console("Handler:  %s\n", funcname)
}

func svcDebugURL(r *http.Request, d *ServiceData) {
	//-----------------------------------------------------------------------
	// pathElements: 0         1     2
	// Break up {subservice}/{BUI}/{ID} into an array of strings
	// BID is common to nearly all commands
	//-----------------------------------------------------------------------
	//ss := strings.Split(r.RequestURI[1:], "?") // it could be GET command
	//pathElements := strings.Split(ss[0], "/")
	rlib.Console("\t%s\n", r.URL.String()) // print before we strip it off
	for i := 0; i < len(d.pathElements); i++ {
		rlib.Console("\t\t%d. %s\n", i, d.pathElements[i])
	}
	rlib.Console("BUSINESS: %d\n", d.BID)
	rlib.Console("ID:       %d\n", d.ID)
}

func svcDebugTxnEnd() {
	rlib.Console("END\n")
}

// SvcWriteResponse finishes the transaction with the W2UI client
func SvcWriteResponse(g interface{}, w http.ResponseWriter) {
	b, err := json.Marshal(g)
	if err != nil {
		e := fmt.Errorf("Error marshaling json data: %s", err.Error())
		rlib.Ulog("SvcWriteResponse: %s\n", err.Error())
		SvcErrorReturn(w, e, "SvcWriteResponse")
		return
	}
	SvcWrite(w, b)
}

// SvcWrite is a general write routine for service calls... it is a bottleneck
// where we can place debug statements as needed.
func SvcWrite(w http.ResponseWriter, b []byte) {
	charsToPrint := 500
	format := fmt.Sprintf("First %d chars of response: %%-%d.%ds\n", charsToPrint, charsToPrint, charsToPrint)
	// rlib.Console("Format string = %q\n", format)
	rlib.Console(format, string(b))
	// rlib.Console("\nResponse Data:  %s\n\n", string(b))
	w.Write(b)
}

// SvcWriteSuccessResponse is used to complete a successful write operation on w2ui form save requests.
func SvcWriteSuccessResponse(w http.ResponseWriter) {
	var g = SvcStatusResponse{Status: "success"}
	w.Header().Set("Content-Type", "application/json")
	SvcWriteResponse(&g, w)
}

// SvcWriteSuccessResponseWithID is used to complete a successful write operation on w2ui form save requests.
func SvcWriteSuccessResponseWithID(w http.ResponseWriter, id int64) {
	var g = SvcStatusResponse{Status: "success", Recid: id}
	w.Header().Set("Content-Type", "application/json")
	SvcWriteResponse(&g, w)
}
