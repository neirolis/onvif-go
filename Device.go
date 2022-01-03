package onvif

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"

	"github.com/kikimor/onvif/device"
	"github.com/kikimor/onvif/gosoap"
	"github.com/kikimor/onvif/networking"
	wsdiscovery "github.com/kikimor/onvif/ws-discovery"
)

//Xlmns XML Scheam
var Xlmns = map[string]string{
	"onvif":   "http://www.onvif.org/ver10/schema",
	"tds":     "http://www.onvif.org/ver10/device/wsdl",
	"trt":     "http://www.onvif.org/ver10/media/wsdl",
	"tev":     "http://www.onvif.org/ver10/events/wsdl",
	"tptz":    "http://www.onvif.org/ver20/ptz/wsdl",
	"timg":    "http://www.onvif.org/ver20/imaging/wsdl",
	"tan":     "http://www.onvif.org/ver20/analytics/wsdl",
	"xmime":   "http://www.w3.org/2005/05/xmlmime",
	"wsnt":    "http://docs.oasis-open.org/wsn/b-2",
	"xop":     "http://www.w3.org/2004/08/xop/include",
	"wsa":     "http://www.w3.org/2005/08/addressing",
	"wstop":   "http://docs.oasis-open.org/wsn/t-1",
	"wsntw":   "http://docs.oasis-open.org/wsn/bw-2",
	"wsrf-rw": "http://docs.oasis-open.org/wsrf/rw-2",
	"wsaw":    "http://www.w3.org/2006/05/addressing/wsdl",
}

//DeviceType alias for int
type DeviceType int

// Onvif Device Tyoe
const (
	NVD DeviceType = iota
	NVS
	NVA
	NVT
)

func (devType DeviceType) String() string {
	stringRepresentation := []string{
		"NetworkVideoDisplay",
		"NetworkVideoStorage",
		"NetworkVideoAnalytics",
		"NetworkVideoTransmitter",
	}
	i := uint8(devType)
	switch {
	case i <= uint8(NVT):
		return stringRepresentation[i]
	default:
		return strconv.Itoa(int(i))
	}
}

//DeviceInfo struct contains general information about ONVIF device
type DeviceInfo struct {
	Manufacturer    string
	Model           string
	FirmwareVersion string
	SerialNumber    string
	HardwareId      string
}

//Device for a new device of onvif and DeviceInfo
//struct represents an abstract ONVIF device.
//It contains methods, which helps to communicate with ONVIF device
type Device struct {
	params    DeviceParams
	endpoints map[string]string
	info      DeviceInfo
	deltaTime time.Duration
}

type DeviceParams struct {
	Xaddr      string
	Username   string
	Password   string
	HttpClient *http.Client
}

//GetServices return available endpoints
func (dev *Device) GetServices() map[string]string {
	return dev.endpoints
}

//GetServices return available endpoints
func (dev *Device) GetDeviceInfo() DeviceInfo {
	return dev.info
}

func readResponse(resp *http.Response) string {
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return string(b)
}

//GetAvailableDevicesAtSpecificEthernetInterface ...
func GetAvailableDevicesAtSpecificEthernetInterface(interfaceName string) []Device {
	/*
		Call an ws-discovery Probe Message to Discover NVT type Devices
	*/
	devices := wsdiscovery.SendProbe(interfaceName, nil, []string{"dn:" + NVT.String()}, map[string]string{"dn": "http://www.onvif.org/ver10/network/wsdl"})
	nvtDevices := make([]Device, 0)

	for _, j := range devices {
		doc := etree.NewDocument()
		if err := doc.ReadFromString(j); err != nil {
			fmt.Errorf("%s", err.Error())
			return nil
		}

		endpoints := doc.Root().FindElements("./Body/ProbeMatches/ProbeMatch/XAddrs")
		for _, xaddr := range endpoints {
			xaddr := strings.Split(strings.Split(xaddr.Text(), " ")[0], "/")[2]
			fmt.Println(xaddr)
			c := 0

			for c = 0; c < len(nvtDevices); c++ {
				if nvtDevices[c].params.Xaddr == xaddr {
					fmt.Println(nvtDevices[c].params.Xaddr, "==", xaddr)
					break
				}
			}

			if c < len(nvtDevices) {
				continue
			}

			dev := NewDevice(DeviceParams{Xaddr: strings.Split(xaddr, " ")[0]})
			_, err := dev.Inspect()

			if err != nil {
				fmt.Println("Error", xaddr)
				fmt.Println(err)
				continue
			} else {
				nvtDevices = append(nvtDevices, *dev)
			}
		}
	}

	return nvtDevices
}

