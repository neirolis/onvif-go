package networking

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/kikimor/onvif/gosoap"
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

	if r.body == nil {
		var err error
		r.body, err = ioutil.ReadAll(r.response.Body)
		if err != nil {
			return nil, err
		}
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
