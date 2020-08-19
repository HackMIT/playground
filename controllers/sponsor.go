package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"

	"github.com/labstack/echo/v4"
)

type SponsorController struct {}

// POST /sponsor - creates a new sponsor
func (s SponsorController) CreateSponsor(c echo.Context) error {
	// Create new sponsor model, parse JSON body
	var sponsor = new(models.Sponsor)
	if err := c.Bind(sponsor); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid json")
	}

	// Add new sponsor to Redis
	_, err := db.GetRejsonHandler().JSONSet("sponsor:" + sponsor.Id, ".", sponsor)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
		                         "database error")
	}

	return c.JSON(http.StatusOK, sponsor)
}

// GET /sponsor/<sponsor_id> - get an individual sponsor
func (s SponsorController) GetSponsor(c echo.Context) error {
	// Fetch this sponsor from Redis
	var sponsor models.Sponsor
	sponsorData, _ := db.GetRejsonHandler().JSONGet("sponsor:" + c.Param("id"), ".")
	json.Unmarshal(sponsorData.([]byte), &sponsor)

	return c.JSON(http.StatusOK, sponsor)
}

// GET /sponsor - gets all sponsors
func (s SponsorController) GetSponsors(c echo.Context) error {
	// Get all of the room names from Redis
	sponsorData, err := db.GetInstance().Keys("sponsor:*").Result()

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
		                         "database error")
	}

	// Load each room into this array
	sponsors := make([]models.Room, len(sponsorData))

	// something happens to get all the rooms
	// for i, name := range sponsorData {
	// 	// Error here is unlikely because we already fetched from the DB
	// 	sponsor, _ := db.GetRejsonHandler().JSONGet(name, ".")
	// 	json.Unmarshal(roomData.([]byte), &rooms[i])
	// }

	return c.JSON(http.StatusOK, sponsors)


	
}