// Package main (oauth.go) : 
// Get accesstoken using refreshtoken, and confirm condition of accesstoken.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"ggsrun/utl"

	gettokenbyserviceaccount "github.com/tanaikech/go-gettokenbyserviceaccount"
)

// Goauth : 
func (a *AuthContainer) goauth() error {
	if a.useServiceAccount != "" {
		if err := a.getAtFromSa(); err != nil {
			return fmt.Errorf("could not get access token from service account: %w", err)
		}
		a.Msg = append(a.Msg, "Service Account was used.")
		return nil
	}
	if len(a.GgsrunCfg.Clientid) > 0 &&
		len(a.GgsrunCfg.Clientsecret) > 0 &&
		len(a.GgsrunCfg.Refreshtoken) > 0 {
		if (a.InitVal.pstart.Unix()-a.GgsrunCfg.Expiresin) > 0 ||
			len(a.GgsrunCfg.Accesstoken) == 0 {
			if err := a.getAtoken(); err != nil {
				return err
			}
			return a.makecfgfile()
		} else {
			if a.InitVal.update {
				return a.makecfgfile()
			}
		}
	} else {
		if err := a.readClientSecret(); err != nil {
			return err
		}
		if err := a.getNewAccesstoken(); err != nil {
			return err
		}
		return a.makecfgfile()
	}
	a.Msg = append(a.Msg, "Access Token was was used.")
	return nil
}

// ReAuth : 
func (a *AuthContainer) reAuth() error {
	if err := a.readClientSecret(); err != nil {
		return err
	}
	if err := a.getNewAccesstoken(); err != nil {
		return err
	}
	return a.makecfgfile()
}

// makecfgfile : 
func (a *AuthContainer) makecfgfile() error {
	btok, _ := json.MarshalIndent(a.GgsrunCfg, "", "\t")
	var path string
	if a.InitVal.usedDir == "work" {
		path = a.InitVal.workdir
	} else if a.InitVal.usedDir == "env" {
		path = a.InitVal.cfgdir
	} else {
		return fmt.Errorf("configuration directory was not found: '%s'", a.InitVal.usedDir)
	}
	return ioutil.WriteFile(filepath.Join(path, cfgFile), btok, 0777)
}

// getAtoken : Retrieves accesstoken from refreshtoken. 
func (a *AuthContainer) getAtoken() error {
	a.Msg = append(a.Msg, "Got a new accesstoken.")
	values := url.Values{}
	values.Set("client_id", a.GgsrunCfg.Clientid)
	values.Set("client_secret", a.GgsrunCfg.Clientsecret)
	values.Set("refresh_token", a.GgsrunCfg.Refreshtoken)
	values.Set("grant_type", "refresh_token")
	r := &utl.RequestParams{
		Method:      "POST",
		APIURL:      oauthurl + "token",
		Data:        strings.NewReader(values.Encode()),
		Contenttype: "application/x-www-form-urlencoded",
		Accesstoken: "",
		Dtime:       10,
	}
	body, err := r.FetchAPI()
	if err != nil {
		return fmt.Errorf("hint: If you use old ggsrun.cfg, please remove it and run 'ggsrun auth'. Then try again. API error: %w, Body: %s", err, body)
	}
	json.Unmarshal(body, &a.Atoken)
	a.GgsrunCfg.Accesstoken = a.Atoken.Accesstoken
	exp, err := a.chkAtoken()
	if err != nil {
		return err
	}
	a.GgsrunCfg.Expiresin = exp - 360 // 6 minutes as adjustment time
	return nil
}

// chkAtoken : For AuthContainer
func (a *AuthContainer) chkAtoken() (int64, error) {
	r := &utl.RequestParams{
		Method:      "GET",
		APIURL:      chkatutl + "tokeninfo?access_token=" + a.GgsrunCfg.Accesstoken,
		Data:        nil,
		Contenttype: "application/x-www-form-urlencoded",
		Accesstoken: "",
		Dtime:       10,
	}
	body, err := r.FetchAPI()
	if err != nil {
		return 0, err
	}
	json.Unmarshal(body, &a.ChkAt)
	if len(a.ChkAt.Error) > 0 {
		return 0, fmt.Errorf("access token is invalid: %s", a.ChkAt.Error)
	}
	return strconv.ParseInt(a.ChkAt.Exp, 10, 64)
}

// chkAtokenForExecution : For ExecutionContainer
func (e *ExecutionContainer) chkAtokenForExecution() (*ChkAt, error) {
	r := &utl.RequestParams{
		Method:      "GET",
		APIURL:      chkatutl + "tokeninfo?access_token=" + e.GgsrunCfg.Accesstoken,
		Data:        nil,
		Contenttype: "application/x-www-form-urlencoded",
		Accesstoken: "",
		Dtime:       10,
	}
	body, err := r.FetchAPI()
	if err != nil {
		return nil, err
	}
	var c ChkAt
	json.Unmarshal(body, &c)
	return &c, nil
}

func (a *AuthContainer) chkRedirectURI() bool {
	for _, e := range a.Cs.Cid.Redirecturis {
		if strings.Contains(e, "localhost") {
			return true
		}
	}
	return false
}

