package main

import (
	"github.com/hornbill/goApiLib"
)

// NewEspXmlmcSession - New Xmlmc Session variable (Cloned Session)
func NewEspXmlmcSession(apiKey string) *apiLib.XmlmcInstStruct {
	espXmlmcLocal := apiLib.NewXmlmcInstance(importConf.InstanceID)
	espXmlmcLocal.SetAPIKey(apiKey)
	return espXmlmcLocal
}
