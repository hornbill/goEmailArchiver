package main

import (
	"bytes"
	"encoding/base64"
	_ "encoding/hex"
	"encoding/xml"
	_ "errors"
	"fmt"
	"github.com/cheggaaa/pb"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	_ "net/url"
	"os"
	_ "regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type stateStruct struct {
	Code     string `xml:"code"`
	ErrorRet string `xml:"error"`
}
type HBResults struct {
	HID              int    `xml:"h_pk_id"`
	HReqID           string `xml:"h_request_id"`
	HContentLocation string `xml:"h_contentlocation"`
	HFileName        string `xml:"h_filename"`
	HSize            int    `xml:"h_size"`
	HTimeStamp       string `xml:"h_timestamp"`
	HVisibility      string `xml:"h_visibility"`
	HCount           string `xml:"h_count"`
	HOwnerKey        string `xml:"h_owner_key"`
}

type structQueryResults struct {
	MethodResult string `xml:"status,attr"`
	Params       struct {
		RowData struct {
			Row []HBResults `xml:"row"`
		} `xml:"rowData"`
	} `xml:"params"`
	State stateStruct `xml:"state"`
}
type xmlmcCountResponse struct {
	Params struct {
		RowData struct {
			Row []struct {
				Count string `xml:"h_count"`
			} `xml:"row"`
		} `xml:"rowData"`
	} `xml:"params"`
	State stateStruct `xml:"state"`
}
type xmlmcMailIDResponse struct {
	Params struct {
		RowData struct {
			Row []struct {
				HMsgId string `xml:"h_msg_id"`
			} `xml:"row"`
		} `xml:"rowData"`
	} `xml:"params"`
	State        stateStruct `xml:"state"`
	MethodResult string      `xml:"status,attr"`
	/*	Params struct {
			SearchResultItem []struct {
				Relevance string `xml:"relevance"`
				Message struct {
					MessageID string `xml:"messageId"`
				} `xml:"message"`
			} `xml:"searchResultItem"`
		} `xml:"params"`
		State stateStruct `xml:"state"`
	*/
}
type structEmailResults struct {
	MethodResult string `xml:"status,attr"`
	Params       struct {
		Subject        string `xml:"subject"`
		Body           string `xml:"body"`
		HTMLBody       string `xml:"htmlBody"`
		Recipient      string `xml:"recipientName"`
		Sender         string `xml:"senderName"`
		Received       string `xml:"timeReceived"`
		Sent           string `xml:"timeSent"`
		RFCHeader      string `xml:"rfcHeader"`
		FileAttachment []struct {
			FileName  string `xml:"fileName"`
			FileSize  string `xml:"fileSize"`
			MIMEType  string `xml:"mimeType"`
			ContentID string `xml:"contentId"`
			CAFSToken string `xml:"cafsAccessToken"`
		} `xml:"fileAttachment"`
	} `xml:"params"`
	State stateStruct `xml:"state"`
}

type structMailFoldersResults struct {
	MethodResult string `xml:"status,attr"`
	Params       struct {
		Folder []struct {
			FolderId     int  `xml:"folderId"`
			HasSubFolder bool `xml:"hasSubFolders"`
		} `xml:"folder"`
	} `xml:"params"`
	State stateStruct `xml:"state"`
}

func appendToGlobalMailFolders(folderId int) {
	mutex.Lock()
	alreadyInList := false
	for _, v := range globalMailFolders {
		if v == folderId {
			alreadyInList = true
		}
	}
	if !alreadyInList {
		globalMailFolders = append(globalMailFolders, folderId)
	}
	mutex.Unlock()
}

func getAllMailFolders(mailbox string, folderId int) {

	localAPIKey := globalAPIKeys[0]
	espXmlmc := NewEspXmlmcSession(localAPIKey)
	logger(3, "Checking ["+mailbox+" - "+strconv.Itoa(folderId)+"]", false)
	espXmlmc.SetParam("mailbox", mailbox)
	if folderId != 0 {
		espXmlmc.SetParam("parentFolderId", strconv.Itoa(folderId))
	}
	XMLFolderResults, xmlmcErr := espXmlmc.Invoke("mail", "folderGetList")
	if xmlmcErr != nil {
		logger(4, "Something went wrong - getting folder list from ["+mailbox+"]", true)
		return
	}
	var xmlMailFoldersRespon structMailFoldersResults
	qerr := xml.Unmarshal([]byte(XMLFolderResults), &xmlMailFoldersRespon)
	if qerr != nil {
		fmt.Println("Something went wrong obtaining info from [" + mailbox + "]")
	} else {
		if xmlMailFoldersRespon.MethodResult != "fail" {
			folderCount := len(xmlMailFoldersRespon.Params.Folder)
			for i := 0; i < folderCount; i++ {
				appendToGlobalMailFolders(xmlMailFoldersRespon.Params.Folder[i].FolderId)
				if xmlMailFoldersRespon.Params.Folder[i].HasSubFolder {
					getAllMailFolders(mailbox, xmlMailFoldersRespon.Params.Folder[i].FolderId)
				}
			}
		} else {
			logger(4, xmlMailFoldersRespon.State.ErrorRet, true)
		}
	}

}