func (a *AuthContainer) getCode() (string, error) {
	p := a.InitVal.Port
	if !a.chkRedirectURI() {
		return "", fmt.Errorf("go manual mode")
	}
	fmt.Printf("\n### This is a automatic input mode.\n### Please follow opened browser, login Google and click authentication.\n### It will move to a manual mode if you wait for 30 seconds under this situation.\n")
	a.Cs.Cid.Redirecturis = append(a.Cs.Cid.Redirecturis, "http://localhost:"+strconv.Itoa(p)+"/")
	codepara := url.Values{}
	codepara.Set("client_id", a.Cs.Cid.ClientID)
	codepara.Set("redirect_uri", a.Cs.Cid.Redirecturis[len(a.Cs.Cid.Redirecturis)-1])
	codepara.Set("scope", strings.Join(a.GgsrunCfg.Scopes, " "))
	codepara.Set("response_type", "code")
	codepara.Set("approval_prompt", "force")
	codepara.Set("access_type", "offline")
	codeurl := oauthurl + "auth?" + codepara.Encode()
	s := &serverInfToGetCode{
		Response: make(chan authCode, 1),
		Start:    make(chan bool, 1),
		End:      make(chan bool, 1),
	}
	defer func() {
		s.End <- true
	}()
	go func(port int) {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if len(code) == 0 {
				fmt.Fprintf(w, `<html><head><title>ggsrun status</title></head><body><p>Erorr.</p></body></html>`)
				s.Response <- authCode{Err: fmt.Errorf("not found code")}
				return
			}
			fmt.Fprintf(w, `<html><head><title>ggsrun status</title></head><body><p>The authentication was done. Please close this page.</p></body></html>`)
			s.Response <- authCode{Code: code}
		})
		var err error
		Listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			s.Response <- authCode{Err: err}
			return
		}
		server := http.Server{}
		server.Handler = mux
		go server.Serve(Listener)
		s.Start <- true
		<-s.End
		Listener.Close()
		s.Response <- authCode{Err: err}
	}(p)
	<-s.Start
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", strings.Replace(codeurl, "&", `\&`, -1))
	case "linux":
		cmd = exec.Command("xdg-open", strings.Replace(codeurl, "&", `\&`, -1))
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", strings.Replace(codeurl, "&", `^&`, -1))
	default:
		return "", fmt.Errorf("go manual mode")
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("go manual mode")
	}
	var result authCode
	select {
	case result = <-s.Response:
	case <-time.After(time.Duration(30) * time.Second): // After 30 s, move to manual mode.
		return "", fmt.Errorf("go manual mode")
	}
	if result.Err != nil {
		return "", fmt.Errorf("go manual mode")
	}
	return result.Code, nil
}

// getNewAccesstoken : Retrieve accesstoken when there is no refreshtoken. 
func (a *AuthContainer) getNewAccesstoken() error {
	var code string
	var err error
	fmt.Printf("\n### Since %s is not found, the authorization process is launched.", cfgFile)
	code, err = a.getCode()
	if err != nil {
		codepara := url.Values{}
		codepara.Set("client_id", a.Cs.Cid.ClientID)
		codepara.Set("redirect_uri", a.Cs.Cid.Redirecturis[0])
		codepara.Set("scope", strings.Join(a.GgsrunCfg.Scopes, " "))
		codepara.Set("response_type", "code")
		codepara.Set("approval_prompt", "force")
		codepara.Set("access_type", "offline")
		codeurl := oauthurl + "auth?" + codepara.Encode()
		fmt.Printf("\n### This is a manual input mode.\n### Please input code retrieved by importing following URL to your browser.\n\n"+
			"[URL]==> %v\n"+
			"[CODE]==>", codeurl)
		if _, err := fmt.Scan(&code); err != nil {
			return fmt.Errorf("could not read code from stdin: %w", err)
		}
		a.Cs.Cid.Redirecturis = append(a.Cs.Cid.Redirecturis, a.Cs.Cid.Redirecturis[0])
	}
	tokenparams := url.Values{}
	tokenparams.Set("client_id", a.Cs.Cid.ClientID)
	tokenparams.Set("client_secret", a.Cs.Cid.Clientsecret)
	tokenparams.Set("redirect_uri", a.Cs.Cid.Redirecturis[len(a.Cs.Cid.Redirecturis)-1])
	tokenparams.Set("code", code)
	ttokenparams.Set("grant_type", "authorization_code")
	r := &utl.RequestParams{
		Method:      "POST",
		APIURL:      oauthurl + "token",
		Data:        strings.NewReader(tokenparams.Encode()),
		Contenttype: "application/x-www-form-urlencoded",
		Accesstoken: "",
		Dtime:       10,
	}
	body, err := r.FetchAPI()
	if err != nil {
		return fmt.Errorf("code is wrong: %w", err)
	}
	json.Unmarshal(body, &a.Atoken)
	a.GgsrunCfg.Clientid = a.Cs.Cid.ClientID
	a.GgsrunCfg.Clientsecret = a.Cs.Cid.Clientsecret
	a.GgsrunCfg.Refreshtoken = a.Atoken.Refreshtoken
	a.GgsrunCfg.Accesstoken = a.Atoken.Accesstoken
	exp, err := a.chkAtoken()
	if err != nil {
		return err
	}
	a.GgsrunCfg.Expiresin = exp - 360 // 6 minutes as adjustment time
	return nil
}

// getAtFromSa : Retrieve access token from Service Account
func (a *AuthContainer) getAtFromSa() error {
	credentialsData, err := ioutil.ReadFile(a.useServiceAccount)
	if err != nil {
		return err
	}
	para := struct {
		PrivateKey  string `json:"private_key"`
		ClientEmail string `json:"client_email"`
	}{}
	json.Unmarshal(credentialsData, &para)
	scopes := strings.Join(a.Scopes, " ")
	res, err := gettokenbyserviceaccount.Do(para.PrivateKey, para.ClientEmail, "", scopes)
	if err != nil {
		return err
	}
	a.GgsrunCfg.Accesstoken = res.AccessToken
	return nil
}