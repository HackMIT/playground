package server

import (
	"net/http"
	"os"

	"github.com/techx/playground/controllers"
	"github.com/techx/playground/socket"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func newRouter(hub *socket.Hub) *echo.Echo {
	e := echo.New()

	// Define middlewares
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	if os.Getenv("PRODUCTION") == "true" {
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				req := c.Request()

				if req.Header.Get("X-Forwarded-Proto") == "http" {
					return c.Redirect(http.StatusMovedPermanently, "https://"+req.Host+req.RequestURI)
				}

				return next(c)
			}
		})
	}

	// Rooms controller
	room := new(controllers.RoomController)
	e.GET("/rooms", room.GetRooms)

	return e
}
