package natsproxy_test

import (
	"testing"
	"github.com/nats-io/go-nats"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"time"
	"net/http"
	"github.com/fino-digital/nats-http-proxy"
	"github.com/stretchr/testify/assert"
)

func TestCreateNatsProxy(t *testing.T) {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())


	nc, _ := nats.Connect(nats.DefaultURL)
	c, _ := nats.NewEncodedConn(nc, nats.JSON_ENCODER)

	testData := map[string]string{
		"just": "some",
		"testing": "data",
	}

	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, testData)
	})

	natsproxy.CreateNatsProxy(e, nc)

	var resp map[string]string
	err := c.Request("/", nil, &resp, time.Second * 15)
	if assert.NoError(t, err) {
		assert.Equal(t, testData, resp)
	}
}
