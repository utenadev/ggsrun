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

// doExe2Function handles the logic for the 'exe2' command.
func doExe2Function(c *cli.Context, param *Param, ggsrunCfg *GgsrunCfg, initVal *InitVal, msg []string, dlFileByScript *DlFileByScript) (*FeedBackData, []string, *DlFileByScript, error) {
	var err error
	if c.Bool("foldertree") {
		btof := "function main(){return new ggsrun(null, null, null).foldertree()}"
		if err = doExecutionAPIwithServer(utl.ConvStringToRun(c, btof), param, initVal.log); err != nil {
			return nil, msg, dlFileByScript, err
		}
	} else if c.Bool("convert") && len(c.String("value")) > 0 {
		btof := "function main(e){return new ggsrun(e, null, null).nodocsdownloader()}"
		if err = doExecutionAPIwithServer(utl.ConvStringToRun(c, btof), param, initVal.log); err != nil {
			return nil, msg, dlFileByScript, err
		}
	} else if c.Bool("convert") && len(c.String("value")) == 0 {
		return nil, msg, dlFileByScript, fmt.Errorf("no File ID. Please set it using '-v [ File ID ]'")
	} else if len(c.String("stringscript")) > 0 {
		if err = doExecutionAPIwithServer(utl.ConvStringToRun(c, c.String("stringscript")), param, initVal.log); err != nil {
			return nil, msg, dlFileByScript, err
		}
	} else {
		if err = doExecutionAPIwithServer(utl.ConvGasToRun(c), param, initVal.log); err != nil {
			return nil, msg, dlFileByScript, err
		}
	}

	feedBackData, msg, dlFileByScript, err := doEsenderForExe2(c, param, ggsrunCfg, initVal, msg, dlFileByScript)
	if err != nil {
		return feedBackData, msg, dlFileByScript, err
	}

	if c.Bool("convert") && len(c.String("value")) > 0 {
		feedBackData, msg, err = doByteSliceConverter(feedBackData, msg)
		if err != nil {
			return feedBackData, msg, dlFileByScript, err
		}
	}

	return feedBackData, msg, dlFileByScript, nil
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

// doExecutionAPIwithServer prepares parameters for an 'exe2' command execution.
func doExecutionAPIwithServer(sendscript string, param *Param, log bool) error {
	if len(sendscript) == 0 {
		return fmt.Errorf("no script. Please set GAS script using '-s'")
	}
	if len(param.Function) == 0 {
		param.Function = deffuncserv
	}
	scr := &Com{
		Com:     sendscript,
		Exefunc: param.Function,
		Log:     log,
	}
	scri, _ := json.Marshal(scr)
	param.Parameters = []string{string(scri)}
	param.DevMode = true
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


// doEsenderForExe2 sends the request to the Apps Script Execution API for exe2 commands.
func doEsenderForExe2(c *cli.Context, param *Param, ggsrunCfg *GgsrunCfg, initVal *InitVal, msg []string, dlFileByScript *DlFileByScript) (*FeedBackData, []string, *DlFileByScript, error) {
	re, _ := json.Marshal(param)
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
		return nil, msg, nil, err
	}
	feedBackData := &FeedBackData{}
	json.Unmarshal(body, feedBackData)
	var dat string
	if len(feedBackData.Error.Message) > 0 {
		dat = fmt.Sprintf("{code: %d, message: %s}", feedBackData.Error.Code, feedBackData.Error.Message)
		msg = append(msg, dat)
	}
	if len(feedBackData.Error.Detailes) > 0 {
		if strings.Contains(feedBackData.Error.Detailes[0].ErrorMessage, deffuncserv) {
			dat = fmt.Sprintf("{server_error: Server for ggsrun is NOT found. Please deploy the server which is a library for GAS as 'ggsrunif'. Sctipt ID of the library is '%s'.}", serverid)
		} else {
			dat = fmt.Sprintf("{detailmessage: %s}", feedBackData.Error.Detailes[0].ErrorMessage)
		}
		msg = append(msg, dat)
		return feedBackData, msg, nil, nil
	}

	if formattedError, isGasError := handleGasError(feedBackData.Response.Result.Result); isGasError {
		msg = append(msg, formattedError)
	} else {
		dlfileinf, _ := json.Marshal(feedBackData.Response.Result.Result)
		var rs map[string]interface{}
		if err := json.Unmarshal(dlfileinf, &rs); err == nil {
			fid, ok := rs["fileid"].(string)
			if ok {
				dlFileByScript.Fileid = fid
			}
			exn, ok := rs["extension"].(string)
			if ok {
				dlFileByScript.Extension = exn
			}
			if len(fid) > 0 && len(exn) > 0 {
				delete(rs, "fileid")
				delete(rs, "extension")
				feedBackData.Response.Result.Result = rs
				res := newDownloadByScriptContainer(
					msg,
					ggsrunCfg.Accesstoken,
					initVal.workdir,
					initVal.pstart,
					dlFileByScript,
				).
					GetFileinf().
					Downloader(c)
				msg = append(msg, res.Msgar...)
			}
		}
	}

	feedBackData.Response.Result.TotalEt = math.Trunc(time.Since(initVal.pstart).Seconds()*1000) / 1000
	feedBackData.Response.Result.Uapi = eapir2
	msg = append(msg, fmt.Sprintf("'%s()' in the script was run using ggsrun server. Server function is '%s()'.", deffuncwith, param.Function))
	return feedBackData, msg, dlFileByScript, nil
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

// doWebAppswithServerForExe3 sends a request to a Web App and processes the result.
func doWebAppswithServerForExe3(script string, c *cli.Context, pstart time.Time) (*FeedBackData, []string, *DlFileByScript, error) {
	if len(c.String("url")) == 0 {
		return nil, nil, nil, fmt.Errorf("no URL for Web Apps")
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
		return nil, nil, nil, fmt.Errorf("please check Web Apps Service and/or URL of it. Web Apps Service might not be deployed: %w", err)
	}
	feedBackData := &FeedBackData{}
	msg := []string{}
	dlFileByScript := &DlFileByScript{}
	json.Unmarshal(body, &feedBackData.Response.Result)
	feedBackData.Response.Result.TotalEt = math.Trunc(time.Since(pstart).Seconds()*1000) / 1000
	feedBackData.Response.Result.Uapi = wapps
	dlfileinf, _ := json.Marshal(feedBackData.Response.Result.Result)
	var rs map[string]interface{}
	if err := json.Unmarshal(dlfileinf, &rs); err == nil {
		dlFileByScript.Fileid, _ = rs["fileid"].(string)
		dlFileByScript.Extension, _ = rs["extension"].(string)
		if len(dlFileByScript.Fileid) > 0 && len(dlFileByScript.Extension) > 0 {
			delete(rs, "fileid")
			delete(rs, "extension")
			feedBackData.Response.Result.Result = rs
			msg = append(msg, "This mode cannot download files. Because this mode is not authorization.")
		}
	}
	return feedBackData, msg, dlFileByScript, nil
}

// doByteSliceConverter converts the byte slice data from the execution result into a file.
func doByteSliceConverter(feedBackData *FeedBackData, msg []string) (*FeedBackData, []string, error) {
	if !strings.Contains(fmt.Sprintf("%s", feedBackData.Response.Result.Result), "Error") {
		var f ByteSliceFile
		rr, _ := json.Marshal(feedBackData.Response.Result.Result)
		json.Unmarshal(rr, &f)
		c := make([]uint8, len(f.FileData))
		for n := range f.FileData {
			c[n] = uint8(f.FileData[n])
		}
		if err := ioutil.WriteFile(f.Name, c, 0777); err != nil {
			return feedBackData, msg, fmt.Errorf("failed to write byte slice file: %w", err)
		}
		feedBackData.Response.Result.Result = "### Byte Slice of File ###"
		msg = append(msg, fmt.Sprintf("File was downloaded as '%s'. MimeType is '%s'.", f.Name, f.MimeType))
	} else {
		feedBackData.Response.Result.Result = "Server isn't installed or Wrong File ID."
	}
	return feedBackData, msg, nil
}
