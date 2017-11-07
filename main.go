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
	"encoding/json"
)

var pattern = flag.String("pattern", "", "Pattern to find of the servers")
var typefile = flag.String("type", "*", "type of the media to return values are video, image, audio")
var auto = flag.Bool("auto", true, "do the search on all media server available")
var list = flag.String("list", "", "get content from server uri")
var device = flag.String("device", "", "short name device stored into configuration to send the results")
var configureclient = flag.Bool("configurerenderer", false, "configure the client renderer")
var rendererConfigName = "renderers.json"
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
	XMLName        xml.Name                            `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Fault          *SoapFault
	Search         *UpnpContentDirectorySearchRequest  `xml:"urn:schemas-upnp-org:service:ContentDirectory:1 Search"`
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
		ContainerID:    ContainerID,
		SearchCriteria: "<SearchCriteria>" + SearchCriteria + "</SearchCriteria>",
		Filter:         Filter,
		StartingIndex:  StartingIndex,
		RequestedCount: RequestedCount,
		SortCriteria:   SortCriteria,
	}
	env := &SoapEnvelope{Body: &SoapBody{Search: search, Fault: nil}}
	w := &bytes.Buffer{}
	err = xml.NewEncoder(w).Encode(env)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", 0, 0, 0, err
	}
	fmt.Fprintf(os.Stderr, "Envelope SOAP:%s", string(w.String()))
	httpClient := &http.Client{}
	httpRequest, err := http.NewRequest("POST", c.Url.String(), bytes.NewBuffer(w.Bytes()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		return "", 0, 0, 0, err
	}
	httpRequest.Header.Set("SOAPACTION", `"`+URN_ContentDirectory_1+`#Search"`)
	httpRequest.Header.Set("CONTENT-TYPE", "text/xml; charset=\"utf-8\"")
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		return "", 0, 0, 0, err
	}
	if httpResponse.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "%v", httpResponse)
		return "", 0, 0, 0, errors.New("http code " + httpResponse.Status)
	}
	defer httpResponse.Body.Close()

	response := &SoapEnvelope{}
	err = xml.NewDecoder(httpResponse.Body).Decode(response)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", 0, 0, 0, err
	}
	sr := response.Body.SearchResponse
	return sr.Result, sr.NumberReturned, sr.TotalMatches, sr.UpdateID, nil
}

type UpnpContentDirectorySearchRequest struct {
	ContainerID    string `xml:"ContainerID"`
	SearchCriteria string `xml:",innerxml"`
	Filter         string `xml:"Filter"`
	StartingIndex  string `xml:"StartingIndex"`
	RequestedCount string `xml:"RequestedCount"`
	SortCriteria   string `xml:"SortCriteria"`
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

type Renderer struct {
	Name     string `json:"name"`
	Location string `json:"location"`
}

type RenderConfig struct {
	Renderers map[string]Renderer `json:"renderers"`
}

func ReadConfig() (*RenderConfig, error) {
	config := &RenderConfig{Renderers: make(map[string]Renderer, 0)}
	if _, err := os.Stat(rendererConfigName); os.IsNotExist(err) {
		err = SaveConfig(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while reading configuration file with error %v\n", err)
			return config, err
		}
	}
	f, err := os.Open(rendererConfigName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while reading configuration file with error %v\n", err)
		return config, err
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while decoding configuration file with error %v\n", err)
		return config, err
	}
	return config, err
}

func SaveConfig(config *RenderConfig) error {
	f, err := os.Create(rendererConfigName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while creating configuration file with error %v\n", err)
		return err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot encode configuration file with error %v\n", err)
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	mediaType := ""

	conf, err := ReadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while reading configuration file %v\n", err)
		return
	}

	if *configureclient == true {
		controls := make(map[int]Renderer, 0)
		i := 0
		devices, err := goupnp.DiscoverDevices("urn:schemas-upnp-org:device:MediaRenderer:1")
		if err != nil {
			fmt.Fprint(os.Stderr, "cannot discover renderer device with error %v\n", err)
		} else {
			for _, d := range devices {
				control, err := av1.NewAVTransport1ClientsByURL(d.Location)
				if err != nil {
					fmt.Fprintf(os.Stderr, "cannot find  media control with error %v\n", err)
				} else {
					for _, c := range control {
						fmt.Printf("Find device renderer %s\n", c.RootDevice.Device.FriendlyName)
						controls[i] = Renderer{Name: c.RootDevice.Device.FriendlyName, Location: c.Location.String()}
						i++
					}
				}
			}
		}
		if len(controls) == 0 {
			fmt.Fprint(os.Stderr,"No renderer devices found quiting.\n")
			return
		}
		var number int
		var shortName string
		for k, v := range controls {
			fmt.Printf("[%d] - %s : %s\n", k, v.Name, v.Location)
			i++
		}
		fmt.Printf("please enter the number of the device to set ? ")
		fmt.Scanf("%d", &number)
		fmt.Printf("\nplease enter the short name associated to this device (tv,box for instance) ? ")
		fmt.Scanf("%s", &shortName)
		fmt.Printf("\nyou set %s (%s - %s) as new device renderer\n", shortName, controls[number].Name, controls[number].Location)
		conf.Renderers[controls[number].Name] = controls[number]
		err = SaveConfig(conf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while saving configuration file %v\n", err)
		}
		return
	}

	if *typefile != "" {
		switch (*typefile) {
		case "video":
			mediaType = ".videoitem"
		case "audio":
			mediaType = ".audioitem"
		case "image":
			mediaType = ".imageitem"
		}
	}

	if *auto == true {
		files := make([]string,0)
		var renderer Renderer
		if len(conf.Renderers) != 1 && *device == "" {
			fmt.Fprintf(os.Stderr,"cannot the device to display result, you've got %d devices configured and you did not set the device short name\n",len(conf.Renderers))
			return
		}
		if *device != "" {
			renderer = conf.Renderers[*device]
		} else {
			for _,v := range conf.Renderers {
				renderer = v
				break
			}
		}
		fmt.Printf("renderer device selected %s\n",renderer.Name)
		devices, err := goupnp.DiscoverDevices("urn:schemas-upnp-org:device:MediaServer:1")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error %v", err.Error())
		} else {
			for _, d := range devices {
				clients, err := av1.NewContentDirectory1ClientsByURL(d.Location)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error while getting content directory client %v\n", err)
				} else {

					client := NewUpnpContentDirectoryClient(clients[0].Location)
					result, returnNumber, totalMatches, update, err := client.Search("*", "dc:title contains \"" + *pattern+"\" and upnp:class derivedfrom \"object.item"+mediaType+"\"", "*", "0", "0", "")
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error while getting content directory client %v from location :%v\n", err, d.Location)
					} else {
						fmt.Printf("result  %d, %d, %d for server %s\n", returnNumber, totalMatches, update, client.Url.String())
						r := &DIDLLite{}
						err := xml.Unmarshal([]byte(result), r)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error while parsing result xml %v\n", err)
						} else {
							for _, item := range r.Objects {
								value := "nothing to display"
								if len(item.Results) > 0 {
									value = item.Results[0].Value
									files = append(files,value)
								}
								fmt.Printf("%s, %s, %s\n", item.Title, item.Class, value)
							}
						}
					}
				}
			}
			// send files to display
			url, err := url.Parse(renderer.Location)
			if err != nil {
				fmt.Fprintf(os.Stderr,"cannot read url %s with error %v\n",renderer.Location,err)
				return
			}
			clients,err := av1.NewAVTransport1ClientsByURL(url)
			if err != nil {
				fmt.Fprint(os.Stderr,"cannot discover renderer device with error %v\n",err)
				return
			}
			i:=0
			for _,f := range files {
				if i == 0 {
					if err := clients[0].SetAVTransportURI(0, f, ""); err != nil {
						fmt.Fprintf(os.Stderr, "error while sending media %s to %s  with error %v", f, renderer.Name, err)
					}
					i++
				} else {
					if err := clients[0].SetNextAVTransportURI(0, f, ""); err != nil {
						fmt.Fprintf(os.Stderr, "error while sending media %s to %s  with error %v", f, renderer.Name, err)
					}
				}
			}
			clients[0].Play(0,"1")
		}
	} else {
		flag.Usage()
	}
}