func getAllFolders() {

	amountMailBoxes := len(importConf.Mailboxes)
	for i := 0; i < amountMailBoxes; i++ {
		getAllMailFolders(importConf.Mailboxes[i], 0)
	}
	logger(1, "Checked "+strconv.Itoa(amountMailBoxes)+" mailboxes", true)

	amountFolders := len(importConf.Folders)
	for i := 0; i < amountFolders; i++ {
		appendToGlobalMailFolders(importConf.Folders[i])
	}

}

func populateRequestsArray() {

	localAPIKey := globalAPIKeys[0]
	localLink := NewEspXmlmcSession(localAPIKey)
	totalMailFolders := len(globalMailFolders)

	localLink.SetParam("application", "com.hornbill.core")
	localLink.SetParam("queryName", "systemEmails")
	localLink.OpenElement("queryParams")
	for i := 0; i < totalMailFolders; i++ {
		localLink.SetParam("folderId", strconv.Itoa(globalMailFolders[i]))
	}
	localLink.SetParam("msgDateFrom", "1970-01-01 00:00:00")
	localLink.SetParam("msgDateTo", globalCutOffDate+" 23:59:59")
	localLink.CloseElement("queryParams")
	localLink.OpenElement("queryOptions")
	localLink.SetParam("queryType", "count") //h_count
	localLink.CloseElement("queryOptions")

	RespBody, xmlmcErr := localLink.Invoke("data", "queryExec")
	var CountResp xmlmcCountResponse
	if xmlmcErr != nil {
		logger(4, "Unable to count emails: "+fmt.Sprintf("%v", xmlmcErr), true)
		return
	}
	err := xml.Unmarshal([]byte(RespBody), &CountResp)
	if err != nil {
		logger(4, "Unable to read Count "+fmt.Sprintf("%s", err), false)
		return
	}
	if CountResp.State.ErrorRet != "" {
		logger(4, "Unable to process Count "+CountResp.State.ErrorRet, false)
		return
	}

	//-- return Count
	count, errC := strconv.ParseUint(CountResp.Params.RowData.Row[0].Count, 10, 32)
	//-- Check for Error
	if errC != nil {
		logger(4, "Unable to get Count for Count Query "+fmt.Sprintf("%s", err), false)
		return
	} else {
		logger(3, "There are  "+fmt.Sprintf("%d", count)+" emails to be processed", false)
	}

	if count == 0 {
		return
	}

	var loopCount uint64

	bar := pb.StartNew(int(count))
	for loopCount < count {
		logger(1, "Loading Email List Offset: "+fmt.Sprintf("%d", loopCount)+"\n", false)

		localLink.SetParam("application", "com.hornbill.core")
		localLink.SetParam("queryName", "systemEmails")
		localLink.OpenElement("queryParams")
		for i := 0; i < totalMailFolders; i++ {
			localLink.SetParam("folderId", strconv.Itoa(globalMailFolders[i]))
		}
		localLink.SetParam("msgDateFrom", "1970-01-01 00:00:00")
		localLink.SetParam("msgDateTo", globalCutOffDate+" 23:59:59")
		localLink.SetParam("rowstart", strconv.FormatUint(loopCount, 10))
		localLink.SetParam("limit", strconv.Itoa(configPageSize))
		localLink.CloseElement("queryParams")
		localLink.OpenElement("queryOptions")
		localLink.SetParam("queryType", "records")
		localLink.CloseElement("queryOptions")
		localLink.OpenElement("queryOrder")
		localLink.SetParam("column", "h_msg_id")
		localLink.SetParam("direction", "ascending")
		localLink.CloseElement("queryOrder")

		XMLAttachmentSearch, xmlmcErr := localLink.Invoke("data", "queryExec")
		if xmlmcErr != nil {
			logger(6, "Unable to find Calls: "+fmt.Sprintf("%v", xmlmcErr), true)
			break
		}

		var xmlQuestionRespon xmlmcMailIDResponse
		qerr := xml.Unmarshal([]byte(XMLAttachmentSearch), &xmlQuestionRespon)

		if qerr != nil {
			fmt.Println("No Emails Found")
			fmt.Println(qerr)
			break
		} else {
			if xmlQuestionRespon.MethodResult == "fail" {
				fmt.Println(xmlQuestionRespon.State.ErrorRet)
				break
			}
			intResponseSize := len(xmlQuestionRespon.Params.RowData.Row)
			logger(3, "RowResults: "+strconv.Itoa(intResponseSize), false)

			for i := 0; i < intResponseSize; i++ {
				globalArrayRequests = append(globalArrayRequests, xmlQuestionRespon.Params.RowData.Row[i].HMsgId)
			}
		}

		// Add 100
		loopCount += uint64(configPageSize)
		bar.Add(len(xmlQuestionRespon.Params.RowData.Row))
		//-- Check for empty result set
		if len(xmlQuestionRespon.Params.RowData.Row) == 0 {
			break
		}

	}
	logger(3, "Found "+strconv.Itoa(len(globalArrayRequests))+" Emails", false)
	bar.FinishPrint("Requests Loaded \n")
}

