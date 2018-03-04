package client

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"encoding/json"

	"bytes"

	"github.com/glynternet/go-accounting-storage"
	"github.com/glynternet/go-accounting/balance"
	"github.com/pkg/errors"
)

// Client is a client to retrieve accounting items over http using REST
type Client string

func (c Client) getFromEndpoint(endpoint string) (*http.Response, error) {
	return http.Get(string(c) + endpoint)
}

func (c Client) postToEndpoint(endpoint string, contentType string, body io.Reader) (*http.Response, error) {
	return http.Post(string(c)+endpoint, contentType, body)
}

func (c Client) Available() bool {
	return errors.New("not implemented") != nil
}

func (c Client) Close() error {
	return errors.New("not implemented")
}

func (c Client) InsertBalance(a storage.Account, b balance.Balance) (*storage.Balance, error) {
	return nil, errors.New("not implemented")
}

func (c Client) getBodyFromEndpoint(e string) ([]byte, error) {
	res, err := c.getFromEndpoint(e)
	if err != nil {
		return nil, errors.Wrap(err, "getting from endpoint")
	}
	return processRequestForBody(res)
}

func processRequestForBody(res *http.Response) ([]byte, error) {
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned unexpected code %d (%s)", res.StatusCode, res.Status)
	}
	bod, err := ioutil.ReadAll(res.Body)
	defer func() {
		cErr := res.Body.Close()
		if err == nil {
			err = errors.Wrapf(cErr, "closing response body")
		}
	}()
	return bod, errors.Wrap(err, "reading response body")
}

func (c Client) postAsJSONToEndpoint(e string, thing interface{}) (*http.Response, error) {
	bs, err := json.Marshal(thing)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling json")
	}
	res, err := c.postToEndpoint(e, `application/json; charset=UTF-8`, bytes.NewReader(bs))
	return res, errors.Wrap(err, "posting to endpoint")
}
