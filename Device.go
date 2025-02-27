package onvif

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"

	"github.com/neirolis/onvif-go/device"
	"github.com/neirolis/onvif-go/networking"
	wsdiscovery "github.com/neirolis/onvif-go/ws-discovery"
)

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
	Location        string
	MAC             string
}

func (info *DeviceInfo) Name() string {
	if len(info.Manufacturer) > 0 && !strings.Contains(strings.ToLower(info.Model), strings.ToLower(info.Manufacturer)) {
		return fmt.Sprintf("%s %s", info.Manufacturer, info.Model)
	}
	return info.Model
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

//GetXaddr return IP address.
func (dev *Device) GetXaddr() string {
	return dev.params.Xaddr
}

//DeltaTime return delta time between local time and device time (time.Now() - deviceTime).
func (dev *Device) DeltaTime() time.Duration {
	return dev.deltaTime
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
	existDevices := make(map[string]bool)

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
				dev.lookupScopes(doc)
				nvtDevices = append(nvtDevices, *dev)
				existDevices[xaddr] = true
			}
		}
	}

	for _, j := range wsdiscovery.SendProbeHikvision(interfaceName) {
		dev := NewDevice(DeviceParams{})
		doc := etree.NewDocument()
		if err := doc.ReadFromString(j); err != nil {
			fmt.Errorf("%s", err.Error())
			continue
		}
		if dev.LookupHikvisionProbeMatch(doc) && existDevices[dev.params.Xaddr] == false {
			nvtDevices = append(nvtDevices, *dev)
			existDevices[dev.params.Xaddr] = true
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
	extension_services := doc.FindElements("./Envelope/Body/GetCapabilitiesResponse/Capabilities/Extension/*/XAddr")
	for _, j := range extension_services {
		dev.addEndpoint(j.Parent().Tag, j.Text())
	}

	return nil
}

// lookup scopes by path ./Body/ProbeMatches/ProbeMatch/Scopes
// ex: <d:Scopes>onvif://www.onvif.org/type/video_encoder onvif://www.onvif.org/hardware/DS-2CD2042WD-I onvif://www.onvif.org/name/HIKVISION%20DS-2CD2042WD-I</d:Scopes>
func (dev *Device) lookupScopes(doc *etree.Document) {
	elem := doc.Root().FindElement("./Body/ProbeMatches/ProbeMatch/Scopes")
	if elem == nil {
		return
	}

	for _, scope := range strings.Split(elem.Text(), " ") {
		u, err := url.Parse(scope)
		if err != nil {
			continue
		}

		upath := strings.ToLower(u.Path)

		switch {
		case strings.Contains(upath, "hardware"):
			_, dev.info.HardwareId = path.Split(u.Path)
		case strings.Contains(upath, "name"):
			_, dev.info.Model = path.Split(u.Path)
		case strings.Contains(upath, "location"):
			_, dev.info.Location = path.Split(u.Path)
		case strings.Contains(upath, "mac"):
			_, dev.info.MAC = path.Split(u.Path)
		}
	}

	return
}

//LookupHikvisionProbeMatch get Hikvision info
func (dev *Device) LookupHikvisionProbeMatch(doc *etree.Document) bool {
	probeMatch := doc.Root()
	if probeMatch == nil {
		return false
	}

	dev.info.Manufacturer = "HIKVISION"

	for _, child := range probeMatch.ChildElements() {
		switch {
		case child.Tag == "DeviceDescription":
			dev.info.Model = fmt.Sprintf("HIKVISION %s", child.Text())
		case child.Tag == "DeviceSN":
			dev.info.SerialNumber = child.Text()
		case child.Tag == "MAC":
			dev.info.MAC = strings.ReplaceAll(child.Text(), "-", ":")
		case child.Tag == "SoftwareVersion":
			dev.info.FirmwareVersion = child.Text()

		case child.Tag == "IPv4Address":
			dev.params.Xaddr = child.Text()
		}
	}

	return len(dev.params.Xaddr) > 0
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

func (dev *Device) CreateRequest(method interface{}) *networking.Request {
	return networking.NewRequest(dev, method).
		WithHttpClient(dev.params.HttpClient).
		WithUsernamePassword(dev.params.Username, dev.params.Password)
}

func (dev *Device) Inspect() (*device.GetCapabilitiesResponse, error) {
	return dev.inspect(nil)
}

func (dev *Device) InspectWithCtx(ctx context.Context) (*device.GetCapabilitiesResponse, error) {
	return dev.inspect(ctx)
}

func (dev *Device) inspect(ctx context.Context) (*device.GetCapabilitiesResponse, error) {
	_, err := dev.updateDeltaTime(ctx)
	if err != nil {
		return nil, err
	}

	resp := dev.CreateRequest(device.GetCapabilities{Category: "All"}).WithContext(ctx).Do()
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	if !resp.StatusOK() {
		return nil, errors.New("camera is not available at " + dev.params.Xaddr + " or it does not support ONVIF services")
	}

	body, err := resp.Body()
	if err != nil {
		return nil, err
	}

	if err = dev.getSupportedServices(body); err != nil {
		return nil, err
	}

	capabilitiesResponse := device.GetCapabilitiesResponse{}
	if err = resp.Unmarshal(&capabilitiesResponse); err != nil {
		return nil, err
	}

	return &capabilitiesResponse, nil
}

func (dev *Device) UpdateDeltaTime() (time.Duration, error) {
	return dev.updateDeltaTime(nil)
}

func (dev *Device) UpdateDeltaTimeCtx(ctx context.Context) (time.Duration, error) {
	return dev.updateDeltaTime(ctx)
}

func (dev *Device) updateDeltaTime(ctx context.Context) (time.Duration, error) {
	resp := dev.CreateRequest(device.GetSystemDateAndTime{}).WithContext(ctx).Do()
	if resp.Error() != nil {
		return 0, resp.Error()
	}

	systemDateAndTime := device.GetSystemDateAndTimeResponse{}
	if err := resp.Unmarshal(&systemDateAndTime); err != nil {
		return 0, err
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

func (dev *Device) UpdateDeviceInfo(ctx context.Context) (DeviceInfo, error) {
	resp := dev.CreateRequest(device.GetDeviceInformation{}).WithContext(ctx).Do()
	if resp.Error() != nil {
		return dev.info, resp.Error()
	}

	deviceInformationResponse := device.GetDeviceInformationResponse{}
	if err := resp.Unmarshal(&deviceInformationResponse); err != nil {
		return dev.info, err
	}

	dev.info.Manufacturer = deviceInformationResponse.Manufacturer
	dev.info.Model = deviceInformationResponse.Model
	dev.info.FirmwareVersion = deviceInformationResponse.FirmwareVersion
	dev.info.SerialNumber = deviceInformationResponse.SerialNumber
	dev.info.HardwareId = deviceInformationResponse.HardwareId

	return dev.info, nil
}

// ReplaceHostToXAddr replacing host:port on string to dev.params.Xaddr.
// NAT needed.
func (dev *Device) ReplaceHostToXAddr(u string) (string, error) {
	url, err := url.Parse(u)
	if err != nil {
		return u, err
	}
	url.Host = dev.params.Xaddr
	return url.String(), nil
}

func (dev *Device) addEndpoint(Key, Value string) {
	//use lowCaseKey
	//make key having ability to handle Mixed Case for Different vendor devcie (e.g. Events EVENTS, events)
	lowCaseKey := strings.ToLower(Key)
	Value, _ = dev.ReplaceHostToXAddr(Value)
	dev.endpoints[lowCaseKey] = Value
}

//getEndpoint functions get the target service endpoint in a better way
func (dev Device) GetEndpoint(endpoint string) (string, error) {

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
