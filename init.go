// Package main (init.go) :
// These methods are for reading and writing configuration file (ggsrun.cfg).
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/urfave/cli"
)

// GgsrunIni : Initialize ggsrun
func (a *AuthContainer) ggsrunIni(c *cli.Context) error {
	if cfgdata, err := a.chkInitFile(cfgFile); err == nil {
		err = json.Unmarshal(cfgdata, &a.GgsrunCfg)
		if err != nil {
			return fmt.Errorf("format error of '%s'", cfgFile)
		}
		if c.Command.Names()[0] == "exe1" ||
			c.Command.Names()[0] == "exe2" {
			if len(c.String("scriptid")) == 0 && len(a.GgsrunCfg.Scriptid) == 0 {
				return fmt.Errorf("no script id. Please use option '-i [Script ID]'")
			}
			if len(c.String("scriptid")) > 0 {
				a.GgsrunCfg.Scriptid = c.String("scriptid")
				a.InitVal.update = true
			}
			if len(c.String("function")) > 0 {
				a.Param.Function = c.String("function")
			}
		}
	} else {
		return a.readClientSecret()
	}
	return nil
}

// readClientSecret : Read client secret file
func (a *AuthContainer) readClientSecret() error {
	if csecret, err := a.chkInitFile(clientsecretFile); err == nil {
		err := json.Unmarshal(csecret, &a.Cs)
		if err != nil || (len(a.Cs.Cid.ClientID) == 0 && len(a.Cs.Ciw.ClientID) == 0) {
			return fmt.Errorf("please confirm '%s'. Error is %s", clientsecretFile, err)
		}
		if len(a.Cs.Cid.ClientID) == 0 && len(a.Cs.Ciw.ClientID) > 0 {
			a.Cs.Cid = a.Cs.Ciw
		}
	} else {
		return fmt.Errorf("no materials for retrieving accesstoken. Please download '%s'", clientsecretFile)
	}
	return nil
}

// chkInitFile : Check initial files.
// By this method, at first, files are searched in working directory, and next, they are searched in the directory declared by the environment variable.
func (a *AuthContainer) chkInitFile(file string) ([]byte, error) {
	var err error
	var body []byte
	if body, err = ioutil.ReadFile(filepath.Join(a.InitVal.workdir, file)); err == nil {
		a.InitVal.usedDir = "work"
		return body, err
	}
	if a.InitVal.workdir != a.InitVal.cfgdir {
		if body, err = ioutil.ReadFile(filepath.Join(a.InitVal.cfgdir, file)); err == nil {
			a.InitVal.usedDir = "env"
			return body, err
		}
	}
	return nil, fmt.Errorf("error: %s was not found", file)
}