/*

func populateRequestsArray(threadId int) {

	localAPIKey := globalAPIKeys[threadId]
	localLink := NewEspXmlmcSession(localAPIKey)

	localBar := globalArrayBars[threadId+1]

	for {
		boolIDExists, FolderID := pickOffMailFolder()

		if !boolIDExists {
			logger(3, "Finished Thread "+strconv.Itoa(threadId+1), false)
			break
		} else {
			folderId = strconv.Itoa(FolderID)
			logger(3, "Processing Folder: "+ folderId, false)

			localLink.SetParam("application", "com.hornbill.core")
			localLink.SetParam("queryName", "systemEmails")
			localLink.OpenElement("queryParams")
				localLink.SetParam("folderId", folderId)
				localLink.SetParam("msgDateFrom", "1970-01-01 00:00:00")
				localLink.SetParam("msgDateTo", globalCutOffDate + " 23:59:59")
			localLink.CloseElement("queryParams")
			localLink.OpenElement("queryOptions")
				localLink.SetParam("queryType", "count") //h_count
			localLink.CloseElement("queryOptions")

			XMLAttachmentSearch, xmlmcErr := espXmlmc.Invoke("data", "queryExec")
			if xmlmcErr != nil {
				logger(6, "Unable to find Calls: "+fmt.Sprintf("%v", xmlmcErr), true)
				break
			}




			localLink.SetParam("application", "com.hornbill.core")
			localLink.SetParam("queryName", "systemEmails")
			localLink.OpenElement("queryParams")
				localLink.SetParam("folderId", folderId)
				localLink.SetParam("msgDateFrom", "1970-01-01 00:00:00")
				localLink.SetParam("msgDateTo", globalCutOffDate + " 23:59:59")
				localLink.SetParam("rowstart", "0")
				localLink.SetParam("limit", "10")
			localLink.CloseElement("queryParams")
			localLink.OpenElement("queryOptions")
				localLink.SetParam("queryType", "records")
			localLink.CloseElement("queryOptions")
			localLink.OpenElement("queryOrder")
				localLink.SetParam("column", "h_msg_id")
				localLink.SetParam("direction", "ascending")
			localLink.CloseElement("queryOrder")

			XMLAttachmentSearch, xmlmcErr := espXmlmc.Invoke("data", "queryExec")
			if xmlmcErr != nil {
				logger(6, "Unable to find Calls: "+fmt.Sprintf("%v", xmlmcErr), true)
				break
			}

			var xmlQuestionRespon xmlmcMailIDResponse
			qerr := xml.Unmarshal([]byte(XMLAttachmentSearch), &xmlQuestionRespon)

			if qerr != nil {
				fmt.Println("No emails found")
				fmt.Println(qerr)
				//break
			} else {
				if xmlQuestionRespon.MethodResult == "fail" {
					fmt.Println(xmlQuestionRespon.State.ErrorRet)
					break
				}
				intResponseSize := len(xmlQuestionRespon.Params.SearchResultItem)
				logger(3, "RowResults: "+strconv.Itoa(intResponseSize), false)

				for i := 0; i < intResponseSize; i++ {
					if (xmlQuestionRespon.Params.SearchResultItem[i].Relevance[0:3] == "100") {
						globalArrayRequests = append(globalArrayRequests, xmlQuestionRespon.Params.SearchResultItem[i].Message.MessageID)
					}
				}
				//###globalArrayRequests = globalArrayRequests[1:100]
			}

		}

	}

	localBar.Finish()

	logger(3, "Found "+strconv.Itoa(len(globalArrayRequests))+" Emails", false)
}

func pickOffMailFolder() (bool, int) {
	boolReturn := false
	intLastItem := 0

	if len(globalMailFolders) > 0 {
		boolReturn = true
		mutex.Lock()
		intLastItem = globalMailFolders[len(globalMailFolders)-1]
		globalMailFolders[len(globalMailFolders)-1] = 0
		globalMailFolders = globalMailFolders[:len(globalMailFolders)-1]
		mutex.Unlock()
		globalArrayBars[0].Increment()
	}
	boolReturn = !(intLastItem == 0)
	return boolReturn, intLastItem
}


*/

func checkAPIKeys() bool {

	logger(3, "Checking API Keys", false)
	intAPIKeysLength := len(importConf.APIKeys)

	for i := 0; i < intAPIKeysLength; i++ {

		logger(3, "Checking API Key : "+importConf.APIKeys[i], false)

		espXmlmc := NewEspXmlmcSession(importConf.APIKeys[i])
		espXmlmc.SetParam("stage", "1")
		strAPIResult, xmlmcErr := espXmlmc.Invoke("system", "pingCheck")
		if xmlmcErr != nil {
			logger(4, "Failed PingCheck for : "+importConf.APIKeys[i], false)
		} else {
			var xmlQuestionRespon structQueryResults
			qerr := xml.Unmarshal([]byte(strAPIResult), &xmlQuestionRespon)
			if qerr != nil || xmlQuestionRespon.MethodResult == "fail" {
				//fmt.Println(strAPIResult)
				//fmt.Println(xmlQuestionRespon.State.ErrorRet)
				logger(5, "Found "+importConf.APIKeys[i]+" to be an invalid API key", true)
			} else {
				globalAPIKeys = append(globalAPIKeys, importConf.APIKeys[i])
			}
		}
	}

	logger(3, "Found "+strconv.Itoa(len(globalAPIKeys))+" valid API Keys", true)

	return len(globalAPIKeys) > 0
}

