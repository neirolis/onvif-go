package networking

import (
	"context"
	"encoding/xml"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/beevik/etree"

	"github.com/neirolis/onvif-go/gosoap"
)

type device interface {
	GetEndpoint(endpoint string) (string, error)
	DeltaTime() time.Duration
}

type Request struct {
	ctx        context.Context
	device     device
	method     interface{}
	httpClient *http.Client
	username   string
	password   string
	endpoint   string
}

func NewRequest(device device, method interface{}) *Request {
	return &Request{
		device: device,
		method: method,
	}
}

func (r *Request) WithContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

func (r *Request) WithHttpClient(httpClient *http.Client) *Request {
	r.httpClient = httpClient
	return r
}

func (r *Request) WithUsernamePassword(username, password string) *Request {
	r.username = username
	r.password = password
	return r
}

func (r *Request) WithEndpoint(endpoint string) *Request {
	r.endpoint = endpoint
	return r
}

func (r *Request) Do() *Response {
	resp := &Response{}

	endpoint, err := r.getEndpoint(r.method)
	if err != nil {
		resp.error = err
		return resp
	}

	soap, err := r.buildSOAP(r.method, endpoint)
	if err != nil {
		resp.error = err
		return resp
	}

	if r.httpClient == nil {
		r.httpClient = new(http.Client)
	}

	var response *http.Response
	if r.ctx != nil {
		response, resp.error = SendSoapWithCtx(r.ctx, r.httpClient, endpoint, soap.String())
	} else {
		response, resp.error = SendSoap(r.httpClient, endpoint, soap.String())
	}

	resp.SetResponse(response)
	return resp
}

func (r *Request) getEndpoint(request interface{}) (string, error) {
	if len(r.endpoint) > 0 {
		return r.endpoint, nil
	}

	pkgPath := strings.Split(reflect.TypeOf(request).PkgPath(), "/")
	pkg := strings.ToLower(pkgPath[len(pkgPath)-1])
	endpoint, err := r.device.GetEndpoint(pkg)
	if err != nil {
		return "", err
	}
	return endpoint, nil
}

func (r *Request) buildSOAP(method interface{}, endpoint string) (gosoap.SoapMessage, error) {
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

	if r.username != "" && r.password != "" {
		if err := soap.AddWSSecurity(r.username, r.password, r.device.DeltaTime()); err != nil {
			return "", err
		}
	}

	if err := soap.AddTo(endpoint); err != nil {
		return "", err
	}

	return soap, nil
}
