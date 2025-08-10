// Package main (sender.go) :
// These methods are for sending GAS scripts to Google Drive.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ggsrun/utl"

	"github.com/urfave/cli"
)

// handleGasError checks for and formats a detailed error from the GAS execution result.
func handleGasError(result interface{}) (string, bool) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", false
	}

	var gasErr GasError
	if err := json.Unmarshal(resultBytes, &gasErr); err == nil && gasErr.GasError.Message != "" {
		formattedError := fmt.Sprintf(
			"---\nError Type: %s\nMessage: %s\nStack Trace:\n%s\n---",
			gasErr.GasError.Name,
			gasErr.GasError.Message,
			gasErr.GasError.Stack,
		)
		return formattedError, true
	}
	return "", false
}

// Exe1Function : 
func (e *ExecutionContainer) exe1Function(c *cli.Context) error {
	if len(c.String("scriptfile")) > 0 || c.Bool("backup") {
		if err := e.projectBackup(c); err != nil {
			return err
		}
		if err := e.projectUpdateIni(utl.ConvGasToPut(c)); err != nil {
			return err
		}
		return e.projectUpdate2()
	}
	return nil
}

// Exe2Function : 
func (e *ExecutionContainer) exe2Function(c *cli.Context) error {
	if c.Bool("foldertree") {
		btof := "function main(){return new ggsrun(null, null, null).foldertree()}"
		e.executionAPIwithServer(utl.ConvStringToRun(c, btof))
		return e.esenderForExe2(c)
	}
	if c.Bool("convert") && len(c.String("value")) > 0 {
		btof := "function main(e){return new ggsrun(e, null, null).nodocsdownloader()}"
		e.executionAPIwithServer(utl.ConvStringToRun(c, btof))
		if err := e.esenderForExe2(c); err != nil {
			return err
		}
		return e.byteSliceConverter()
	} else if c.Bool("convert") && len(c.String("value")) == 0 {
		return fmt.Errorf("no File ID. Please set it using '-v [ File ID ]'")
	}
	if len(c.String("stringscript")) > 0 {
		e.executionAPIwithServer(utl.ConvStringToRun(c, c.String("stringscript")))
		return e.esenderForExe2(c)
	}
	e.executionAPIwithServer(utl.ConvGasToRun(c))
	return e.esenderForExe2(c)
}

// doExecutionAPIwithoutServer prepares parameters for an 'exe1' command execution.
// It sets the default function name if not provided and enables development mode.
func doExecutionAPIwithoutServer(param *Param, msg []string) []string {
	if len(param.Function) == 0 {
		param.Function = deffuncwithout
		msg = append(msg, fmt.Sprintf("Executed default function '%s()'.", deffuncwithout))
	}
	param.DevMode = true
	return msg
}

// executionAPIwithServer : 
func (e *ExecutionContainer) executionAPIwithServer(sendscript string) error {
	if len(sendscript) == 0 {
		return fmt.Errorf("no script. Please set GAS script using '-s'")
	}
	if len(e.Param.Function) == 0 {
		e.Param.Function = deffuncserv
	}
	scr := &Com{
		Com:     sendscript,
		Exefunc: e.Param.Function,
		Log:     e.InitVal.log,
	}
	scri, _ := json.Marshal(scr)
	e.Param.Parameters = []string{string(scri)}
	e.Param.DevMode = true
	return nil
}

// doExecutionError checks for and handles errors from the Apps Script Execution API.
func doExecutionError(body []byte, err error, accessToken string) error {
	if err == nil {
		return nil
	}
	feedBackData := &FeedBackData{}
	json.Unmarshal(body, feedBackData)
	if feedBackData.Error.Status == "UNAUTHENTICATED" {
		chk, _ := doChkAtokenForExecution(accessToken)
		if chk != nil && len(chk.Error) > 0 {
			return fmt.Errorf("invalid Access token. Please retrieve it again using command '%s auth'.\nCurrent access token is '%s'", appname, accessToken)
		}
		return fmt.Errorf("authorization Error: Please check SCOPEs of your GAS script and server using GAS Script Editor.\nIf the SCOPEs have changed, modify them in '%s' and delete a line of 'refresh_token', then, execute '%s' again. You can retrieve new access token with modified SCOPEs", cfgFile, appname)
	}
	if feedBackData.Error.Message == "PERMISSION_DENIED" &&
		feedBackData.Error.Code == 403 {
		return fmt.Errorf("please check Execution API at Developer console.\nIf Execution API is unable, please enable it. Or please check 'client_secret.json'. It might be that that is not for the project with Execution API")
	}
	if feedBackData.Error.Message == "Requested entity was not found." &&
		feedBackData.Error.Code == 404 {
		return fmt.Errorf("please check the deployment of API executable and/or the ggsrun server.\n - If you use command 'e1', please deploy API executable again. If you use command 'e2', please check both again.\n - After deployed API executable, please save each scripts on the project again. This is very important point!\n - When you use the server as library, please confirm server.\n - Also you can use 'Logger.log(ggsrunif.Beacon())' at Google Apps Script Editor to confirm server condition.\n - Also, please check the script ID")
	}
	if len(feedBackData.Error.Detailes) > 0 && feedBackData.Error.Detailes[0].ErrorMessage == "The script completed but the returned value is not a supported return type." &&
		feedBackData.Error.Code == 500 {
		return fmt.Errorf(feedBackData.Error.Detailes[0].ErrorMessage)
	}
	return fmt.Errorf("API Error: %w, Body: %s", err, body)
}