func pickOffRequestArray() (bool, string) {
	boolReturn := false
	stringLastItem := ""

	if len(globalArrayRequests) > 0 {
		boolReturn = true
		mutex.Lock()
		stringLastItem = globalArrayRequests[len(globalArrayRequests)-1]
		globalArrayRequests[len(globalArrayRequests)-1] = ""
		globalArrayRequests = globalArrayRequests[:len(globalArrayRequests)-1]
		mutex.Unlock()
		//globalBarRequests.Increment()
		globalArrayBars[0].Increment()
	}
	boolReturn = !(stringLastItem == "")
	return boolReturn, stringLastItem
}

func addToProcessedArray(processedRequestID string) {
	mutex.Lock()
	globalArrayProcessed = append(globalArrayProcessed, processedRequestID)
	mutex.Unlock()
}

func setOutputFolder() {
	localFolder := ""

	if importConf.AttachmentFolder != "" {
		localFolder = importConf.AttachmentFolder
	}
	if configOutputFolder != "" {
		localFolder = configOutputFolder
	}

	logger(2, "Checking "+localFolder, false)
	if src, err := os.Stat(localFolder); !os.IsNotExist(err) {
		//folder/file exists
		if !src.IsDir() {
			//not a directory
			logger(5, localFolder+" is not a folder.", true)
		} else {
			if src.Mode().Perm()&(1<<(uint(7))) == 0 {
				logger(5, "Write permission not set on this folder.", true)
			} else {
				globalAttachmentLocation = localFolder
			}
		}
	} else {
		logger(5, localFolder+" does not exist, trying to create folder", true)
		err := os.Mkdir(localFolder, 0777)
		if err == nil {
			//folder creation successful, so use created folder
			globalAttachmentLocation = localFolder
		}

	}

	if globalAttachmentLocation == "" {
		logger(2, "Using current folder for attachments", false)
		globalAttachmentLocation = "."
	}

	logger(2, "Using: "+globalAttachmentLocation, false)

}

func base64Wrap(w io.Writer, b []byte) {
	// 57 raw bytes per 76-byte base64 line.
	const maxRaw = 57
	// Buffer for each line, including trailing CRLF.
	//buffer := make([]byte, MaxLineLength+len("\r\n"))
	//buffer := make([]byte, 76+len("\r\n"))
	buffer := make([]byte, 78)
	//copy(buffer[MaxLineLength:], "\r\n")
	copy(buffer[76:], "\r\n")
	// Process raw chunks until there's no longer enough to fill a line.
	for len(b) >= maxRaw {
		base64.StdEncoding.Encode(buffer, b[:maxRaw])
		w.Write(buffer)
		b = b[maxRaw:]
	}
	// Handle the last chunk of bytes.
	if len(b) > 0 {
		out := buffer[:base64.StdEncoding.EncodedLen(len(b))]
		base64.StdEncoding.Encode(out, b)
		out = append(out, "\r\n"...)
		w.Write(out)
	}
}

