// Package main (setup.go) :
// These methods are for interactive setup.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"ggsrun/utl"

	"github.com/urfave/cli"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const gcpProjectURL = "https://console.cloud.google.com/projectcreate"
const gcpCredURL = "https://console.cloud.google.com/apis/credentials"

// getConfigDir returns the path to the configuration directory.
func getConfigDir() (string, error) {
	var dir string
	switch runtime.GOOS {
	case "windows":
		dir = os.Getenv("APPDATA")
		if dir == "" {
			return "", fmt.Errorf("APPDATA environment variable is not set")
		}
	default: // For Mac, Linux, and other Unix-like systems.
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "ggsrun"), nil
}

// confirm asks a yes/no question to the user.
func confirm(s string) bool {
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [y/n]: ", s)
		res, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return false
			}
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			return false
		}
		res = strings.ToLower(strings.TrimSpace(res))
		if res == "y" || res == "yes" {
			return true
		} else if res == "n" || res == "no" {
			return false
		}
	}
}

// RunInteractiveSetup runs the interactive setup process.
func RunInteractiveSetup(c *cli.Context) error {
	fmt.Println("--- ggsrun Interactive Setup ---")

	workdir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get working directory: %w", err)
	}
	cfgdir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("could not get config directory: %w", err)
	}
	if _, _, err := chkInitFile(cfgFile, workdir, cfgdir); err == nil {
		if !confirm("ggsrun.cfg already exists. Do you want to overwrite it and start a new setup?") {
			fmt.Println("Setup aborted.")
			return nil
		}
	}

	fmt.Println("\nStep 1: Configure Client Secret")
	fmt.Println("This tool needs a `client_secret.json` file.")
	fmt.Printf("Please create and download it from:\n%s\n", gcpCredURL)
	var clientSecretData []byte
	var cs Cs
	for {
		fmt.Print("\nEnter path to client_secret.json: ")
		reader := bufio.NewReader(os.Stdin)
		path, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				return fmt.Errorf("input cancelled")
			}
			return fmt.Errorf("could not read path: %w", err)
		}
		clientSecretPath := strings.TrimSpace(path)
		if data, err := os.ReadFile(clientSecretPath); err == nil {
			if json.Unmarshal(data, &cs) != nil || (cs.Cid.ClientID == "" && cs.Ciw.ClientID == "") {
				fmt.Fprintf(os.Stderr, "Error: '%s' is not a valid client_secret.json file. Please try again.\n", clientSecretPath)
				continue
			}
			clientSecretData = data
			break
		}
		fmt.Fprintf(os.Stderr, "Error: File not found at '%s'. Please try again.\n", clientSecretPath)
	}
	if len(cs.Cid.ClientID) == 0 && len(cs.Ciw.ClientID) > 0 {
		cs.Cid = cs.Ciw
	}

	fmt.Println("\nStep 2: Authorize ggsrun")
	fmt.Println("Your browser will open to ask for authorization.")
	scopes := []string{
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/script.projects",
	}
	config, err := google.ConfigFromJSON(clientSecretData, scopes...)
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %v", err)
	}
	token, err := getTokenFromWeb(config)
	if err != nil {
		return fmt.Errorf("failed to retrieve token from web: %v", err)
	}
	fmt.Println("Authorization successful.")

	fmt.Println("\nStep 3: Create Google Apps Script Project")
	if !confirm("Create a new Google Apps Script project for the server script?") {
		fmt.Println("Setup aborted by user.")
		return nil
	}

	projectTitle := "ggsrun-server"
	createReq := &utl.RequestParams{
		Method:      "POST",
		APIURL:      "https://script.googleapis.com/v1/projects",
		Data:        strings.NewReader(fmt.Sprintf(`{"title": "%s"}`, projectTitle)),
		Contenttype: "application/json",
		Accesstoken: token.AccessToken,
	}
	project, err := createReq.FetchAPI()
	if err != nil {
		return fmt.Errorf("could not create Apps Script project: %w", err)
	}
	var createdProject utl.AppsScriptApiInf
	if err := json.Unmarshal(project, &createdProject); err != nil {
		return fmt.Errorf("failed to parse created project response: %w", err)
	}
	fmt.Printf("Project '%s' created successfully. Script ID: %s\n", createdProject.Title, createdProject.ScriptId)

	fmt.Println("\nStep 4: Upload Server Script")
	files := []File{
		{
			Name:   "server",
			Type:   "SERVER_JS",
			Source: serverScriptContent,
		},
		{
			Name:   "appsscript",
			Type:   "JSON",
			Source: `{"timeZone":"Asia/Tokyo","dependencies":{},"exceptionLogging":"STACKDRIVER"}`,
		},
	}
	uploadData, _ := json.Marshal(&Project{Files: files})
	updateReq := &utl.RequestParams{
		Method:      "PUT",
		APIURL:      fmt.Sprintf("https://script.googleapis.com/v1/projects/%s/content", createdProject.ScriptId),
		Data:        strings.NewReader(string(uploadData)),
		Contenttype: "application/json",
		Accesstoken: token.AccessToken,
	}
	if _, err := updateReq.FetchAPI(); err != nil {
		return fmt.Errorf("could not upload server script: %w", err)
	}
	fmt.Println("Server script uploaded successfully.")

	ggsrunCfg := &GgsrunCfg{
		Scriptid:     createdProject.ScriptId,
		Clientid:     cs.Cid.ClientID,
		Clientsecret: cs.Cid.Clientsecret,
		Refreshtoken: token.RefreshToken,
		Scopes:       scopes,
	}
	cfgData, _ := json.MarshalIndent(ggsrunCfg, "", "  ")
	cfgPath := filepath.Join(cfgdir, cfgFile)
	if err := os.WriteFile(cfgPath, cfgData, 0644); err != nil {
		return fmt.Errorf("failed to write config file to %s: %w", cfgPath, err)
	}
	fmt.Printf("\nConfiguration saved to %s\n", cfgPath)

	fmt.Println("\nStep 5: Enable APIs")
	fmt.Println("Please ensure that both 'Google Apps Script API' and 'Google Drive API' are enabled for your Cloud project.")
	fmt.Println("You can check their status and enable them if necessary at the following URL:")
	projectID := cs.Cid.Projectid
	if projectID != "" {
		fmt.Printf("  - Apps Script API: https://console.cloud.google.com/apis/library/script.googleapis.com?project=%s\n", projectID)
		fmt.Printf("  - Drive API: https://console.cloud.google.com/apis/library/drive.googleapis.com?project=%s\n", projectID)
	} else {
		fmt.Println("Could not determine Project ID from client_secret.json. Please visit your Google Cloud Console to enable APIs.")
	}
	fmt.Println("\nAlso, you need to enable 'Google Apps Script API' in the script editor settings:")
	fmt.Println("  - https://script.google.com/home/usersettings")

	fmt.Println("\nSetup complete! You can now use ggsrun.")
	return nil
}

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	config.RedirectURL = "http://localhost:8080"
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Println("Please open the following URL in your browser, then authorize ggsrun:")
	fmt.Println(authURL)
	if err := openBrowser(authURL); err != nil {
		log.Printf("Failed to open browser: %v. Please open the URL manually.", err)
	}

	codeCh := make(chan string)
	errCh := make(chan error)
	server := &http.Server{Addr: ":8080"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")
		if code == "" {
			fmt.Fprintf(w, "Error: No auth code received.")
			errCh <- fmt.Errorf("no auth code received in callback")
			return
		}
		fmt.Fprintf(w, "Authorization successful! You can close this tab.")
		codeCh <- code
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- fmt.Errorf("failed to start local server: %v", err)
		}
	}()

	var authCode string
	select {
	case code := <-codeCh:
		authCode = code
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("timed out waiting for authorization code")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Failed to shut down local server: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %v", err)
	}
	return tok, nil
}

