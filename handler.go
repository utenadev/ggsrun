// Package main (handler.go) :
// Handler for ggsrun
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"ggsrun/utl"

	"github.com/urfave/cli"
)

// exeAPIWithout : exe1
// Update project and Execution API withour server script.
func exeAPIWithout(c *cli.Context) error {
	initVal, resMsg, ggsrunCfg, param, cs, _, _, err := newAuthContainer(c)
	if err != nil {
		return err
	}

	ggsrunCfg, param, cs, err = doGgsrunIni(c, initVal)
	if err != nil {
		return err
	}

	if err := doGoauth(initVal, ggsrunCfg, cs, resMsg); err != nil {
		return err
	}

	if err := doExe1Function(c, initVal, resMsg, ggsrunCfg, param); err != nil {
		return err
	}

	resMsg.Msg = doExecutionAPIwithoutServer(param, resMsg.Msg)

	feedBackData, msg, err := doEsenderForExe1(c, param, ggsrunCfg, initVal.pstart, resMsg.Msg)
	if err != nil {
		return err
	}
	resMsg.Msg = msg

	doDispResult(c, feedBackData, resMsg.Msg)
	return nil
}

// exeAPIWith : exe2
// No update project. Only execute GAS using Execution API with server script.
func exeAPIWith(c *cli.Context) error {
	initVal, resMsg, ggsrunCfg, param, cs, _, _, err := newAuthContainer(c)
	if err != nil {
		return err
	}

	ggsrunCfg, param, cs, err = doGgsrunIni(c, initVal)
	if err != nil {
		return err
	}

	if err := doGoauth(initVal, ggsrunCfg, cs, resMsg); err != nil {
		return err
	}

	e, err := newExecutionContainer(initVal, resMsg, ggsrunCfg, param)
	if err != nil {
		return err
	}

	feedBackData, msg, dlFileByScript, err := doExe2Function(c, e.Param, e.GgsrunCfg, e.InitVal, e.Msg, e.DlFileByScript)
	if err != nil {
		return err
	}
	e.FeedBackData = feedBackData
	e.Msg = msg
	e.DlFileByScript = dlFileByScript

	doDispResult(c, e.FeedBackData, resMsg.Msg)
	return nil
}

// webAppsWith : exe3
// No update project. Only execute GAS using Web Apps with server script.
func webAppsWith(c *cli.Context) error {
	e, err := newExecutionContainerWebApps()
	if err != nil {
		return err
	}

	feedBackData, msg, dlFileByScript, err := doWebAppswithServerForExe3(utl.ConvGasToRun(c), c, e.InitVal.pstart)
	if err != nil {
		return err
	}
	e.FeedBackData = feedBackData
	e.Msg = msg
	e.DlFileByScript = dlFileByScript

	doDispResult(c, e.FeedBackData, e.Msg)
	return nil
}

// downloadFiles : Download files from Google Drive.
func downloadFiles(c *cli.Context) error {
	initVal, resMsg, ggsrunCfg, _, cs, _, _, err := newAuthContainer(c)
	if err != nil {
		return err
	}

	ggsrunCfg, _, cs, err = doGgsrunIni(c, initVal)
	if err != nil {
		return err
	}

	if err := doGoauth(initVal, ggsrunCfg, cs, resMsg); err != nil {
		return err
	}

	fileInf := newDownloadContainer(c, resMsg.Msg, ggsrunCfg.Accesstoken, initVal.workdir, initVal.useServiceAccount, initVal.pstart)

	res := fileInf.
		GetFileinf().
		Downloader(c)

	dispTransferResult(c, res)
	return nil
}

// uploadFiles : Uploads files
func uploadFiles(c *cli.Context) error {
	initVal, resMsg, ggsrunCfg, _, cs, _, _, err := newAuthContainer(c)
	if err != nil {
		return err
	}

	ggsrunCfg, _, cs, err = doGgsrunIni(c, initVal)
	if err != nil {
		return err
	}

	if err := doGoauth(initVal, ggsrunCfg, cs, resMsg); err != nil {
		return err
	}

	fileInf := newUploadContainer(c, resMsg.Msg, ggsrunCfg.Accesstoken, initVal.workdir, initVal.useServiceAccount, initVal.pstart)

	res := fileInf.Uploader(c)

	dispTransferResult(c, res)
	return nil
}