//func processCalls(localLink *apiLib.XmlmcInstStruct) (){
//func processCalls(threadId int, arrayPB []*pb.ProgressBar) (){
func processCalls(threadId int) {

	localAPIKey := globalAPIKeys[threadId]
	localLink := NewEspXmlmcSession(localAPIKey)
	//localBar := arrayPB[threadId]

	localBar := globalArrayBars[threadId+1]
	//	re_boundary := regexp.MustCompile(`[cC]ontent-[tT]ype:\s{0,}multipart.*?\s*boundary=\"(.*?)\"`)
	//	re_boundary := regexp.MustCompile(`[cC]ontent-[tT]ype:\s{0,}multipart[.\s]*boundary=\"(.*?)\"`)
	//localBar.Prefix("Thread " + strconv.Itoa(threadId) + ":")
	//defer localBar.FinishPrint(" Completed")
	for {
		boolIDExists, requestID := pickOffRequestArray()

		if !boolIDExists {
			logger(3, "Finished Thread "+strconv.Itoa(threadId+1), false)
			break
		} else {
			logger(3, "Processing Email: "+requestID, false)

			localLink.SetParam("messageId", requestID)
			localLink.SetParam("excludeFileAttachments", "false")

			XMLAttachmentSearch, xmlmcErr := localLink.Invoke("mail", "getMessage")
			if xmlmcErr != nil {
				logger(4, "Unable to find Email: "+requestID+" - "+fmt.Sprintf("%v", xmlmcErr), false)
				continue
			}

			var xmlQuestionRespon structEmailResults //structAttachmentsResults
			//fmt.Println(XMLAttachmentSearch)
			qerr := xml.Unmarshal([]byte(XMLAttachmentSearch), &xmlQuestionRespon)

			if qerr != nil {
				fmt.Println("No Email Found for " + requestID)
				fmt.Println(qerr)
			} else {
				intCountDownloads := len(xmlQuestionRespon.Params.FileAttachment)
				/*				if intCountDownloads == 0 {
									logger(3, "No downloads found for: "+requestID, false)
									continue
									//return
								}
				*/logger(3, strconv.Itoa(intCountDownloads)+" downloads found for: "+requestID, false)

				localBar.Finish()
				localBar.Reset(intCountDownloads)

				var downloadedFiles []string

				newEmlFile, err := os.Create(globalAttachmentLocation + string(os.PathSeparator) + requestID + "_" + globalTimeNow + ".eml")
				if err != nil {
					logger(4, "Unable to open .eml file for: "+requestID+" - "+fmt.Sprintf("%v", err), false)
					continue
				}
				strBoundary := ""
				strBoundary = requestID + "-EmailAttachmentArchiver"
				newEmlFile.WriteString("Received: " + xmlQuestionRespon.Params.Received + "\n")
				newEmlFile.WriteString("Date: " + xmlQuestionRespon.Params.Sent + "\n")
				newEmlFile.WriteString("MIME-Version: 1.0\n")
				newEmlFile.WriteString("Content-Type: multipart/related;\n\tboundary=\"" + strBoundary + "\"\n")
				newEmlFile.WriteString("Subject: " + xmlQuestionRespon.Params.Subject + "\n")
				newEmlFile.WriteString("From: " + xmlQuestionRespon.Params.Sender + "\n")
				newEmlFile.WriteString("To: " + xmlQuestionRespon.Params.Recipient + "\n")
				newEmlFile.WriteString("\r\n\r\n")
				newEmlFile.WriteString("Please check the attachments for the original email header, body and attachments")
				newEmlFile.WriteString("\r\n\r\n")
				newEmlFile.WriteString("--" + strBoundary + "\n")

				/*
					newEmlFile.WriteString(xmlQuestionRespon.Params.RFCHeader)
					if (intCountDownloads > 0){
						//need to check for boundary to use, otherwise create own boundary
						//r := re_boundary.FindString(xmlQuestionRespon.Params.RFCHeader)
						r := re_boundary.FindAllStringSubmatch(xmlQuestionRespon.Params.RFCHeader, 1)
						if len(r) > 0 {
							if len(r[0]) > 1 {
								strBoundary = r[0][1]
							}
						}
						if strBoundary == "" {
							strBoundary = requestID + "-HornbillEmailSweeper"
							newEmlFile.WriteString("\nContent-Type: multipart/related; boundary=\"" + strBoundary + "\"")
						}
					}
				*/newEmlFile.WriteString("Content-Type: text/plain; name=\"RFCHeader.txt\"\n")
				newEmlFile.WriteString("Content-Transfer-Encoding: 8bit;\n")
				newEmlFile.WriteString("Content-Disposition: attachment")
				newEmlFile.WriteString("\r\n\r\n")
				newEmlFile.WriteString(xmlQuestionRespon.Params.RFCHeader)
				newEmlFile.WriteString("\r\n\r\n")
				newEmlFile.WriteString("--" + strBoundary + "\n")
				newEmlFile.WriteString("Content-Type: text/plain; name=\"TEXTBody.txt\"\n") //just in case
				newEmlFile.WriteString("Content-Transfer-Encoding: utf8;\n")
				newEmlFile.WriteString("Content-Disposition: attachment")
				newEmlFile.WriteString("\r\n\r\n")
				newEmlFile.WriteString(xmlQuestionRespon.Params.Body)
				newEmlFile.WriteString("\r\n\r\n")
				if xmlQuestionRespon.Params.HTMLBody != "" {
					newEmlFile.WriteString("--" + strBoundary + "\n")
					newEmlFile.WriteString("Content-Type: text/html; name=\"HTMLBody.html\"\n") //just in case
					newEmlFile.WriteString("Content-Transfer-Encoding: utf8;\n")
					newEmlFile.WriteString("Content-Disposition: attachment")
					newEmlFile.WriteString("\r\n\r\n")
					newEmlFile.WriteString(xmlQuestionRespon.Params.HTMLBody)
					newEmlFile.WriteString("\r\n\r\n")
				}

				for i := 0; i < intCountDownloads; i++ {

					//20200910 strContentLocation := xmlQuestionRespon.Params.RowData.Row[i].HContentLocation
					strFileName := xmlQuestionRespon.Params.FileAttachment[i].FileName
					strMIME := xmlQuestionRespon.Params.FileAttachment[i].MIMEType
					strAccessToken := xmlQuestionRespon.Params.FileAttachment[i].CAFSToken
					//fmt.Println(strContentLocation)
					var emptyCatch []byte

					time.Sleep(time.Millisecond * time.Duration(rand.Intn(2000))) //think this might be necessary

					strDAVurl := localLink.DavEndpoint
					strDAVurl = strDAVurl + "secure-content/download/" + strAccessToken
					logger(1, "GETting: "+strFileName, false)

					putbody := bytes.NewReader(emptyCatch)
					req, Perr := http.NewRequest("GET", strDAVurl, putbody)
					if Perr != nil {
						logger(3, "GET set-up issue", false)
						continue
					}
					req.Header.Add("Authorization", "ESP-APIKEY "+localAPIKey) //APIKey)
					req.Header.Set("User-Agent", "Go-http-client/1.1")
					response, Perr := client.Do(req)
					if Perr != nil {
						logger(3, "GET connection issue: "+fmt.Sprintf("%v", http.StatusInternalServerError), false)
						continue
					}

					//defer response.Body.Close()
					//_, _ = io.Copy(ioutil.Discard, response.Body)
					if response.StatusCode == 200 {
						newEmlFile.WriteString("--" + strBoundary + "\n")
						if strMIME == "" {
							newEmlFile.WriteString("Content-Type: application/octet-stream; name=\"" + strFileName + "\"\r\n")
						} else {
							newEmlFile.WriteString("Content-Type: " + strMIME + "; name=\"" + strFileName + "\"\r\n")
						}
						newEmlFile.WriteString("Content-Transfer-Encoding: base64\r\n")
						newEmlFile.WriteString("Content-Disposition: attachment\r\n\r\n")
						body, err := ioutil.ReadAll(response.Body)
						if err != nil {
							logger(1, "Wrote Binary instead of base64", false)
							_, _ = io.Copy(newEmlFile, response.Body)
						} else {
							base64Wrap(newEmlFile, body)
						}
						//						b64 := base64.NewEncoder(base64.StdEncoding, writeToEmail)
						//						_, _ = io.Copy(newEmlFile, response.Body)
						//_ = base64.NewEncoder(base64.StdEncoding, newEmlFile)
						//_, _ = io.Copy(newEmlFile, response.Body)

						//newEmlFile.WriteString("\r\n\r\n")
						newEmlFile.WriteString("\r\n")
						//will need to play with content-type headers, and 64bit and email width restrictions

						// yeah do NOT use sanitized filename here!
						downloadedFiles = append(downloadedFiles, xmlQuestionRespon.Params.FileAttachment[i].FileName)

					} else {
						logger(1, "Unsuccesful Download: "+fmt.Sprintf("%v", response.StatusCode), false)
					}

					err = response.Body.Close()
					if err != nil {
						logger(1, "Body Close Error: "+fmt.Sprintf("%v", err), false)
					}
					localBar.Increment()

				}
				if intCountDownloads > 0 {
					newEmlFile.WriteString("--" + strBoundary + "--")
				}
				err = newEmlFile.Close()
				if err != nil {
					logger(1, "emlFile Close Error: "+fmt.Sprintf("%v", err), false)
					downloadedFiles = nil // better ensure we are not removing anything
				}

				iDownloadedFiles := len(downloadedFiles)

				logger(1, "Succesful Downloads: "+fmt.Sprintf("%d", iDownloadedFiles), false)

				if !(configDryRun) {
					logger(3, "Removal of "+requestID, false)
					//we've got the file, so now let's remove from source:
					localLink.SetParam("mailbox", "helpdesk")
					localLink.SetParam("messageId", requestID)
					localLink.SetParam("purge", "true")
					_, xmlmcErr := localLink.Invoke("mail", "deleteMessage")
					if xmlmcErr != nil {
						logger(4, "Unable to remove Email: "+requestID, false)
						//need to decide what to do if unable to remove attachment - it might be because it didn't exist in the first place
					} else {
						logger(1, "Deleted: "+requestID, false)
					}
				} else {
					logger(3, fmt.Sprintf("Skipping removal of %s", requestID), false)
				}

				addToProcessedArray(requestID)

			}

		}
	}

	localBar.Finish()

}

