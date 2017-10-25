package core_test

import (
	"testing"
	"github.com/nats-io/go-nats"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"time"
	"net/http"
	"gitlab.com/fino/banksearch/core"
	"gopkg.in/mgo.v2/bson"
)

func TestCreateNatsProxy(t *testing.T) {
	e := echo.New()
	e.HideBanner = true

	e.Use(core.BindMongoContext())
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	const apiPrefix = "/api/v0/banksearch"

	nc, _ := nats.Connect(nats.DefaultURL)
	c, _ := nats.NewEncodedConn(nc, nats.JSON_ENCODER)

	// Simple Publisher
	nc.Publish("foo", []byte("Hello World"))

	// Simple Async Subscriber
	nc.Subscribe("foo", func(m *nats.Msg) {
		t.Logf("Received a message: %s\n", string(m.Data))
	})

	e.GET(apiPrefix+"/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, bson.M{
			"pooow": "pufff",
		})
	})

	core.CreateNatsProxy(e, c)

	var resp bson.M
	err := c.Request(apiPrefix+"/", nil, &resp, time.Second * 15)
	if err == nil {
		t.Log(resp)
	} else {
		t.Log(err)
	}
}