// updateProject : Updates projects and scripts
func updateProject(c *cli.Context) error {
	initVal, resMsg, ggsrunCfg, _, cs, _, _, err := newAuthContainer(c)
	if err != nil {
		return err
	}

	ggsrunCfg, _, cs, err = doGgsrunIni(c, initVal)
	if err != nil {
		return err
	}

	if err := doGoauth(initVal, ggsrunCfg, cs, resMsg); err != nil {
		return err
	}

	res := doProjectUpdateControl(c, ggsrunCfg, newUpdateProjectContainer(c), resMsg.Msg, initVal.pstart)
	dispTransferResult(c, res)
	return nil
}

// revisionFiles : Retrieves revision IDs and downloads revision files.
func revisionFiles(c *cli.Context) error {
	initVal, resMsg, ggsrunCfg, _, cs, _, _, err := newAuthContainer(c)
	if err != nil {
		return err
	}

	ggsrunCfg, _, cs, err = doGgsrunIni(c, initVal)
	if err != nil {
		return err
	}

	if err := doGoauth(initVal, ggsrunCfg, cs, resMsg); err != nil {
		return err
	}

	fileInf := newDownloadContainer(c, resMsg.Msg, ggsrunCfg.Accesstoken, initVal.workdir, initVal.useServiceAccount, initVal.pstart)
	res := utl.DoGetRevisionList(c, fileInf)
	dispTransferResult(c, res)
	return nil
}

// showFileList : Shows file list on Google Drive
func showFileList(c *cli.Context) error {
	initVal, resMsg, ggsrunCfg, _, cs, _, _, err := newAuthContainer(c)
	if err != nil {
		return err
	}

	ggsrunCfg, _, cs, err = doGgsrunIni(c, initVal)
	if err != nil {
		return err
	}

	if err := doGoauth(initVal, ggsrunCfg, cs, resMsg); err != nil {
		return err
	}

	fileInf := newDownloadContainer(c, resMsg.Msg, ggsrunCfg.Accesstoken, initVal.workdir, initVal.useServiceAccount, initVal.pstart)
	res := utl.DoGetFileList(c, fileInf)
	dispTransferResult(c, res)
	return nil
}

// searchFilesByQueryAndRegex : Search files on Google Drive using search query and regex.
func searchFilesByQueryAndRegex(c *cli.Context) error {
	initVal, resMsg, ggsrunCfg, _, cs, _, _, err := newAuthContainer(c)
	if err != nil {
		return err
	}

	ggsrunCfg, _, cs, err = doGgsrunIni(c, initVal)
	if err != nil {
		return err
	}

	if err := doGoauth(initVal, ggsrunCfg, cs, resMsg); err != nil {
		return err
	}

	fileInf := newDownloadContainer(c, resMsg.Msg, ggsrunCfg.Accesstoken, initVal.workdir, initVal.useServiceAccount, initVal.pstart)
	res := utl.DoSearchFiles(fileInf)
	dispTransferResult(c, res)
	return nil
}

// managePermissions : Manage permissions.
func managePermissions(c *cli.Context) error {
	initVal, resMsg, ggsrunCfg, _, cs, _, _, err := newAuthContainer(c)
	if err != nil {
		return err
	}

	ggsrunCfg, _, cs, err = doGgsrunIni(c, initVal)
	if err != nil {
		return err
	}

	if err := doGoauth(initVal, ggsrunCfg, cs, resMsg); err != nil {
		return err
	}

	fileInf := newPermissionsContainer(c, resMsg.Msg, ggsrunCfg.Accesstoken, initVal.workdir, initVal.useServiceAccount, initVal.pstart)
	res := utl.DoManagePermissions(fileInf)
	dispTransferResult(c, res)
	return nil
}

