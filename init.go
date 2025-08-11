// Package main (init.go) :
// These methods are for reading and writing configuration file (ggsrun.cfg).
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli"
)

// doGgsrunIni handles the initialization by reading config files.
// It's a standalone function that returns the loaded configurations or an error.
func doGgsrunIni(c *cli.Context, initVal *InitVal) (*GgsrunCfg, *Param, *Cs, error) {
	ggsrunCfg := &GgsrunCfg{}
	param := &Param{}
	cs := &Cs{}

	// Try to read ggsrun.cfg
	cfgdata, usedDir, err := chkInitFile(cfgFile, initVal.workdir, initVal.cfgdir)
	initVal.usedDir = usedDir
	if err == nil {
		// If ggsrun.cfg is found, unmarshal it.
		if err := json.Unmarshal(cfgdata, ggsrunCfg); err != nil {
			return nil, nil, nil, fmt.Errorf("format error of '%s'", cfgFile)
		}
		// Populate params from command-line flags
		if c.Command.Names()[0] == "exe1" || c.Command.Names()[0] == "exe2" {
			if len(c.String("scriptid")) == 0 && len(ggsrunCfg.Scriptid) == 0 {
				return nil, nil, nil, fmt.Errorf("no script id. Please use option '-i [Script ID]'")
			}
			if len(c.String("scriptid")) > 0 {
				ggsrunCfg.Scriptid = c.String("scriptid")
				initVal.update = true
			}
			if len(c.String("function")) > 0 {
				param.Function = c.String("function")
			}
		}
	} else {
		// If ggsrun.cfg is not found, read client_secret.json
		csecret, usedDir, err := chkInitFile(clientsecretFile, initVal.workdir, initVal.cfgdir)
		initVal.usedDir = usedDir
		if err != nil {
			return nil, nil, nil, fmt.Errorf("no materials for retrieving accesstoken. Please download '%s'", clientsecretFile)
		}
		if err := json.Unmarshal(csecret, cs); err != nil || (len(cs.Cid.ClientID) == 0 && len(cs.Ciw.ClientID) == 0) {
			return nil, nil, nil, fmt.Errorf("please confirm '%s'. Error is %s", clientsecretFile, err)
		}
		if len(cs.Cid.ClientID) == 0 && len(cs.Ciw.ClientID) > 0 {
			cs.Cid = cs.Ciw
		}
	}
	return ggsrunCfg, param, cs, nil
}

// chkInitFile checks for a file in the working directory first, then the config directory.
// It returns the file content, the directory where the file was found, and an error.
func chkInitFile(file, workdir, cfgdir string) ([]byte, string, error) {
	// Check working directory
	if body, err := os.ReadFile(filepath.Join(workdir, file)); err == nil {
		return body, "work", nil
	}
	// Check config directory if it's different
	if workdir != cfgdir {
		if body, err := os.ReadFile(filepath.Join(cfgdir, file)); err == nil {
			return body, "env", nil
		}
	}
	return nil, "", fmt.Errorf("error: %s was not found", file)
}
