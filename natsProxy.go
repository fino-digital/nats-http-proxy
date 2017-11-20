package natsproxy

import (
	"github.com/labstack/echo"
	"github.com/nats-io/go-nats"
	"net/http"
	"encoding/json"
	"regexp"
	"time"
	"strings"
	"log"
	"bytes"
	"net/http/httptest"
	legnatsproxy "github.com/sohlich/nats-proxy"
	"net/url"
)

type (

	// Wrap sructs to expose some wrapper methods easily
	RestNatsConn struct {
		*nats.Conn
	}

	RestNatsEncConn struct {
		*nats.EncodedConn
	}
)

var (
	pathrgxp = regexp.MustCompile(":[A-z0-9$-_.+!*'(),]{1,}")
)
// SubscribeURLToNats buils the subscription
// channel name with placeholders
// The placeholders are then used to obtain path variables
func URLToNats(urlPath string) string {
	subUrl := pathrgxp.ReplaceAllString(urlPath, "*")
	subUrl = strings.Replace(subUrl, "/", ".", -1)

	subUrl = strings.Trim(subUrl, "./")
	return subUrl
}

// RestRequest - Wrapper to make it to change the given http req into a serializable one
func (rnc *RestNatsConn)RestRequest(subj string, req *http.Request, timeout time.Duration) (*nats.Msg, error) {
	natsReq :=legnatsproxy.NewRequest()
	err := natsReq.FromHTTP(req)

	log.Println("making req to:" + URLToNats(subj))

	if err !=nil {
		return nil, err
	}

	jsonReq, err := json.Marshal(natsReq)
	if err !=nil {
		return nil, err
	}

	log.Println("making req to22:" + URLToNats(subj))

	return rnc.Request(URLToNats(subj), jsonReq, timeout)
}

// RestRequest - Wrapper to make it to change the given http req into a serializable one
func (rnec *RestNatsEncConn) RestRequest (subj string, req *http.Request, vPtr interface{}, timeout time.Duration) error {
	natsReq := legnatsproxy.NewRequest()
	err := natsReq.FromHTTP(req)

	if err !=nil {
		return err
	}

	return rnec.Request(URLToNats(subj), natsReq, vPtr, timeout)
}

// ToHTTP - creates a real http req from a fake but serializable one
func ToHTTP(req *legnatsproxy.Request) (*http.Request, error) {
	// replace our custom prefix from the actual path
	req.URL = strings.Replace(req.URL, "/nats/", "/", -1)
	parsedURL, err := url.Parse(req.URL)

	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(req.Method, parsedURL.RawPath, bytes.NewReader(req.Body))
	if err != nil {
		return nil, err
	}

	httpReq.URL = parsedURL

	// TODO: Evaluate if concurrency could be helpful here
	for k := range req.Header {
		httpReq.Header.Add(k, req.GetHeader().Get(k))
	}

	for k, v := range req.Form {
		httpReq.Form.Add(k, v.String())
	}

	httpReq.RemoteAddr = req.RemoteAddr

	return httpReq, nil
}

// We use a nats.Conn here and not an EncodedConn because we only pass the encoded data on
func CreateNatsProxy(e *echo.Echo, c *nats.Conn) {
	// loop over the routes of the echo server and create a subscription to each of them
	r := regexp.MustCompile(":.*/")
	for _, route := range e.Routes() {
		// first we add the wildcards at the appropiate positions, then we replace the slashes with dots to make the wildcards work
		newRoute := "nats."+ URLToNats(r.ReplaceAllString(route.Path, "*/"))
		log.Println("Adding to nats: " + newRoute)
		routePath := route.Path
		c.Subscribe(newRoute, func(m *nats.Msg) {
			log.Println("Got req for "+ routePath)
			log.Println(string(m.Data))
			// get our fakes req obj from the message
			var req legnatsproxy.Request
			err := json.Unmarshal(m.Data, &req)

			// Recreate a real request object from our fake object
			httpReq, err := ToHTTP(&req)
			if err != nil {
				return
			}

			// Make echo invoke our faked request
			rec := httptest.NewRecorder()
			ctx := e.NewContext(httpReq, rec)
			e.Router().Find(req.Method, routePath, ctx)
			e.ServeHTTP(ctx.Response(), ctx.Request())


			c.Publish(m.Reply, rec.Body.Bytes())
		})
	}
}