// getDriveInformation : Get drive information.
func getDriveInformation(c *cli.Context) error {
	initVal, resMsg, ggsrunCfg, _, cs, _, _, err := newAuthContainer(c)
	if err != nil {
		return err
	}

	ggsrunCfg, _, cs, err = doGgsrunIni(c, initVal)
	if err != nil {
		return err
	}

	if err := doGoauth(initVal, ggsrunCfg, cs, resMsg); err != nil {
		return err
	}

	fileInf := newDownloadContainer(c, resMsg.Msg, ggsrunCfg.Accesstoken, initVal.workdir, initVal.useServiceAccount, initVal.pstart)
	res := utl.DoGetDriveInformation(fileInf)
	dispTransferResult(c, res)
	return nil
}

// reAuth : Retrieve tokens again.
func reAuth(c *cli.Context) error {
	initVal, ggsrunCfg, cs, err := newAuthContainerForReAuth(c)
	if err != nil {
		return err
	}

	ggsrunCfg, _, cs, err = doGgsrunIni(c, initVal)
	if err != nil {
		return err
	}

	if err := doReAuth(initVal, ggsrunCfg, cs); err != nil {
		return err
	}
	fmt.Print("Done.")
	return nil
}

// interactiveSetup : New handler for 'init' command
func interactiveSetup(c *cli.Context) error {
	return RunInteractiveSetup(c)
}

// doDispResult formats and displays the final execution result to the user.
func doDispResult(c *cli.Context, feedBackData *FeedBackData, msg []string) {
	var dispRes []byte
	if len(msg) > 0 {
		feedBackData.Response.Result.Message = msg
	}
	if c.Bool("jsonparser") {
		dispRes, _ = json.MarshalIndent(feedBackData.Response.Result, "", "  ")
	} else {
		dispRes, _ = json.Marshal(feedBackData.Response.Result)
	}
	if c.Bool("onlyresult") {
		if c.Bool("jsonparser") {
			onlyres, _ := json.MarshalIndent(feedBackData.Response.Result.Result, "", "  ")
			fmt.Printf("%s\n", string(onlyres))
		} else {
			onlyres, _ := json.Marshal(feedBackData.Response.Result.Result)
			fmt.Printf("%s\n", string(onlyres))
		}
	} else {
		fmt.Printf("%v\n", string(dispRes))
	}
}

// dispTransferResult : Display result
func dispTransferResult(c *cli.Context, f *utl.FileInf) {
	var dispRes []byte
	if c.Bool("jsonparser") {
		dispRes, _ = json.MarshalIndent(f, "", "  ")
	} else {
		dispRes, _ = json.Marshal(f)
	}
	fmt.Printf("%s\n", string(dispRes))
}

// commandNotFound :
func commandNotFound(c *cli.Context, command string) {
	fmt.Fprintf(os.Stderr, "'%s' is not a %s command. Check '%s --help' or '%s -h'.", command, c.App.Name, c.App.Name, c.App.Name)
	os.Exit(2)
}

// TODO: The following functions are placeholders to fix build errors.
// The actual implementations should be provided.

func doExe1Function(c *cli.Context, initVal *InitVal, resMsg *ResMsg, ggsrunCfg *GgsrunCfg, param *Param) error {
	fmt.Println("Warning: doExe1Function is not implemented. This is a placeholder.")
	return nil
}

// TODO: The following are placeholders to allow compilation.
// They should be moved to their appropriate files (container.go, auth.go)
// and implemented correctly.

func newAuthContainerForReAuth(c *cli.Context) (*InitVal, *GgsrunCfg, *Cs, error) {
	fmt.Println("Warning: newAuthContainerForReAuth is a placeholder.")
	// This placeholder needs to return valid pointers to avoid panics.
	return new(InitVal), new(GgsrunCfg), new(Cs), nil
}

func doReAuth(initVal *InitVal, ggsrunCfg *GgsrunCfg, cs *Cs) error {
	fmt.Println("Warning: doReAuth is a placeholder.")
	return nil
}

