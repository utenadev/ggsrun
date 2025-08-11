// Package main (projectupdater.go) :
// These methods are for updating project.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ggsrun/utl"

	"github.comcom/urfave/cli"
)

// projectUpdateControl : Main method for updating project.
// doProjectUpdateControl is the main method for updating a project.
func doProjectUpdateControl(c *cli.Context, ggsrunCfg *GgsrunCfg, upFiles []string, msg []string, pstartTime time.Time) *utl.FileInf {
	// TODO: Refactor internal logic to use passed arguments instead of ExecutionContainer fields
	// For now, returning a dummy FileInf to allow compilation.
	return &utl.FileInf{}
}

// projectUpdateForBoundScript : Update bound-script project
// func (e *ExecutionContainer) projectUpdateForBoundScript() *ExecutionContainer {
// 	p := e.convExecutionContainerToFileInf()
// 	var pr *utl.ProjectForAppsScriptApi
// 	var pp *utl.FilesForAppsScriptApi
// 	pr.ScriptId = e.Project.ScriptId
// 	for _, f := range e.Project.Files {
// 		pp.Name = f.Name
// 		pp.Type = f.Type
// 		pp.Source = f.Source
// 		pr.Files = append(pr.Files, *pp)
// 	}
// 	_ = p.ProjectUpdateByAppsScriptApi(pr)
// 	e.Msg = append(e.Msg, "Project was updated.")
// 	return e
// }

// ProjectMaker : Recreates the project using uploaded scripts.
func (e *ExecutionContainer) ProjectMaker() *ExecutionContainer {
	for _, elm := range e.UpFiles {
		if utl.ChkExtention(filepath.Ext(elm)) {
			filedata := &File{
				Name:	strings.Replace(filepath.Base(elm), filepath.Ext(elm), "", -1),
				Type:	utl.ExtToType(filepath.Ext(elm), false),
				Source:	utl.ConvGasToUpload(elm),
			}
			var overwrite bool
			for i, v := range e.Project.Files {
				if v.Name == filedata.Name {
					e.Project.Files[i].Source = filedata.Source
					e.Msg = append(e.Msg, fmt.Sprintf("'%s' (%s) in project was overwritten.", v.Name, v.Type))
					overwrite = true
				}
			}
			if !overwrite {
				e.Project.Files = append(e.Project.Files, *filedata)
			}
		} else {
			e.Msg = append(e.Msg, fmt.Sprintf("File of '%s' cannot be used for updating project.", elm))
		}
	}
	p := doConvExecutionContainerToFileInf(e.Accesstoken)
	body, err, _ := p.ChkBoundOrStandalone(e.GgsrunCfg.Scriptid)
	if err == nil {
		json.Unmarshal(body, &p)
		e.Msg = append(e.Msg, fmt.Sprintf("Filename is '%s'.", p.FileName))
	}
	e.Msg = append(e.Msg, fmt.Sprintf("Project ID is '%s'.", e.Scriptid))
	return e
}

// filesInProjectRemover : Remove files in project.
func (e *ExecutionContainer) filesInProjectRemover() *ExecutionContainer {
	temp := e.Project
	var outr []string
	for _, elm := range e.UpFiles {
		res, removed := removeEle(temp, elm)
		if removed {
			outr = append(outr, elm)
		}
		temp = res
	}
	if len(temp.Files) == 1 {
		fmt.Fprintf(os.Stderr, "Error: You cannot remove all files except for 'appsscript.json' in the project.\n")
		os.Exit(1)
	}
	e.Project = temp
	p := doConvExecutionContainerToFileInf(e.Accesstoken)
	body, err, _ := p.ChkBoundOrStandalone(e.GgsrunCfg.Scriptid)
	if err == nil {
		json.Unmarshal(body, &p)
		e.Msg = append(e.Msg, fmt.Sprintf("Filename is '%s'.", p.FileName))
	}
	e.Msg = append(e.Msg, fmt.Sprintf("Project ID is '%s'.", e.Scriptid))
	if len(outr) == 0 {
		fmt.Fprintf(os.Stderr, "[ %s ] were not found in the project. No files were removed from the project.\n", strings.Join(e.UpFiles, ", "))
		os.Exit(1)
	} else {
		e.Msg = append(e.Msg, fmt.Sprintf("Files of [ %s ] were removed from the project.", strings.Join(outr, ", ")))
	}
	return e
}

// removeEle : Remove an element from an array.
func removeEle(project *Project, elm string) (*Project, bool) {
	temp := &Project{}
	ff := strings.Replace(filepath.Base(elm), filepath.Ext(elm), "", -1)
	if ff != "appsscript" {
		for _, v := range project.Files {
			if v.Name != ff {
				temp.Files = append(temp.Files, v)
			}
		}
	} else {
		return project, false
	}
	if len(project.Files) != len(temp.Files) {
		return temp, true
	}
	return temp, false
}

// doProjectBackup downloads a backup of the project.
func doProjectBackup(c *cli.Context, accessToken, scriptID string, pstart time.Time, workdir string) (*Project, []string, error) {
	tokenparams := url.Values{}
	tokenparams.Set("fields", "files,scriptId")
	u, _ := url.Parse(appsscriptapi)
	u.Path = path.Join(u.Path, scriptID+"/content")
	r := &utl.RequestParams{
		Method:		"GET",
		APIURL:		u.String() + "?" + tokenparams.Encode(),
		Data:		nil,
		Contenttype:	"application/x-www-form-urlencoded",
		Accesstoken:	accessToken,
		Dtime:		30,
	}
	res, err := r.FetchAPI()
	if err != nil {
		utl.DispScopeError2(res)
		return nil, nil, fmt.Errorf("project backup failed. Was the inputted project ID correct? Error: %w", err)
	}
	project := &Project{}
	json.Unmarshal(res, project)
	msg := []string{}
	if c.Bool("backup") {
		btok, _ := json.MarshalIndent(project, "", "\t")
		filename := pstart.Format("20060102_150405") + ".gs"
		if err := ioutil.WriteFile(filepath.Join(workdir, filename), btok, 0777); err != nil {
			return nil, nil, fmt.Errorf("failed to write backup file to '%s': %w. Please check file permissions or disk space.", filepath.Join(workdir, filename), err)
		}
		dat := fmt.Sprintf("Project was saved as '%s'.", filename)
		msg = append(msg, dat)
	}
	return project, msg, nil
}