func main() {
	startTime = time.Now()
	//-- Start Time for Log File
	globalTimeNow = time.Now().Format(time.RFC3339)
	globalTimeNow = strings.Replace(globalTimeNow, ":", "-", -1)
	localLogFileName += globalTimeNow
	localLogFileName += ".log"

	parseFlags()

	//-- Output to CLI and Log
	logger(1, "---- Hornbill Email Download and Removal Utility v"+fmt.Sprintf("%v", version)+" ----", false)
	logger(1, "Flag - Config File "+configFileName, false)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), false)

	//-- Load Configuration File Into Struct
	boolConfLoaded := false
	importConf, boolConfLoaded = loadConfig()
	if !boolConfLoaded {
		logger(4, "Unable to load config, process closing.", true)
		return
	}
	if !(configOverride) && configCutOff < globalUltimateCutOff {
		logger(4, "The cut off date is too short (must be >= 12 (weeks)), process closing.", true)
		return
	}
	if !(checkAPIKeys()) {
		logger(4, "No valid API keys.", true)
		return
	}

	globalMaxRoutines = len(globalAPIKeys)
	if globalMaxRoutines < 1 || globalMaxRoutines > 10 {
		logger(5, "The maximum allowed workers is between 1 and 10 (inclusive).", true)
		logger(4, "You have included "+strconv.Itoa(globalMaxRoutines)+" API keys. Please try again, with a valid number of keys.", true)
		return
	}

	//Determine Cut off date.
	t := time.Now()
	after := t.AddDate(0, 0, -1*7*configCutOff)
	globalCutOffDate = after.Format("2006-01-02")
	logger(3, "Cut Off Date: "+globalCutOffDate, true)

	setOutputFolder()

	getAllFolders()
	logger(3, "Folders Found: "+fmt.Sprintln(globalMailFolders), true)

	/*
		if len(globalMailFolders) > 0 {

			globalBarRequests = pb.New(len(globalMailFolders)).Prefix("Overall :")

			globalArrayBars = append(globalArrayBars, globalBarRequests)

			//pool := pb.NewPool(globalBarRequests)
			//var pool Pool

			amount_per_bar := len(globalMailFolders) / globalMaxRoutines
			if amount_per_bar > 0 && globalMaxRoutines > 1 {
				logger(1, "Spawning multiple processes", false)

				var wg sync.WaitGroup
				wg.Add(globalMaxRoutines)

				for i := 0; i < globalMaxRoutines; i++ {
					ppp := pb.New(amount_per_bar).Prefix("Thread " + strconv.Itoa(i+1) + ":")
					ppp.ShowTimeLeft = false
					ppp.ShowCounters = false
					ppp.ShowFinalTime = false
					globalArrayBars = append(globalArrayBars, ppp)
				}
				pool, err := pb.StartPool(globalArrayBars...)
				if err != nil {
					panic(err)
				}

				for i := 0; i < globalMaxRoutines; i++ {
					go func(i int) {
						defer wg.Done()
						populateRequestsArray(i)
					}(i)
				}
				wg.Wait()

				pool.Stop()

		} else {

			logger(1, "Just a single process", false)
			//presumably == 0 or just a single thread, so just need a single total bar.
			ppp := pb.New(1).Prefix("Only folder :")
			globalArrayBars = append(globalArrayBars, ppp)
			pool, err := pb.StartPool(globalArrayBars...)
			if err != nil {
				panic(err)
			}
			populateRequestsArray(0)
			globalArrayBars[0].Finish()
			pool.Stop()

		}
	*/
	populateRequestsArray()

	if len(globalArrayRequests) > 0 {

		//globalBarRequests = pb.StartNew(len(globalArrayRequests))
		globalBarRequests = pb.New(len(globalArrayRequests)).Prefix("Overall :")

		globalArrayBars = append(globalArrayBars, globalBarRequests)

		//pool := pb.NewPool(globalBarRequests)
		//var pool Pool

		amount_per_bar := len(globalArrayRequests) / globalMaxRoutines
		if amount_per_bar > 0 && globalMaxRoutines > 1 {
			logger(1, "Spawning multiple processes", false)

			var wg sync.WaitGroup
			wg.Add(globalMaxRoutines)

			for i := 0; i < globalMaxRoutines; i++ {
				ppp := pb.New(amount_per_bar).Prefix("Thread " + strconv.Itoa(i+1) + ":")
				ppp.ShowTimeLeft = false
				ppp.ShowCounters = false
				ppp.ShowFinalTime = false
				//defer ppp.Finish()
				//pool.Add(ppp)
				globalArrayBars = append(globalArrayBars, ppp)
			}
			pool, err := pb.StartPool(globalArrayBars...)
			//err := pool.Start()
			if err != nil {
				panic(err)
			}

			for i := 0; i < globalMaxRoutines; i++ {
				go func(i int) {
					defer wg.Done()
					processCalls(i)
				}(i)
			}
			wg.Wait()

			//globalBarRequests.FinishPrint("Utility Completed")
			//globalBarRequests.Finish()
			//globalArrayBars[0].Finish()
			pool.Stop()

		} else {
			logger(1, "Just a single process", false)
			//presumably == 0 or just a single thread, so just need a single total bar.
			ppp := pb.New(1).Prefix("Main Thread :")
			//			pool.Add(ppp)
			globalArrayBars = append(globalArrayBars, ppp)
			pool, err := pb.StartPool(globalArrayBars...)

			//			err := pool.Start()
			if err != nil {
				panic(err)
			}
			processCalls(0)
			globalArrayBars[0].Finish()
			//globalBarRequests.Finish()
			pool.Stop()

		}
	} else {
		fmt.Println("No downloads found")
	}

	//-- End output
	//logger(3, "Requests Logged: "+fmt.Sprintf("%d", counters.created), true)
	//-- Show Time Takens
	endTime = time.Since(startTime)
	logger(3, "Time Taken: "+fmt.Sprintf("%v", endTime), true)
	logger(1, "---- Hornbill Request Attachment Download and Removal Complete ---- ", false)

}