// MarshalJSON : For exe1
func (e *e1para) MarshalJSON() ([]byte, error) {
	var outd string
	if len(e.Parameters) > 0 {
		if regexp.MustCompile(`^[+-]?[0-9]*[.]?[0-9]+$`).Match([]byte(e.Parameters[0].(string))) ||
			regexp.MustCompile(`^[[]]$`).Match([]byte(e.Parameters[0].(string))) ||
			regexp.MustCompile("^{|} ").Match([]byte(e.Parameters[0].(string))) {
			outd = fmt.Sprintf("{\"devMode\":%t, \"parameters\":%v, \"function\":%q}", e.DevMode, e.Parameters, e.Function)
		} else if regexp.MustCompile("([a-zA-Z]|[0-9].*[a-zA-Z]|[a-zA-Z].*[0-9])").Match([]byte(e.Parameters[0].(string))) {
			outd = fmt.Sprintf("{\"devMode\":%t, \"parameters\":%q, \"function\":%q}", e.DevMode, e.Parameters, e.Function)
		}
	} else {
		outd = fmt.Sprintf("{\"devMode\":%t, \"function\":%q}", e.DevMode, e.Function)
	}
	return []byte(outd), nil
}

// doEsenderForExe1 sends the request to the Apps Script Execution API and processes the response.
func doEsenderForExe1(c *cli.Context, param *Param, ggsrunCfg *GgsrunCfg, pstart time.Time, msg []string) (*FeedBackData, []string, error) {
	var paraint []interface{}
	if len(c.String("value")) > 0 {
		paraint = []interface{}{c.String("value")}
	}
	epara := &e1para{
		Function:   param.Function,
		Parameters: paraint,
		DevMode:    param.DevMode,
	}
	re, _ := json.Marshal(epara)
	if len(re) == 0 {
		return nil, msg, fmt.Errorf("format of values is wrong. Double and single quotates have to be escaped.\n - Inputted value was  %s", c.String("value"))
	}
	r := &utl.RequestParams{
		Method:      "POST",
		APIURL:      executionurl + ggsrunCfg.Scriptid + ":run",
		Data:        bytes.NewBuffer(re),
		Contenttype: "application/json;charset=UTF-8",
		Accesstoken: ggsrunCfg.Accesstoken,
		Dtime:       370,
	}
	body, err := r.FetchAPI()
	if err := doExecutionError(body, err, ggsrunCfg.Accesstoken); err != nil {
		return nil, msg, err
	}
	feedBackData := &FeedBackData{}
	json.Unmarshal(body, feedBackData)
	var dat string
	if len(feedBackData.Error.Message) > 0 {
		if len(feedBackData.Error.Detailes[0].ScriptStackTraceElements) > 0 {
			dat = fmt.Sprintf("{code: %d, message: %s, function: %s, linenumber: %d}", feedBackData.Error.Code, feedBackData.Error.Message, feedBackData.Error.Detailes[0].ScriptStackTraceElements[0].Function, feedBackData.Error.Detailes[0].ScriptStackTraceElements[0].LineNumber)
		} else {
			dat = fmt.Sprintf("{code: %d, message: %s}", feedBackData.Error.Code, feedBackData.Error.Message)
		}
		msg = append(msg, dat)
	} else {
		var rs map[string]interface{}
		json.Unmarshal(body, &rs)
		result := rs["response"].(map[string]interface{})["result"]
		if formattedError, isGasError := handleGasError(result); isGasError {
			msg = append(msg, formattedError)
			feedBackData.Response.Result.Result = nil // Clear the result to avoid double printing
		} else {
			feedBackData.Response.Result.Result = result
		}
	}
	if len(feedBackData.Error.Detailes) > 0 {
		dat = fmt.Sprintf("{detailmessage: %s}", feedBackData.Error.Detailes[0].ErrorMessage)
		msg = append(msg, dat)
	}
	feedBackData.Response.Result.TotalEt = math.Trunc(time.Since(pstart).Seconds()*1000) / 1000
	feedBackData.Response.Result.Uapi = eapir1
	msg = append(msg, fmt.Sprintf("Function '%s()' was run.", param.Function))
	return feedBackData, msg, nil
}


