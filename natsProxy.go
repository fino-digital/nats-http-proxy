package natsproxy

import (
	"github.com/labstack/echo"
	"github.com/nats-io/go-nats"
	"net/http/httptest"
	"net/url"
	"net/http"
	"bytes"
	"encoding/json"
	"regexp"
	"time"
	"strings"
)

type (
	// Fake request object (you will need to use it to communicate with rest endpoints over http)
	Request struct {
		URL         url.URL            `protobuf:"bytes,1,opt,name=URL,json=uRL" json:"URL,omitempty"`
		Method      string             `protobuf:"bytes,2,opt,name=Method,json=method" json:"Method,omitempty"`
		RemoteAddr  string             `protobuf:"bytes,3,opt,name=RemoteAddr,json=remoteAddr" json:"RemoteAddr,omitempty"`
		Body        []byte             `protobuf:"bytes,4,opt,name=Body,json=body,proto3" json:"Body,omitempty"`
		Form        url.Values         `protobuf:"bytes,5,rep,name=Form,json=form" json:"Form,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
		PostForm    url.Values		   `protobuf:"bytes,5,rep,name=PostForm,json=postForm" json:"PostForm,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
		Header      http.Header        `protobuf:"bytes,6,rep,name=Header,json=header" json:"Header,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
		WebSocketID string             `protobuf:"bytes,7,opt,name=WebSocketID,json=webSocketID" json:"WebSocketID,omitempty"`
	}

	RestNatsEncodedConn struct {
		*nats.EncodedConn
	}
)

var (
	pathrgxp = regexp.MustCompile(":[A-z0-9$-_.+!*'(),]{1,}")
)
// SubscribeURLToNats buils the subscription
// channel name with placeholders
// The placeholders are than used to obtain path variables
func SubscribeURLToNats(urlPath string) string {
	subURL := pathrgxp.ReplaceAllString(urlPath, "*")
	subURL = strings.Replace(subURL, "/", ".", -1)
	return subURL
}

func RestRequest(nc *nats.EncodedConn, subj string, v interface{}, vPtr interface{}, timeout time.Duration) error {
	var req Request
	switch reqT := v.(type) {
	case Request:
		req = reqT
	default:
		req = Request{}
	}

	req.URL.Path = subj
	req.URL.RawPath = subj
	return nc.Request(SubscribeURLToNats(subj), req, vPtr, timeout)
}

func (rnc *RestNatsEncodedConn) RestRequest (subj string, v interface{}, vPtr interface{}, timeout time.Duration) error {
	req := v.(Request)
	req.URL.Path = subj
	return rnc.Request(SubscribeURLToNats(subj), req, vPtr, timeout)
}

// We use a nats.Conn here and not an EncodedConn because we only pass the encoded data on
func CreateNatsProxy(e *echo.Echo, c *nats.Conn) {
	// loop over the routes of the echo server and create a subscription to each of them
	r := regexp.MustCompile(":.*/")
	for _, route := range e.Routes() {
		// first we add the wildcards at the appropiate positions, then we replace the slashes with dots to make the wildcards work
		newRoute := SubscribeURLToNats(r.ReplaceAllString(route.Path, "*/"))
		c.Subscribe(newRoute, func(m *nats.Msg) {
			// get our fakes req obj from the message
			var req Request
			err := json.Unmarshal(m.Data, &req)

			reqMethod := req.Method
			if reqMethod == "" {
				reqMethod = "GET"
			}

			// Recreate a real request object from our fake object
			httpReq, err := http.NewRequest(reqMethod, "/" + req.URL.Path, bytes.NewReader(req.Body))
			if err != nil {
				return
			}

			httpReq.Header = req.Header
			httpReq.PostForm = req.PostForm
			httpReq.Form = req.Form
			httpReq.RemoteAddr = req.RemoteAddr

			// Make echo invoke our faked request
			rec := httptest.NewRecorder()
			ctx := e.NewContext(httpReq, rec)
			e.Router().Find(reqMethod, route.Path, ctx)
			e.ServeHTTP(ctx.Response(), ctx.Request())


			c.Publish(m.Reply, rec.Body.Bytes())
		})
	}
}