/* test of progress bars
func mainplaycheck() {
	barMax := 60
	// create bars
	//mainBar := pb.StartNew(3 * barMax).Prefix("Main ")
	mainBar := pb.New(3 * barMax).Prefix("Main ")
	first := pb.New(barMax).Prefix("First ")
	second := pb.New(barMax).Prefix("Second ")
	third := pb.New(barMax).Prefix("Third ")
	// start pool
	pool, err := pb.StartPool(mainBar, first, second, third)
	if err != nil {
		panic(err)
	}
	// update bars
	wg := new(sync.WaitGroup)
	for _, bar := range []*pb.ProgressBar{first, second, third} {
		wg.Add(1)
		go func(cb *pb.ProgressBar) {
			//cb.Total = barMax
			for n := 0; n < barMax; n++ {
				cb.Increment()
				mainBar.Increment()
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))
				//		if (rand.Intn(100) < 50) {
				//			cb.Total = 400
				//			if (rand.Intn(100) < 10) {
				//				cb.Set(0)
				//			}
				//		}
			}
			cb.Finish()
			wg.Done()
		}(bar)
	}
	wg.Wait()

	mainBar.Finish()
	// close pool
	pool.Stop()
}
*/
//NOTES

//[]byte(XMLSiteSearch)
//	fmt.Println(XMLSiteSearch)

