package server

import (
	"github.com/techx/playground/controllers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func newRouter() *echo.Echo {
	e := echo.New()

	// Define middlewares
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Rooms controller
	room := new(controllers.RoomController)
	e.GET("/rooms", room.GetRooms)
	e.POST("/rooms", room.CreateRoom)
	e.POST("/rooms/:id/hallways", room.CreateHallway)

	return e
}
