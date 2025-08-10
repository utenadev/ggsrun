// Package main (handler.go) :
// Handler for ggsrun
package main

import (
	"encoding/json"
	"fmt"
	"os"

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

	if err := doGoauth(initVal, ggsrunCfg, cs); err != nil {
		return err
	}

	if err := doExe1Function(c, initVal, resMsg, ggsrunCfg, param); err != nil {
		return err
	}

	resMsg.Msg = doExecutionAPIwithoutServer(param, resMsg.Msg)

	// TODO: Refactor remaining chain to use DI
	// The following lines still rely on ExecutionContainer methods
	// which need to be refactored into standalone functions.
	e, err := newExecutionContainer(initVal, resMsg, ggsrunCfg, param)
	if err != nil {
		return err
	}

	feedBackData, msg, err := doEsenderForExe1(c, param, ggsrunCfg, initVal.pstart, resMsg.Msg)
	if err != nil {
		return err
	}
	resMsg.Msg = msg
	e.FeedBackData = feedBackData

	doDispResult(c, e.FeedBackData, resMsg.Msg)
	return nil
}

// exeAPIWith : exe2
// No update project. Only execute GAS using Execution API with server script.
func exeAPIWith(c *cli.Context) error {
	defAuthContainer(c).
		ggsrunIni(c).
		goauth().
		defExecutionContainer().
		exe2Function(c).
		dispResult(c)
	return nil
}

// webAppsWith : exe3
// No update project. Only execute GAS using Web Apps with server script.
func webAppsWith(c *cli.Context) error {
	defExecutionContainerWebApps().
		webAppswithServerForExe3(utl.ConvGasToRun(c), c).
		dispResult(c)
	return nil
}

// downloadFiles : Download files from Google Drive.
func downloadFiles(c *cli.Context) error {
	res := defAuthContainer(c).
		ggsrunIni(c).
		goauth().
		defDownloadContainer(c).
		GetFileinf().
		Downloader(c)
	dispTransferResult(c, res)
	return nil
}

// uploadFiles : Uploads files
func uploadFiles(c *cli.Context) error {
	res := defAuthContainer(c).
		ggsrunIni(c).
		goauth().
		defUploadContainer(c).
		Uploader(c)
	dispTransferResult(c, res)
	return nil
}

// updateProject : Updates projects and scripts
func updateProject(c *cli.Context) error {
	res := defAuthContainer(c).
		ggsrunIni(c).
		goauth().
		defExecutionContainer().
		projectUpdateControl(c)
	dispTransferResult(c, res)
	return nil
}

// revisionFiles : Retrieves revision IDs and downloads revision files.
func revisionFiles(c *cli.Context) error {
	res := defAuthContainer(c).
		ggsrunIni(c).
		goauth().
		defDownloadContainer(c).
		GetRevisionList(c)
	dispTransferResult(c, res)
	return nil
}

// showFileList : Shows file list on Google Drive
func showFileList(c *cli.Context) error {
	res := defAuthContainer(c).
		ggsrunIni(c).
		goauth().
		defDownloadContainer(c).
		GetFileList(c)
	dispTransferResult(c, res)
	return nil
}

// searchFilesByQueryAndRegex : Search files on Google Drive using search query and regex.
func searchFilesByQueryAndRegex(c *cli.Context) error {
	res := defAuthContainer(c).
		ggsrunIni(c).
		goauth().
		defDownloadContainer(c).
		SearchFiles()
	dispTransferResult(c, res)
	return nil
}

// managePermissions : Manage permissions.
func managePermissions(c *cli.Context) error {
	res := defAuthContainer(c).
		ggsrunIni(c).
		goauth().
		defPermissionsContainer(c).
		ManagePermissions()
	dispTransferResult(c, res)
	return nil
}

// getDriveInformation : Get drive information.
func getDriveInformation(c *cli.Context) error {
	res := defAuthContainer(c).
		ggsrunIni(c).
		goauth().
		defDownloadContainer(c).
		GetDriveInformation()
	dispTransferResult(c, res)
	return nil
}

// reAuth : Retrieve tokens again.
func reAuth(c *cli.Context) error {
	defAuthContainer(c).
		ggsrunIni(c).
		reAuth()
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