/*

	<methodCall service="data" method="queryExec">
	<params>
	<application>com.hornbill.servicemanager</application>
	<queryName>getRequestAttachments</queryName>
	<queryParams>
	<requestId>IN00000014</requestId>
	</queryParams>
	<queryOptions>
	<resultType>allData</resultType>
	</queryOptions>
	</params>
	</methodCall>

			<rowData>
				<row>
					<h_pk_id>3</h_pk_id>
					<h_request_id>IN00000014</h_request_id>
					<h_contentlocation>/cafs_raw/fs_entity/9e7ccd808d13ce4c2825f3cfcb38c444ed7118ed.data</h_contentlocation>
					<h_filename>8_minute_world_map_gray.pdf</h_filename>
					<h_size>6397065</h_size>
					<h_timestamp>2019-05-15 11:21:57Z</h_timestamp>
					<h_visibility>trustedGuest</h_visibility>
				</row>


	<methodCall service="data" method="entityAttachRemove">
		<params>
	       <application>com.hornbill.servicemanager</application>
			<entity>Requests</entity>
			<keyValue>IN00000014</keyValue>
			<filePath>8_minute_world_map_gray_3.pdf</filePath>
		</params>
	</methodCall>

		<methodCall service="data" method="entityAttachCleanup">
	</methodCall>
*/

/*
		// get binary to upload via WEBDAV and then set value to relative "session" URI
		client := http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
			Timeout: time.Duration(10 * time.Second),
		}

		rel_link := "session/" + UserID
		strDAVurl := ldapImportConf.DAVURL + rel_link

		var imageB []byte
		var Berr error

			resp, err := http.Get(strFileName)
			if err != nil {
				logger(4, "Unable to find "+value+" ["+fmt.Sprintf("%v", http.StatusInternalServerError)+"]", false)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 201 || resp.StatusCode == 200 {
				imageB, _ = ioutil.ReadAll(resp.Body)

			} else {
				logger(4, "Unsuccesful download: "+fmt.Sprintf("%v", resp.StatusCode), false)
				return
			}

		}
		//WebDAV upload
		if len(imageB) > 0 {
			putbody := bytes.NewReader(imageB)
			req, Perr := http.NewRequest("PUT", strDAVurl, putbody)
			req.Header.Set("Content-Type", strContentType)
			req.Header.Add("Authorization", "ESP-APIKEY "+APIKey)
			req.Header.Set("User-Agent", "Go-http-client/1.1")
			response, Perr := client.Do(req)
			if Perr != nil {
				logger(4, "PUT connection issue: "+fmt.Sprintf("%v", http.StatusInternalServerError), false)
				return
			}
			defer response.Body.Close()
			_, _ = io.Copy(ioutil.Discard, response.Body)
			if response.StatusCode == 201 || response.StatusCode == 200 {
				fmt.Println("Uploaded")
				value = "/" + rel_link
			} else {
				fmt.Println("Unsuccesful Upload: "+fmt.Sprintf("%v", response.StatusCode))
				return
			}
		} else {
			fmt.Println("No Image to upload")
			return
		}
	}
*/

/* Re-Attach Files
espXmlmc := apiLib.NewXmlmcInstance(strURL)
espXmlmc.SetAPIKey(APIKey)

	espXmlmc.SetParam("application", "com.hornbill.servicemanager")
	espXmlmc.SetParam("entity", "Requests")
	espXmlmc.SetParam("keyValue", "IN00000012")

	//espXmlmc.SetParam("folder", "")
	espXmlmc.OpenElement("localFile")
		espXmlmc.SetParam("fileName", strFileName)
		espXmlmc.SetParam("fileData", "dGhpcyBpcyBmdW4=")
	espXmlmc.CloseElement("localFile")
	//espXmlmc.SetParam("serverFile", "")
	// espXmlmc.SetParam("overwrite", "")


XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityAttachFile")
if xmlmcErr != nil {
	log.Fatal(xmlmcErr)
	fmt.Println("Unable to associate Image to User Profile: "+fmt.Sprintf("%v", xmlmcErr))
}
//[]byte(XMLSiteSearch)
fmt.Println(XMLSiteSearch)
*/