func (dev *Device) getSupportedServices(data []byte) error {
	doc := etree.NewDocument()

	if err := doc.ReadFromBytes(data); err != nil {
		return err
	}
	services := doc.FindElements("./Envelope/Body/GetCapabilitiesResponse/Capabilities/*/XAddr")
	for _, j := range services {
		dev.addEndpoint(j.Parent().Tag, j.Text())
	}

	return nil
}

//NewDevice function construct a ONVIF Device entity
func NewDevice(params DeviceParams) *Device {
	dev := Device{
		params:    params,
		endpoints: make(map[string]string),
	}

	dev.addEndpoint("Device", "http://"+dev.params.Xaddr+"/onvif/device_service")

	if dev.params.HttpClient == nil {
		dev.params.HttpClient = new(http.Client)
	}

	return &dev
}

func (dev *Device) Inspect() (*device.GetCapabilitiesResponse, error) {
	return dev.inspect(dev.CallMethod)
}

func (dev *Device) InspectWithCtx(ctx context.Context) (*device.GetCapabilitiesResponse, error) {
	return dev.inspect(func(method interface{}) (*http.Response, error) {
		return dev.CallMethodWithCtx(ctx, method)
	})
}

func (dev *Device) inspect(callMethod func(method interface{}) (*http.Response, error)) (*device.GetCapabilitiesResponse, error) {
	_, err := dev.updateDeltaTime(callMethod)
	if err != nil {
		return nil, err
	}

	getCapabilities := device.GetCapabilities{Category: "All"}
	resp, err := callMethod(getCapabilities)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("camera is not available at " + dev.params.Xaddr + " or it does not support ONVIF services")
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := dev.getSupportedServices(data); err != nil {
		return nil, err
	}

	capabilitiesResponse := device.GetCapabilitiesResponse{}
	if err := xml.Unmarshal([]byte(gosoap.SoapMessage(data).Body()), &capabilitiesResponse); err != nil {
		return nil, err
	}

	return &capabilitiesResponse, nil
}

func (dev *Device) UpdateDeltaTime() (time.Duration, error) {
	return dev.updateDeltaTime(dev.CallMethod)
}

func (dev *Device) UpdateDeltaTimeCtx(ctx context.Context) (time.Duration, error) {
	return dev.updateDeltaTime(func(method interface{}) (*http.Response, error) {
		return dev.CallMethodWithCtx(ctx, method)
	})
}

func (dev *Device) updateDeltaTime(callMethod func(method interface{}) (*http.Response, error)) (time.Duration, error) {
	systemDateAndTime := device.GetSystemDateAndTimeResponse{}
	if err := dev.request(device.GetSystemDateAndTime{}, &systemDateAndTime, callMethod); err != nil {
		return 0, nil
	}

	date := systemDateAndTime.SystemDateAndTime.UTCDateTime
	deviceTime := time.Date(
		int(date.Date.Year),
		time.Month(date.Date.Month),
		int(date.Date.Day),
		int(date.Time.Hour),
		int(date.Time.Minute),
		int(date.Time.Second),
		0,
		time.UTC,
	)
	localTime := time.Now().UTC()

	dev.deltaTime = localTime.Sub(deviceTime)
	return dev.deltaTime, nil
}

func (dev *Device) addEndpoint(Key, Value string) {
	//use lowCaseKey
	//make key having ability to handle Mixed Case for Different vendor devcie (e.g. Events EVENTS, events)
	lowCaseKey := strings.ToLower(Key)

	// Replace host with host from device params.
	if u, err := url.Parse(Value); err == nil {
		u.Host = dev.params.Xaddr
		Value = u.String()
	}

	dev.endpoints[lowCaseKey] = Value
}

