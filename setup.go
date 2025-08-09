// Package main (setup.go) :
// These methods are for interactive setup.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"ggsrun/utl"

	"github.com/urfave/cli"
)

const gcpProjectURL = "https://console.cloud.google.com/projectcreate"
const gcpCredURL = "https://console.cloud.google.com/apis/credentials"
const scriptEditorURL = "https://script.google.com/d/%s/edit"

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
              res = emessage.call(this, "Script Error on GAS side: " + err.message, startTime);
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