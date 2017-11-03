package main

import (
	"flag"
	"log"
	"github.com/huin/goupnp"
	"github.com/huin/goupnp/dcps/av1"
	"encoding/xml"
)

var servers = flag.Bool("servers", true, "list all the dlna servers availabled.")
var list = flag.String("list", "", "get content from server uri")

type DIDLLite struct {
	XMLName xml.Name
	DC      string   `xml:"xmlns:dc,attr"`
	UPNP    string   `xml:"xmlns:upnp,attr"`
	XSI     string   `xml:"xmlns:xsi,attr"`
	XLOC    string   `xml:"xsi:schemaLocation,attr"`
	Objects []Object `xml:"item"`
}

type Object struct {
	ID         string `xml:"id,attr"`
	Parent     string `xml:"parentID,attr"`
	Restricted string `xml:"restricted,attr"`
	Title      string `xml:"title"`
	Creator    string `xml:"creator"`
	Class      string `xml:"class"`
	Date       string `xml:"date"`
	Results    []Res  `xml:"res"`
}

type Res struct {
	Resolution      string `xml:"resolution,attr"`
	Size            uint64 `xml:"size,attr"`
	ProtocolInfo    string `xml:"protocolInfo,attr"`
	Duration        string `xml:"duration,attr"`
	Bitrate         string `xml:"bitrate,attr"`
	SampleFrequency uint64 `xml:"sampleFrequency"`
	NrAudioChannels uint64 `xml:"nrAudioChannels"`
	Value           string `xml:",chardata"`
}

func main() {
	flag.Parse()
	if *servers == true {
		devices, err := goupnp.DiscoverDevices("urn:schemas-upnp-org:device:MediaServer:1")
		if err != nil {
			log.Printf("error %v", err.Error())
		} else {
			for _, d := range devices {
				//log.Printf("Device %v,%v",d.Root,d.Location)
				clients, err := av1.NewContentDirectory1ClientsByURL(d.Location)
				if err != nil {
					log.Printf("Error while getting content directory client %v", err)
				} else {
					client := clients[0]

					result, returnedNumber, totalMatches, _, err := client.Search("*", "(dc:title contains star wars) and (upnp:class derivedfrom object.item.videoItem)", "*", 0, 0, "")
					if err != nil {
						log.Printf("Error while getting content directory client %v from location :%v", err, d.Location)
					} else {
						r := &DIDLLite{}
						err := xml.Unmarshal([]byte(result), r)
						if err != nil {
							log.Printf("Error while parsing result xml %v", err)
						} else {
							for _, item := range r.Objects {
								value := "nothing to display"
								if len(item.Results) > 0 {
									value = item.Results[0].Value
								}
								log.Printf("%s, %s, %s", item.Title, item.Class, value)
							}
							log.Printf("result %d, %d", returnedNumber, totalMatches)
						}
					}
				}
			}
		}
	} else {
		flag.Usage()
	}
}