//GetEndpoint returns specific ONVIF service endpoint address
func (dev *Device) GetEndpoint(name string) string {
	return dev.endpoints[name]
}

func (dev Device) buildMethodSOAP(method interface{}) (gosoap.SoapMessage, error) {
	output, err := xml.MarshalIndent(method, "  ", "    ")
	if err != nil {
		return "", err
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromString(string(output)); err != nil {
		return "", err
	}

	soap := gosoap.NewEmptySOAP()
	soap.AddRootNamespaces(Xlmns)
	if err := soap.AddBodyContent(doc.Root()); err != nil {
		return "", err
	}

	//Auth Handling
	if dev.params.Username != "" && dev.params.Password != "" && reflect.TypeOf(method) != reflect.TypeOf(device.GetSystemDateAndTime{}) {
		if err := soap.AddWSSecurity(dev.params.Username, dev.params.Password, dev.deltaTime); err != nil {
			return "", err
		}
	}

	return soap, nil
}

//getEndpoint functions get the target service endpoint in a better way
func (dev Device) getEndpoint(endpoint string) (string, error) {

	// common condition, endpointMark in map we use this.
	if endpointURL, bFound := dev.endpoints[endpoint]; bFound {
		return endpointURL, nil
	}

	//but ,if we have endpoint like event、analytic
	//and sametime the Targetkey like : events、analytics
	//we use fuzzy way to find the best match url
	var endpointURL string
	for targetKey := range dev.endpoints {
		if strings.Contains(targetKey, endpoint) {
			endpointURL = dev.endpoints[targetKey]
			return endpointURL, nil
		}
	}
	return endpointURL, errors.New("target endpoint service not found")
}

//CallMethod functions call an method, defined <method> struct.
//You should use Authenticate method to call authorized requests.
func (dev Device) CallMethod(method interface{}) (*http.Response, error) {
	return dev.callMethod(method, networking.SendSoap)
}

//CallMethodWithCtx functions call an method, defined <method> struct.
//You should use Authenticate method to call authorized requests.
func (dev Device) CallMethodWithCtx(ctx context.Context, method interface{}) (*http.Response, error) {
	return dev.callMethod(method, func(httpClient *http.Client, endpoint, message string) (*http.Response, error) {
		return networking.SendSoapWithCtx(ctx, httpClient, endpoint, message)
	})
}

//callMethod functions call an method, defined <method> struct.
//You should use Authenticate method to call authorized requests.
func (dev Device) callMethod(
	method interface{},
	sendSoap func(httpClient *http.Client, endpoint, message string) (*http.Response, error),
) (*http.Response, error) {
	soap, err := dev.buildMethodSOAP(method)
	if err != nil {
		return nil, err
	}

	pkgPath := strings.Split(reflect.TypeOf(method).PkgPath(), "/")
	pkg := strings.ToLower(pkgPath[len(pkgPath)-1])
	endpoint, err := dev.getEndpoint(pkg)
	if err != nil {
		return nil, err
	}

	return sendSoap(dev.params.HttpClient, endpoint, soap.String())
}

// Request executes a query with request struct and puts the result in response struct.
func (dev Device) Request(request interface{}, response interface{}) error {
	return dev.request(response, response, func(method interface{}) (*http.Response, error) {
		return dev.CallMethod(request)
	})
}

// RequestWithCtx executes a context query with request struct and puts the result in response struct.
func (dev Device) RequestWithCtx(ctx context.Context, request interface{}, response interface{}) error {
	return dev.request(response, response, func(method interface{}) (*http.Response, error) {
		return dev.CallMethodWithCtx(ctx, request)
	})
}

func (dev Device) request(
	request interface{},
	response interface{},
	callMethod func(method interface{}) (*http.Response, error),
) error {
	resp, err := callMethod(request)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err = xml.Unmarshal([]byte(gosoap.SoapMessage(data).Body()), response); err != nil {
		return err
	}

	return nil
}
