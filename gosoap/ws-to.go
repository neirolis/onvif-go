package gosoap

import (
	"encoding/xml"
)

type To struct {
	XMLName   xml.Name `xml:"wsa:To"`
	Operation string   `xml:",chardata"`
}
