package main

import (
	"github.com/cheggaaa/pb"
	_ "github.com/hornbill/goApiLib"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	version              = "1.0.1"
	applicationName      = "Hornbill Email Attachment Archiver"
	appName              = "goEmailAttachmentArchiver"
	execName             = "goEmailAttachmentArchiver"
	globalUltimateCutOff = 12
	globalDefaultCutOff  = 0
)

var (
	localLogFileName         = "EAA_"
	espLogFileName           = "EmailAttachmentArchiver"
	configFileName           string
	configDryRun             bool
	configOverride           bool
	configForceDelete        bool
	configOutputFolder       string
	configCutOff             int
	configPageSize           = 100
	globalMaxRoutines        int
	globalAPIKeys            []string
	globalMailFolders        []int
	globalTimeNow            string
	globalCutOffDate         string
	globalAttachmentLocation = ""
	importConf               importConfStruct
	startTime                time.Time
	endTime                  time.Duration
	wg                       sync.WaitGroup
	client                   = http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   600 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   1,
			Proxy:                 http.ProxyFromEnvironment,
		},
		//Timeout: time.Duration(120 * time.Second),
	}

	mutex                = &sync.Mutex{}
	globalArrayEmails    []string
	globalArrayProcessed []string
	globalBarEmails      *pb.ProgressBar
	globalArrayBars      []*pb.ProgressBar
	//globalArrayLinks     []*apiLib.XmlmcInstStruct
)

//----- Config Data Structs
type importConfStruct struct {
	InstanceID       string
	APIKeys          []string
	AttachmentFolder string
	Mailboxes        []string
	Folders          []int
}
