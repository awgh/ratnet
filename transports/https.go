package transports

import (
	"bencrypt"
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"ratnet"
	"sync"
)

func init() {
	web := new(httpsModule)
	Servers["https"] = web
	Clients["https"] = web
	SyncHole["https"] = &sync.WaitGroup{}

	web.transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	web.client = &http.Client{Transport: web.transport}
}

// InitSSL - Generate you some SSL cert/key
func InitSSL(certfile string, keyfile string) {
	haveCert := true
	if _, err := os.Stat(keyfile); os.IsNotExist(err) {
		haveCert = false
	}
	if _, err := os.Stat(certfile); os.IsNotExist(err) {
		haveCert = false
	}
	if !haveCert {
		bencrypt.GenerateSSLCert(certfile, keyfile, bencrypt.ECC_MODE)
	}
}

type httpsModule struct {
	transport *http.Transport
	client    *http.Client
}

func (httpsModule) getName() string {
	return "HTTPS Web Module"
}

// Server interface
func (h *httpsModule) Accept(
	listen string, certfile string, keyfile string,
	db func() *sql.DB, adminMode bool) {

	// singleton setup things
	InitSSL(certfile, keyfile)
	// end singleton setup things

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		h.handleJSON(w, r, db, adminMode)
	})
	SyncHole["https"].Add(1)
	go func() {
		err := http.ListenAndServeTLS(listen, certfile, keyfile, serveMux)
		if err != nil {
			log.Println("Https Server Crashed: " + err.Error())
		}
		SyncHole["https"].Done()
	}()
}

func (httpsModule) handleJSON(w http.ResponseWriter, r *http.Request, db func() *sql.DB, adminPrivs bool) {
	var a ratnet.ApiCall
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&a)
	if err != nil {
		log.Println(err.Error())
	}

	result, err := ratnet.Api(&a, db, adminPrivs, "from remote: ")
	if err != nil {
		log.Println(err.Error())
	}
	w.Write([]byte(result))
}

/*
func (httpsModule) handleDispatch(params interface{}, msg []byte) error {

	log.Println("PrintModule handler:")
	log.Println("\n" + string(msg))
	return nil
}
*/

// client interface
func (h *httpsModule) RemoteAPIImpl(host string, a *ratnet.ApiCall) ([]byte, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "https":
		{
			b, err := json.Marshal(a)
			if err != nil {
				return nil, err
			}

			req, _ := http.NewRequest("POST", u.String(), bytes.NewReader(b))
			//req.Close = true
			req.Header.Add("Accept", "application/json")

			resp, err := h.client.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			buf, err := ioutil.ReadAll(resp.Body)
			return buf, err
		}
	}
	return nil, errors.New("Unknown URL Scheme")
}
