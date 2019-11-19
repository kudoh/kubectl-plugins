package check

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type HttpChecker struct {
	Method          string
	Url             *url.URL
	SkipHttpsVerify bool
	Repeat          bool
	Body            io.Reader
	ContentType     string
}

func (c *HttpChecker) Execute() error {

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: c.SkipHttpsVerify,
			},
		},
	}

	if c.Repeat {
		for {
			if err := c.doRequest(client); err != nil {
				return err
			}
			time.Sleep(1 * time.Second)
		}
	} else {
		if err := c.doRequest(client); err != nil {
			return err
		}
	}

	return nil
}

func (c *HttpChecker) doRequest(restClient *http.Client) error {

	uri := c.Url.String()
	r, err := http.NewRequest(c.Method, uri, c.Body)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", c.ContentType)

	fmt.Println("requesting... ", c.Method, uri)

	start := time.Now().UnixNano()
	resp, err := restClient.Do(r)
	fmt.Println("Elapsed time(ms) :", (time.Now().UnixNano()-start)/1000000)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println("Status :", resp.StatusCode)
	for k, v := range resp.Header {
		fmt.Println(k, ":", v)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(body))

	return nil
}