// esenderForExe2 : Sends GAS to Google and retrieves results.
func (e *ExecutionContainer) esenderForExe2(c *cli.Context) error {
	re, _ := json.Marshal(e.Param)
	r := &utl.RequestParams{
		Method:      "POST",
		APIURL:      executionurl + e.GgsrunCfg.Scriptid + ":run",
		Data:        bytes.NewBuffer(re),
		Contenttype: "application/json;charset=UTF-8",
		Accesstoken: e.GgsrunCfg.Accesstoken,
		Dtime:       370,
	}
	body, err := r.FetchAPI()
	if err := e.executionError(body, err); err != nil {
		return err
	}
	json.Unmarshal(body, &e.FeedBackData)
	var dat string
	if len(e.FeedBackData.Error.Message) > 0 {
		dat = fmt.Sprintf("{code: %d, message: %s}", e.FeedBackData.Error.Code, e.FeedBackData.Error.Message)
		e.Msg = append(e.Msg, dat)
	}
	if len(e.FeedBackData.Error.Detailes) > 0 {
		if strings.Contains(e.FeedBackData.Error.Detailes[0].ErrorMessage, deffuncserv) {
			dat = fmt.Sprintf("{server_error: Server for ggsrun is NOT found. Please deploy the server which is a library for GAS as 'ggsrunif'. Sctipt ID of the library is '%s'.}", serverid)
		} else {
			dat = fmt.Sprintf("{detailmessage: %s}", e.FeedBackData.Error.Detailes[0].ErrorMessage)
		}
		e.Msg = append(e.Msg, dat)
		return nil
	}

	if formattedError, isGasError := handleGasError(e.FeedBackData.Response.Result.Result); isGasError {
		e.Msg = append(e.Msg, formattedError)
		e.FeedBackData.Response.Result.Result = nil // Clear the result to avoid double printing
	} else {
			dlfileinf, _ := json.Marshal(e.FeedBackData.Response.Result.Result)
			var rs map[string]interface{}
			if err := json.Unmarshal(dlfileinf, &rs); err == nil {
				fid, ok := rs["fileid"].(string)
				if ok {
					e.DlFileByScript.Fileid = fid
				}
				exn, ok := rs["extension"].(string)
				if ok {
					e.DlFileByScript.Extension = exn
				}
				if len(fid) > 0 && len(exn) > 0 {
					delete(rs, "fileid")
					delete(rs, "extension")
					e.FeedBackData.Response.Result.Result = rs
					res := e.defDownloadByScriptContainer().
						GetFileinf().
						Downloader(c)
					e.Msg = append(e.Msg, res.Msgar...)
				}
			}
	}

	e.FeedBackData.Response.Result.TotalEt = math.Trunc(time.Since(e.InitVal.pstart).Seconds()*1000) / 1000
	e.FeedBackData.Response.Result.Uapi = eapir2
	e.Msg = append(e.Msg, fmt.Sprintf("'%s()' in the script was run using ggsrun server. Server function is '%s()'.", deffuncwith, e.Param.Function))
	return nil
}

// projectUpdateIni : Initialize for updating project
func (e *ExecutionContainer) projectUpdateIni(sendscript string) error {
	var overwrite bool
	for i := range e.Project.Files {
		if e.Project.Files[i].Name == defprojectname {
			e.Project.Files[i].Source = sendscript
			overwrite = true
		}
	}
	if !overwrite {
		filedata := &File{
			Name:   defprojectname,
			Type:   "SERVER_JS",
			Source: sendscript,
		}
		e.Project.Files = append(e.Project.Files, *filedata)
	}
	return nil
}

// projectUpdate2 : In this method, the project is updated using Apps Script API.
func (e *ExecutionContainer) projectUpdate2() error {
	script, _ := json.Marshal(e.Project)
	tokenparams := url.Values{}
	tokenparams.Set("fields", "files,scriptId")
	u, _ := url.Parse(appsscriptapi)
	u.Path = path.Join(u.Path, e.GgsrunCfg.Scriptid+"/content")
	r := &utl.RequestParams{
		Method:      "PUT",
		APIURL:      u.String() + "?" + tokenparams.Encode(),
		Data:        bytes.NewBuffer(script),
		Accesstoken: e.GgsrunCfg.Accesstoken,
		Dtime:       30,
	}
	res, err := r.FetchAPI()
	if err != nil {
		utl.DispScopeError2(res)
		return fmt.Errorf("project update failed: %w", err)
	}
	e.Msg = append(e.Msg, "Project was updated.")
	_ = res // Now, no results are returned.
	return nil
}

