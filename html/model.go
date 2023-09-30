package html

import "encoding/xml"

type Entity struct {
	XMLName xml.Name `xml:"div"`
	Style   string   `xml:"style,attr"`
	Title   Title    `xml:"div"`
	Table   Table    `xml:"table"`
}

type Title struct {
	XMLName xml.Name `xml:"div"`
	Style   string   `xml:"style,attr"`
	Body    string   `xml:",innerxml"`
}

type Table struct {
	XMLName xml.Name   `xml:"table"`
	Style   string     `xml:"style,attr"`
	Row     []TableRow `xml:"tr"`
}

type TableRow struct {
	XMLName xml.Name    `xml:"tr"`
	Data    []TableData `xml:"td"`
}

type TableData struct {
	XMLName xml.Name `xml:"td"`
	Style   string   `xml:"style,attr"`
	Data    string   `xml:",innerxml"`
}
