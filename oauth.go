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

	"github.com/urfave/cli"
	gettokenbyserviceaccount "github.com/tanaikech/go-gettokenbyserviceaccount"
)

// doGoauth manages the OAuth2 flow.
func doGoauth(initVal *InitVal, ggsrunCfg *GgsrunCfg, cs *Cs, resMsg *ResMsg) error {
	if initVal.useServiceAccount != "" {
		accessToken, err := getAtFromSa(initVal.useServiceAccount, ggsrunCfg.Scopes)
		if err != nil {
			return fmt.Errorf("could not get access token from service account: %w", err)
		}
		ggsrunCfg.Accesstoken = accessToken
		resMsg.Msg = append(resMsg.Msg, "Service Account was used.")
		return nil
	}

	if len(ggsrunCfg.Refreshtoken) > 0 {
		if (initVal.pstart.Unix()-ggsrunCfg.Expiresin) > 0 || len(ggsrunCfg.Accesstoken) == 0 {
			if err := getAtoken(ggsrunCfg); err != nil {
				return err
			}
			if err := makecfgfile(ggsrunCfg, initVal); err != nil {
				return err
			}
			resMsg.Msg = append(resMsg.Msg, "Access Token was refreshed.")
			return nil
		} else if initVal.update {
			if err := makecfgfile(ggsrunCfg, initVal); err != nil {
				return err
			}
			resMsg.Msg = append(resMsg.Msg, "Access Token was updated.")
			return nil
		}
	} else {
		if err := getNewAccesstoken(initVal, ggsrunCfg, cs); err != nil {
			return err
		}
		if err := makecfgfile(ggsrunCfg, initVal); err != nil {
			return err
		}
		resMsg.Msg = append(resMsg.Msg, "New Access Token was retrieved.")
		return nil
	}
	resMsg.Msg = append(resMsg.Msg, "Access Token was used.")
	return nil
}

// doReAuth performs a full re-authentication.
func doReAuth(initVal *InitVal, ggsrunCfg *GgsrunCfg, cs *Cs) error {
	if err := getNewAccesstoken(initVal, ggsrunCfg, cs); err != nil {
		return err
	}
	return makecfgfile(ggsrunCfg, initVal)
}

// newAuthContainerForReAuth initializes authentication-related structs for re-authentication.
func newAuthContainerForReAuth(c *cli.Context) (*InitVal, *GgsrunCfg, *Cs, error) {
	initVal := &InitVal{}
	ggsrunCfg := &GgsrunCfg{}
	cs := &Cs{}

	var err error
	initVal.pstart = time.Now()
	initVal.workdir, err = os.Getwd()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not get working directory: %w", err)
	}
	initVal.cfgdir, err = getConfigDir()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not get config directory: %w", err)
	}
	initVal.useServiceAccount = c.String("serviceaccount")
	initVal.log = c.Bool("log")

	return initVal, ggsrunCfg, cs, nil
}

// makecfgfile creates the ggsrun.cfg file.
func makecfgfile(ggsrunCfg *GgsrunCfg, initVal *InitVal) error {
	btok, _ := json.MarshalIndent(ggsrunCfg, "", "\t")
	var path string
	if initVal.usedDir == "work" {
		path = initVal.workdir
	} else if initVal.usedDir == "env" {
		path = initVal.cfgdir
	} else {
		return fmt.Errorf("configuration directory was not found: '%s'", initVal.usedDir)
	}
	return ioutil.WriteFile(filepath.Join(path, cfgFile), btok, 0777)
}

// getAtoken retrieves a new access token from a refresh token.
func getAtoken(ggsrunCfg *GgsrunCfg) error {
	values := url.Values{}
	values.Set("client_id", ggsrunCfg.Clientid)
	values.Set("client_secret", ggsrunCfg.Clientsecret)
	values.Set("refresh_token", ggsrunCfg.Refreshtoken)
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
	var atoken Atoken
	json.Unmarshal(body, &atoken)
	ggsrunCfg.Accesstoken = atoken.Accesstoken
	exp, err := chkAtoken(ggsrunCfg.Accesstoken)
	if err != nil {
		return err
	}
	ggsrunCfg.Expiresin = exp - 360 // 6 minutes as adjustment time
	return nil
}