func openBrowser(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}

// serverScriptContent holds the content of server/server.gs
const serverScriptContent = `
var VERSION = "1.0.0";
var IGGSRUN;

function ExecutionApi(e) {
  IGGSRUN = new ggsrun(e, null, []);
  return IGGSRUN.executionapi();
}

function WebApps(e, password) {
  IGGSRUN = new ggsrun(e, password, []);
  return IGGSRUN.webapps();
}

function Log(l) {
  IGGSRUN.logg(l);
}

function Beacon() {
  return new ggsrun(null, null, null).beacon();
}

(function(x) {
  var ggsrun;

  ggsrun = (function() {
      var emessage, wmessage, logsheet, recLog;
      ggsrun.help = "This is a server script for ggsrun using Execution API and Web Apps.";
      ggsrun.name = 'ggsrun';
  
      function ggsrun(e, pass, logar) {
        this.e = e;
        this.pass = pass;
        this.ss = logsheet("ggsrun.log");
        this.logar = logar;
      }

      ggsrun.prototype.beacon = function() {
        return "This is a server for ggsrun. Version is " + VERSION + ". Autor is https://github.com/tanaikech .";
      };
    
      ggsrun.prototype.logg = function(val) {
        this.logar.push(val);
      };
    
      ggsrun.prototype.nodocsdownloader = function() {
        return (function(id){
          try {
            var file = DriveApp.getFileById(id);
            return {
              result: file.getBlob().getBytes(),
              name: file.getName(),
              mimeType: file.getBlob().getContentType()
            };
          } catch(err) {
            return {
              result: "Error"
            };
          }
        })(this.e);
      };

      ggsrun.prototype.foldertree = function() {
          return (function(folder, folderSt, results){
              var ar = [];
              var folders = folder.getFolders();
              while(folders.hasNext()) ar.push(folders.next());
              folderSt += folder.getName() + "(" + folder.getId() + ")#_aabbccddee_#";
              var array_folderSt = folderSt.split("#_aabbccddee_#");
              array_folderSt.pop()
              results.push(array_folderSt);
              ar.length == 0 && (folderSt = "");
              for (var i in ar) arguments.callee(ar[i], folderSt, results);
              return results;
          })(DriveApp.getRootFolder(), "", []);
      };
    
      ggsrun.prototype.executionapi = function() {
          var startTime = Date.now();
          var dateDat = Utilities.formatDate(new Date(), "GMT", "yyyy-MM-dd_HH:mm:ss'_GMT'");
          var rec = JSON.parse(this.e);
          if (rec.log) {
              try {
                  recLog.call(this, [[
                      dateDat,
                      "{API: \"Execution API\", ContentLength: " + this.e.length + ", ExecutedFunction: \"" + rec.exefunc + "()\"}",
                      rec.com
                  ]]);
              } catch(err) {
                  var ss = err.message; // temporary
              }
          }
          var res = "";
          try {
              var resValues = (0,eval)((0,eval)(rec.com));
              res = emessage.call(this, rec.com ? resValues : "Error on GAS side: Bad parameters.", startTime, dateDat);
          } catch(err) {
              var errorDetail = {
                gasError: {
                  name: err.name,
                  message: err.message,
                  stack: err.stack,
                }
              };
              res = emessage.call(this, errorDetail, startTime);
          }
          return res;
      };
  
      ggsrun.prototype.webapps = function() {
          var startTime = Date.now();
          var dateDat = Utilities.formatDate(new Date(), "GMT", "yyyy-MM-dd_HH:mm:ss'_GMT'");
          if (this.e.parameters.log == "false" || !this.e.parameters.log) {
              try {
                  recLog.call(this, [[
                      dateDat,
                      "{API: \"Web Apps\", ContentLength: " + this.e.contentLength + ", Password: \"" + this.e.parameters.pass + "\"}",
                      this.e.parameters.com
                  ]]);
              } catch(err) {
                  var ss = err.message; // temporary
              }
          }
          var res = "";
          if (this.e.parameters.pass == this.pass) {
              try {
                  var resValues = (0,eval)((0,eval)(this.e.parameters.com[0]));
                  res = wmessage.call(this, this.e.parameters.com ? resValues : "Error on GAS side: Bad parameters.", startTime, dateDat);
              } catch(err) {
                  res = wmessage.call(this, "Script Error on GAS side: " + err.message, startTime);
              }
          } else {
              res = wmessage.call(this, "Error on GAS side: Bad password.", startTime);
          }
          return res;
      };
  
      emessage = function(data, startTime, dateDat) {
          return {
              result: data,
              logger: this.logar,
              GoogleElapsedTime: ((Date.now() - startTime) / 1000),
              ScriptDate: dateDat
          };
      };
  
      wmessage = function(data, startTime, dateDat) {
          return ContentService
              .createTextOutput(JSON.stringify({
                  result: data,
                  logger: this.logar,
                  GoogleElapsedTime: ((Date.now() - startTime) / 1000),
                  ScriptDate: dateDat
              }))
              .setMimeType(ContentService.MimeType.JSON);
      };
  
      logsheet = function(_log) {
          var logit = DriveApp.getFilesByName(_log);
          var logar = [];
          while (logit.hasNext()) {
              logar.push(logit.next().getId());
          }
          if (logar.length == 0) {
              var ss = SpreadsheetApp.create(_log);
              var logss = DriveApp.getFileById(ss.getId());
              try {
                  DriveApp.getFileById(ScriptApp.getScriptId()).getParents().next().addFile(logss);
              } catch(e) {
                  DriveApp.getFileById(SpreadsheetApp.getActiveSpreadsheet().getId()).getParents().next().addFile(logss);
              }
              logss.getParents().next().removeFile(logss);
          } else {
              var ss = SpreadsheetApp.openById(logar[0]);
          }
          return ss.getSheets()[0];
      };
  
      recLog = function(logAr) {
          this.ss.getRange(this.ss.getLastRow() + 1, 1, logAr.length, logAr[0].length).setValues(logAr);
      };
    
      return ggsrun;
  })();
  return x.ggsrun = ggsrun;
})(this);
`
