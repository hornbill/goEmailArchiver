package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	_ "strings"
	_ "time"

	"github.com/hornbill/goHornbillHelpers"
)

//loadConfig -- Function to Load Configruation File
func loadConfig() (importConfStruct, bool) {
	boolLoadConf := true
	//-- Check Config File File Exists
	cwd, _ := os.Getwd()
	configurationFilePath := cwd + "/" + configFileName
	logger(1, "Loading Config File: "+configurationFilePath, false)
	if _, fileCheckErr := os.Stat(configurationFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "No Configuration File", true)
		os.Exit(102)
	}
	//-- Load Config File
	file, fileError := os.Open(configurationFilePath)
	//-- Check For Error Reading File
	if fileError != nil {
		logger(4, "Error Opening Configuration File: "+fmt.Sprintf("%v", fileError), true)
		boolLoadConf = false
	}

	//-- New Decoder
	decoder := json.NewDecoder(file)
	//-- New Var based on importConfStruct
	edbConf := importConfStruct{}
	//-- Decode JSON
	err := decoder.Decode(&edbConf)
	//-- Error Checking
	if err != nil {
		logger(4, "Error Decoding Configuration File: "+fmt.Sprintf("%v", err), true)
		boolLoadConf = false
	}
	//-- Return New Config
	return edbConf, boolLoadConf
}

//parseFlags - grabs and parses command line flags
func parseFlags() {
	flag.StringVar(&configFileName, "file", "conf.json", "Name of the configuration file to load")
	flag.StringVar(&configOutputFolder, "output", "", "Folder to store downloads in - overrides AttachmentFolder from the conf.json")
	flag.BoolVar(&configDryRun, "dryrun", false, "Do not delete the files from the server")
	flag.IntVar(&configCutOff, "cutoff", globalDefaultCutOff, "Set the cut off date in weeks ("+strconv.Itoa(globalUltimateCutOff)+" or greater)")
	flag.IntVar(&configPageSize, "pagesize", 100, "Set the Query Page Size (default: 100)")
	flag.BoolVar(&configOverride, "override", false, "Set this to true to override the "+strconv.Itoa(globalUltimateCutOff)+" week limit")
	flag.Parse()
}

func logger(t int, s string, outputToCLI bool) {
	hornbillHelpers.Logger(t, s, outputToCLI, localLogFileName)
}

/*
func parseDateTime(dateTime string) string {
	layout := "2006-01-02 15:04:05"
	t, err := time.Parse(layout, dateTime)
	if err != nil && dateTime != "" {
		//return first 19 chars of string
		dateString := dateTime[0:18]
		return dateString

	}
	return fmt.Sprintf("%s", t)
}
*/
/*
func loggerGen(t int, s string) string {

	var errorLogPrefix = ""
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
	case 2:
		errorLogPrefix = "[MESSAGE] "
	case 3:
		errorLogPrefix = ""
	case 4:
		errorLogPrefix = "[ERROR] "
	case 5:
		errorLogPrefix = "[WARNING] "
	case 6:
		errorLogPrefix = ""
	}
	return errorLogPrefix + s + "\n\r"
}
*/
/*
func loggerWriteBuffer(s string) {
	if s != "" {
		logLines := strings.Split(s, "\n\r")
		for _, line := range logLines {
			if line != "" {
				logger(0, line, false)
			}
		}
	}
}
*/
