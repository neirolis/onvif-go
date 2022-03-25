package wsdiscovery

/*******************************************************
 * Copyright (C) 2018 Palanjyan Zhorzhik
 *
 * This file is part of ws-discovery project.
 *
 * ws-discovery can be copied and/or distributed without the express
 * permission of Palanjyan Zhorzhik
 *******************************************************/

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/net/ipv4"
)

const bufSize = 8192

//SendProbe to device
func SendProbe(interfaceName string, scopes, types []string, namespaces map[string]string) []string {
	// Creating UUID Version 4
	uuidV4 := uuid.Must(uuid.NewV4())
	//fmt.Printf("UUIDv4: %s\n", uuidV4)

	probeSOAP := buildProbeMessage(uuidV4.String(), scopes, types, namespaces)
	//probeSOAP = `<?xml version="1.0" encoding="UTF-8"?>
	//<Envelope xmlns="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">
	//<Header>
	//<a:Action mustUnderstand="1">http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</a:Action>
	//<a:MessageID>uuid:78a2ed98-bc1f-4b08-9668-094fcba81e35</a:MessageID><a:ReplyTo>
	//<a:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:Address>
	//</a:ReplyTo><a:To mustUnderstand="1">urn:schemas-xmlsoap-org:ws:2005:04:discovery</a:To>
	//</Header>
	//<Body><Probe xmlns="http://schemas.xmlsoap.org/ws/2005/04/discovery">
	//<d:Types xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery" xmlns:dp0="http://www.onvif.org/ver10/network/wsdl">dp0:NetworkVideoTransmitter</d:Types>
	//</Probe>
	//</Body>
	//</Envelope>`

	return sendUDPMulticast(probeSOAP.String(), interfaceName, 3702, 1024)
}

//SendProbeHikvision lookup hikvision devices
func SendProbeHikvision(interfaceName string) []string {
	// Creating UUID Version 4
	uuidV4 := uuid.Must(uuid.NewV4())
	// fmt.Printf("UUIDv4: %s\n", uuidV4)

	probeSOAP := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><Probe><Uuid>%s</Uuid><Types>inquiry</Types></Probe>`, uuidV4.String())
	return sendUDPMulticast(probeSOAP, interfaceName, 37020, 37020)
}

func sendUDPMulticast(msg string, interfaceName string, dstPort int, receivePort int) []string {
	c, err := net.ListenPacket("udp4", fmt.Sprintf("0.0.0.0:%d", receivePort))
	if err != nil {
		fmt.Printf("%+v (%d)\n", err, receivePort)
		return nil
	}
	defer c.Close()

	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		fmt.Printf("%+v (%s)\n", err, interfaceName)
	}

	p := ipv4.NewPacketConn(c)
	group := net.IPv4(239, 255, 255, 250)
	if err := p.JoinGroup(iface, &net.UDPAddr{IP: group}); err != nil {
		fmt.Println(err)
	}

	dst := &net.UDPAddr{IP: group, Port: dstPort}
	data := []byte(msg)
	for _, ifi := range []*net.Interface{iface} {
		if err := p.SetMulticastInterface(ifi); err != nil {
			fmt.Println(err)
		}
		p.SetMulticastTTL(2)
		if _, err := p.WriteTo(data, nil, dst); err != nil {
			fmt.Println(err)
		}
	}

	if err := p.SetReadDeadline(time.Now().Add(time.Second * 1)); err != nil {
		log.Fatal(err)
	}

	var result []string
	for {
		b := make([]byte, bufSize)
		n, _, _, err := p.ReadFrom(b)
		if err != nil {
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				fmt.Println(err)
			}
			break
		}
		result = append(result, string(b[0:n]))
	}
	return result
}
