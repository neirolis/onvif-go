package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	goonvif "github.com/kikimor/onvif"
	"github.com/kikimor/onvif/device"
	"github.com/kikimor/onvif/xsd/onvif"
)

const (
	login    = "admin"
	password = "Supervisor"
)

func main() {
	//Getting an camera instance
	dev := goonvif.NewDevice(goonvif.DeviceParams{
		Xaddr:      "192.168.13.14:80",
		Username:   login,
		Password:   password,
		HttpClient: &http.Client{Timeout: 5 * time.Second},
	})
	_, err := dev.Inspect()
	if err != nil {
		panic(err)
	}

	//Preparing commands
	systemDateAndTyme := device.GetSystemDateAndTime{}
	getCapabilities := device.GetCapabilities{Category: "All"}
	createUser := device.CreateUsers{User: onvif.User{
		Username:  "TestUser",
		Password:  "TestPassword",
		UserLevel: "User",
	},
	}

	//Commands execution
	if data, err := dev.CreateRequest(systemDateAndTyme).Do().Body(); err != nil {
		log.Println(err)
	} else {
		fmt.Println(string(data))
	}

	if data, err := dev.CreateRequest(getCapabilities).Do().Body(); err != nil {
		log.Println(err)
	} else {
		fmt.Println(string(data))
	}

	if data, err := dev.CreateRequest(createUser).Do().Body(); err != nil {
		log.Println(err)
	} else {
		fmt.Println(string(data))
	}
}
