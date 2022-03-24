package event

import (
	"github.com/kikimor/onvif/xsd"
)

//GetServiceCapabilities action
type GetServiceCapabilities struct {
	XMLName string `xml:"tev:GetServiceCapabilities"`
}

//GetServiceCapabilitiesResponse type
type GetServiceCapabilitiesResponse struct {
	Capabilities Capabilities
}

//SubscriptionPolicy action
type SubscriptionPolicy struct { //tev http://www.onvif.org/ver10/events/wsdl
	ChangedOnly xsd.Boolean `xml:"ChangedOnly,attr"`
}

//Subscribe action for subscribe event topic
type Subscribe struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	XMLName                struct{}              `xml:"wsnt:Subscribe"`
	ConsumerReference      EndpointReferenceType `xml:"wsnt:ConsumerReference"`
	Filter                 *FilterType           `xml:"wsnt:Filter"`
	SubscriptionPolicy     *SubscriptionPolicy   `xml:"wsnt:SubscriptionPolicy"`
	InitialTerminationTime TerminationTime       `xml:"wsnt:InitialTerminationTime"`
}

//SubscribeResponse message for subscribe event topic
type SubscribeResponse struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	SubscriptionReference EndpointReferenceTypeResponse `xml:"SubscriptionReference"`
	CurrentTime           CurrentTime                   `xml:"CurrentTime"`
	TerminationTime       TerminationTime               `xml:"TerminationTime"`
}

//Renew action for refresh event topic subscription
type Renew struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	XMLName         struct{}        `xml:"wsnt:Renew"`
	TerminationTime TerminationTime `xml:"wsnt:TerminationTime"`
}

//RenewResponse for Renew action
type RenewResponse struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	TerminationTime TerminationTime `xml:"TerminationTime"`
	CurrentTime     CurrentTime     `xml:"CurrentTime"`
}

//Unsubscribe action for Unsubscribe event topic
type Unsubscribe struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	XMLName struct{} `xml:"wsnt:Unsubscribe"`
	Any     string
}

//UnsubscribeResponse message for Unsubscribe event topic
type UnsubscribeResponse struct { //http://docs.oasis-open.org/wsn/b-2.xsd
	Any string
}

//CreatePullPointSubscription action
//BUG(r) Bad AbsoluteOrRelativeTimeType type
type CreatePullPointSubscription struct {
	XMLName                string              `xml:"tev:CreatePullPointSubscription"`
	Filter                 *FilterType         `xml:"tev:Filter"`
	InitialTerminationTime TerminationTime     `xml:"tev:InitialTerminationTime"`
	SubscriptionPolicy     *SubscriptionPolicy `xml:"tev:sSubscriptionPolicy"`
}

//CreatePullPointSubscriptionResponse action
type CreatePullPointSubscriptionResponse struct {
	SubscriptionReference EndpointReferenceTypeResponse
	CurrentTime           CurrentTime
	TerminationTime       TerminationTime
}

//GetEventProperties action
type GetEventProperties struct {
	XMLName string `xml:"tev:GetEventProperties"`
}

//GetEventPropertiesResponse action
type GetEventPropertiesResponse struct {
	TopicNamespaceLocation          xsd.AnyURI
	FixedTopicSet                   FixedTopicSet
	TopicSet                        TopicSet
	TopicExpressionDialect          TopicExpressionDialect
	MessageContentFilterDialect     xsd.AnyURI
	ProducerPropertiesFilterDialect xsd.AnyURI
	MessageContentSchemaLocation    xsd.AnyURI
}

//Port type PullPointSubscription

//PullMessages Action
type PullMessages struct {
	XMLName      string       `xml:"tev:PullMessages"`
	Timeout      xsd.Duration `xml:"tev:Timeout"`
	MessageLimit xsd.Int      `xml:"tev:MessageLimit"`
}

//PullMessagesResponse response type
type PullMessagesResponse struct {
	CurrentTime         CurrentTime
	TerminationTime     TerminationTime
	NotificationMessage NotificationMessage
}

//PullMessagesFaultResponse response type
type PullMessagesFaultResponse struct {
	MaxTimeout      xsd.Duration
	MaxMessageLimit xsd.Int
}

//Seek action
type Seek struct {
	XMLName string       `xml:"tev:Seek"`
	UtcTime xsd.DateTime `xml:"tev:UtcTime"`
	Reverse xsd.Boolean  `xml:"tev:Reverse"`
}

//SeekResponse action
type SeekResponse struct {
}

//SetSynchronizationPoint action
type SetSynchronizationPoint struct {
	XMLName string `xml:"tev:SetSynchronizationPoint"`
}

//SetSynchronizationPointResponse action
type SetSynchronizationPointResponse struct {
}

type Notify struct {
	XMLName             string `xml:"Notify"`
	NotificationMessage NotificationMessage
}
