package server

import (
	"github.com/techx/playground/controllers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func newRouter() *echo.Echo {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	room := new(controllers.RoomController)
	e.GET("/rooms", room.GetRooms)
	e.POST("/rooms", room.CreateRoom)

	return e
}