// ProjectBackup : Download and backup project (Apps Script API v1)
func (e *ExecutionContainer) projectBackup(c *cli.Context) error {
	tokenparams := url.Values{}
	tokenparams.Set("fields", "files,scriptId")
	u, _ := url.Parse(appsscriptapi)
	u.Path = path.Join(u.Path, e.GgsrunCfg.Scriptid+"/content")
	r := &utl.RequestParams{
		Method:      "GET",
		APIURL:      u.String() + "?" + tokenparams.Encode(),
		Data:        nil,
		Contenttype: "application/x-www-form-urlencoded",
		Accesstoken: e.GgsrunCfg.Accesstoken,
		Dtime:       30,
	}
	res, err := r.FetchAPI()
	if err != nil {
		utl.DispScopeError2(res)
		return fmt.Errorf("project backup failed. Was the inputted project ID correct? Error: %w", err)
	}
	json.Unmarshal(res, &e.Project)
	if c.Bool("backup") {
		btok, _ := json.MarshalIndent(e.Project, "", "\t")
		filename := e.InitVal.pstart.Format("20060102_150405") + ".gs"
		if err := ioutil.WriteFile(filepath.Join(e.InitVal.workdir, filename), btok, 0777); err != nil {
			return fmt.Errorf("failed to write backup file: %w", err)
		}
		dat := fmt.Sprintf("Project was saved as '%s'.", filename)
		e.Msg = append(e.Msg, dat)
	}
	return nil
}

// WebAppswithServerForExe3 : Sends GAS to Google and retrieves results.
func (e *ExecutionContainer) webAppswithServerForExe3(script string, c *cli.Context) error {
	if len(c.String("url")) == 0 {
		return fmt.Errorf("no URL for Web Apps")
	}
	tokenparams := url.Values{}
	tokenparams.Set("com", script)
	tokenparams.Set("pass", c.String("password"))
	tokenparams.Set("log", strconv.FormatBool(c.Bool("log")))
	r := &utl.RequestParams{
		Method:      "POST",
		APIURL:      c.String("url"),
		Data:        strings.NewReader(tokenparams.Encode()),
		Contenttype: "application/x-www-form-urlencoded",
		Accesstoken: "",
		Dtime:       370,
	}
	body, err := r.FetchAPI()
	if err != nil {
		return fmt.Errorf("please check Web Apps Service and/or URL of it. Web Apps Service might not be deployed: %w", err)
	}
	json.Unmarshal(body, &e.FeedBackData.Response.Result)
	e.FeedBackData.Response.Result.TotalEt = math.Trunc(time.Since(e.InitVal.pstart).Seconds()*1000) / 1000
	e.FeedBackData.Response.Result.Uapi = wapps
	dlfileinf, _ := json.Marshal(e.FeedBackData.Response.Result.Result)
	var rs map[string]interface{}
	if err := json.Unmarshal(dlfileinf, &rs); err == nil {
		e.DlFileByScript.Fileid, _ = rs["fileid"].(string)
		e.DlFileByScript.Extension, _ = rs["extension"].(string)
		if len(e.DlFileByScript.Fileid) > 0 && len(e.DlFileByScript.Extension) > 0 {
			delete(rs, "fileid")
			delete(rs, "extension")
			e.FeedBackData.Response.Result.Result = rs
			e.Msg = append(e.Msg, "This mode cannot download files. Because this mode is not authorization.")
		}
	}
	return nil
}

// ByteSliceConverter : 
func (e *ExecutionContainer) byteSliceConverter() error {
	if !strings.Contains(fmt.Sprintf("%s", e.FeedBackData.Response.Result.Result), "Error") {
		var f ByteSliceFile
		rr, _ := json.Marshal(e.FeedBackData.Response.Result.Result)
		json.Unmarshal(rr, &f)
		c := make([]uint8, len(f.FileData))
		for n := range f.FileData {
			c[n] = uint8(f.FileData[n])
		}
		if err := ioutil.WriteFile(f.Name, c, 0777); err != nil {
			return fmt.Errorf("failed to write byte slice file: %w", err)
		}
		e.FeedBackData.Response.Result.Result = "### Byte Slice of File ###"
		e.Msg = append(e.Msg, fmt.Sprintf("File was downloaded as '%s'. MimeType is '%s'.", f.Name, f.MimeType))
	} else {
		e.FeedBackData.Response.Result.Result = "Server isn't installed or Wrong File ID."
	}
	return nil
}