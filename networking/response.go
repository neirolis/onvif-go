package networking

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/neirolis/onvif-go/gosoap"
)

var invalidResponse = errors.New("invalid response")

type Response struct {
	response *http.Response
	error    error
	body     []byte
}

func (r *Response) Error() error {
	return r.error
}

func (r *Response) SetResponse(response *http.Response) {
	if response == nil {
		return
	}
	r.response = response
	defer r.response.Body.Close()
	r.body, r.error = ioutil.ReadAll(r.response.Body)
}

func (r *Response) StatusOK() bool {
	if r.error != nil || r.response == nil {
		return false
	}
	return r.response.StatusCode == http.StatusOK
}

func (r *Response) Body() ([]byte, error) {
	if r.error != nil {
		return nil, r.error
	}
	if r.response == nil {
		return nil, invalidResponse
	}

	return r.body, nil
}

func (r *Response) Unmarshal(responses ...interface{}) error {
	if r.error != nil {
		return r.error
	}
	if r.response == nil {
		return invalidResponse
	}
	if !r.StatusOK() {
		return errors.New("return status code != 200")
	}

	data, err := r.Body()
	if err != nil {
		return err
	}

	body, err := gosoap.SoapMessage(data).Body()
	if err != nil {
		return err
	}

	for _, response := range responses {
		if err = xml.Unmarshal([]byte(body), response); err != nil {
			return err
		}
	}

	return nil
}
