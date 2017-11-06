package main

import (
	"flag"
	"encoding/xml"

	"github.com/huin/goupnp/dcps/av1"
	"net/url"
	"bytes"
	"net/http"
	"fmt"
	"github.com/huin/goupnp"
	"errors"
	"os"
)
var pattern = flag.String("pattern","","Pattern to find of the servers")
var auto = flag.Bool("auto", true, "do the search on all media server available")
var list = flag.String("list", "", "get content from server uri")
var URN_ContentDirectory_1 = "urn:schemas-upnp-org:service:ContentDirectory:1"

type SoapEnvelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    *SoapBody
}

type SoapFault struct {
	Faultstring string
	Detail      string
}

type SoapBody struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Fault          *SoapFault
	Search *UpnpContentDirectorySearchRequest `xml:"urn:schemas-upnp-org:service:ContentDirectory:1 Search"`
	SearchResponse *UpnpContentDirectorySearchResponse `xml:"urn:schemas-upnp-org:service:ContentDirectory:1 SearchResponse"`
}

type UpnpContentDirectoryClient struct {
	Url *url.URL `xml:"-"`
}

func NewUpnpContentDirectoryClient(url *url.URL) (*UpnpContentDirectoryClient) {
	return &UpnpContentDirectoryClient{Url: url}
}

func (c *UpnpContentDirectoryClient) Search(ContainerID string, SearchCriteria string, Filter string, StartingIndex string, RequestedCount string, SortCriteria string) (Result string, NumberReturned uint32, TotalMatches uint32, UpdateID uint32, err error) {
	search := &UpnpContentDirectorySearchRequest{
		ContainerID: ContainerID,
		SearchCriteria: "<SearchCriteria>"+SearchCriteria+"</SearchCriteria>",
		Filter: Filter,
		StartingIndex: StartingIndex,
		RequestedCount: RequestedCount,
		SortCriteria: SortCriteria,
	}
	env := &SoapEnvelope{Body:&SoapBody{Search:search,Fault:nil}}
	w := &bytes.Buffer{}
	err = xml.NewEncoder(w).Encode(env)
	if err != nil {
		fmt.Fprintln(os.Stderr,err)
		return "",0,0,0,err
	}
	fmt.Fprintf(os.Stderr ,"Envelope SOAP:%s",string(w.String()))
	httpClient := &http.Client{}
	httpRequest,err := http.NewRequest("POST",c.Url.String(),bytes.NewBuffer(w.Bytes()))
	if err != nil {
		fmt.Fprintf(os.Stderr ,"%v",err)
		return "",0,0,0,err
	}
	httpRequest.Header.Set("SOAPACTION",`"` + URN_ContentDirectory_1 + `#Search"`)
	httpRequest.Header.Set("CONTENT-TYPE","text/xml; charset=\"utf-8\"")
	httpResponse,err := httpClient.Do(httpRequest)
	if err != nil {
		fmt.Fprintf(os.Stderr ,"%v",err)
		return "",0,0,0,err
	}
	if httpResponse.StatusCode != 200 {
		fmt.Fprintf(os.Stderr ,"%v",httpResponse)
		return "",0,0,0,errors.New("http code "+ httpResponse.Status)
	}
	defer httpResponse.Body.Close()

	response := &SoapEnvelope{}
	err = xml.NewDecoder(httpResponse.Body).Decode(response)
	if err != nil {
		fmt.Fprintln(os.Stderr ,err)
		return "",0,0,0,err
	}
	sr := response.Body.SearchResponse
	return sr.Result,sr.NumberReturned,sr.TotalMatches,sr.UpdateID,nil
}

type UpnpContentDirectorySearchRequest struct {
	ContainerID    string `xml:"ContainerID"`
	SearchCriteria  string `xml:",innerxml"`
	Filter         string `xml:"Filter"`
	StartingIndex  string `xml:"StartingIndex"`
	RequestedCount string `xml:"RequestedCount"`
	SortCriteria  string `xml:"SortCriteria"`
}



type UpnpContentDirectorySearchResponse struct {
	Result         string
	NumberReturned uint32
	TotalMatches   uint32
	UpdateID       uint32
}

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
	if *auto == true {
		devices, err := goupnp.DiscoverDevices("urn:schemas-upnp-org:device:MediaServer:1")
		if err != nil {
			fmt.Fprintf(os.Stderr ,"error %v", err.Error())
		} else {
			for _, d := range devices {
				clients, err := av1.NewContentDirectory1ClientsByURL(d.Location)
				if err != nil {
					fmt.Fprintf(os.Stderr,"Error while getting content directory client %v", err)
				} else {

					client := NewUpnpContentDirectoryClient(clients[0].Location)
					result, returnNumber, totalMatches, update, err := client.Search("*", "dc:title contains \"" + *pattern + "\"", "*", "0", "0", "")
					if err != nil {
						fmt.Fprintf(os.Stderr,"Error while getting content directory client %v from location :%", err, d.Location)
					} else {
						fmt.Printf("result  %d, %d, %d\n", returnNumber, totalMatches, update)
						r := &DIDLLite{}
						err := xml.Unmarshal([]byte(result), r)
						if err != nil {
							fmt.Fprintf(os.Stderr ,"Error while parsing result xml %v", err)
						} else {
							for _, item := range r.Objects {
								value := "nothing to display"
								if len(item.Results) > 0 {
									value = item.Results[0].Value
								}
								fmt.Printf("%s, %s, %s\n", item.Title, item.Class, value)
							}
						}
					}
				}
			}
		}
	} else {
		flag.Usage()
	}
}