// chkAtoken checks if an access token is valid.
func chkAtoken(accessToken string) (int64, error) {
	r := &utl.RequestParams{
		Method:      "GET",
		APIURL:      chkatutl + "tokeninfo?access_token=" + accessToken,
		Data:        nil,
		Contenttype: "application/x-www-form-urlencoded",
		Accesstoken: "",
		Dtime:       10,
	}
	body, err := r.FetchAPI()
	if err != nil {
		return 0, err
	}
	var chkAt ChkAt
	json.Unmarshal(body, &chkAt)
	if len(chkAt.Error) > 0 {
		return 0, fmt.Errorf("access token is invalid: %s", chkAt.Error)
	}
	return strconv.ParseInt(chkAt.Exp, 10, 64)
}

// doChkAtokenForExecution checks the validity of an access token for the Execution API.
func doChkAtokenForExecution(accessToken string) (*ChkAt, error) {
	r := &utl.RequestParams{
		Method:      "GET",
		APIURL:      chkatutl + "tokeninfo?access_token=" + accessToken,
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

// getCode retrieves the authorization code from Google.
func getCode(initVal *InitVal, scopes []string, cs *Cs) (string, error) {
	p := initVal.Port
	var hasLocalhost bool
	for _, e := range cs.Cid.Redirecturis {
		if strings.Contains(e, "localhost") {
			hasLocalhost = true
			break
		}
	}
	if !hasLocalhost {
		return "", fmt.Errorf("go manual mode")
	}

	fmt.Printf("\n### This is a automatic input mode.\n### Please follow opened browser, login Google and click authentication.\n### It will move to a manual mode if you wait for 30 seconds under this situation.\n")
	redirectURI := "http://localhost:" + strconv.Itoa(p) + "/"
	codepara := url.Values{}
	codepara.Set("client_id", cs.Cid.ClientID)
	codepara.Set("redirect_uri", redirectURI)
	codepara.Set("scope", strings.Join(scopes, " "))
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

// getNewAccesstoken retrieves a new access and refresh token.
func getNewAccesstoken(initVal *InitVal, ggsrunCfg *GgsrunCfg, cs *Cs) error {
	var code string
	var err error
	fmt.Printf("\n### Since %s is not found, the authorization process is launched.", cfgFile)
	redirectURI := cs.Cid.Redirecturis[0]
	code, err = getCode(initVal, ggsrunCfg.Scopes, cs)
	if err != nil {
		codepara := url.Values{}
		codepara.Set("client_id", cs.Cid.ClientID)
		codepara.Set("redirect_uri", redirectURI)
		codepara.Set("scope", strings.Join(ggsrunCfg.Scopes, " "))
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
	}
	tokenparams := url.Values{}
	tokenparams.Set("client_id", cs.Cid.ClientID)
	tokenparams.Set("client_secret", cs.Cid.Clientsecret)
	tokenparams.Set("redirect_uri", redirectURI)
	tokenparams.Set("code", code)
	tokenparams.Set("grant_type", "authorization_code")
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
	var atoken Atoken
	json.Unmarshal(body, &atoken)
	ggsrunCfg.Clientid = cs.Cid.ClientID
	ggsrunCfg.Clientsecret = cs.Cid.Clientsecret
	ggsrunCfg.Refreshtoken = atoken.Refreshtoken
	ggsrunCfg.Accesstoken = atoken.Accesstoken
	exp, err := chkAtoken(ggsrunCfg.Accesstoken)
	if err != nil {
		return err
	}
	ggsrunCfg.Expiresin = exp - 360 // 6 minutes as adjustment time
	return nil
}

// getAtFromSa retrieves an access token from a service account.
func getAtFromSa(useServiceAccount string, scopes []string) (string, error) {
	credentialsData, err := ioutil.ReadFile(useServiceAccount)
	if err != nil {
		return "", err
	}
	para := struct {
		PrivateKey  string `json:"private_key"`
		ClientEmail string `json:"client_email"`
	}{}
	json.Unmarshal(credentialsData, &para)
	scope := strings.Join(scopes, " ")
	res, err := gettokenbyserviceaccount.Do(para.PrivateKey, para.ClientEmail, "", scope)
	if err != nil {
		return "", err
	}
	return res.AccessToken, nil
}
