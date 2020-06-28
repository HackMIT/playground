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
	json_map := make(map[string]interface{})
	if err := json.NewDecoder(c.Request().Body).Decode(&json_map); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid json")
	}

	sponsor := new(models.Sponsor).Init(json_map["name"].(string), json_map["id"].(string), json_map["color"].(string));

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

// PUT /sponsor/<sponsor_id> - update an individual sponsor
// only supports changing color at the moment
func (s SponsorController) UpdateSponsor(c echo.Context) error {	
	// parse json body
	json_map := make(map[string]interface{})
	if err := json.NewDecoder(c.Request().Body).Decode(&json_map); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid json")
	}

	// update to Redis if color is in request body
	if (json_map["color"].(string) != "") {
		_, err := db.GetRejsonHandler().JSONSet("sponsor:" + c.Param("id"), "color", json_map["color"].(string))

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
									 "database error")
		}
	}

	// Fetch updated sponsor data from Redis
	var sponsor models.Sponsor
	sponsorData, _ := db.GetRejsonHandler().JSONGet("sponsor:" + c.Param("id"), ".")
	json.Unmarshal(sponsorData.([]byte), &sponsor)
	
	return c.JSON(http.StatusOK, sponsor)
}