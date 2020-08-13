package controllers

import (
	"adsmall-v2/api-item/config"
	"adsmall-v2/api-item/helper"
	"adsmall-v2/api-item/library/encryption"
	"adsmall-v2/api-item/structs/models"
	"adsmall-v2/api-item/structs/requests"
	"adsmall-v2/api-item/structs/responses"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"strings"
)

func (idb *InDB) UpdateItem(c *gin.Context) {
	var requestURI requests.UpdateItemURI
	var requestForm requests.UpdateItemForm
	var item models.Item
	var itemHeadlines models.Item
	var itemLocation models.ItemXLocation
	var dimension models.Dimension

	if err := c.ShouldBindUri(&requestURI); err != nil {
		errMessage := helper.ErrorValidationMessage(err)
		helper.CustomResponse(c, http.StatusBadRequest, "98", errMessage, nil)
		return
	}

	if err := c.ShouldBind(&requestForm); err != nil {
		errMessage := helper.ErrorValidationMessage(err)
		helper.CustomResponse(c, http.StatusBadRequest, "98", errMessage, nil)
		return
	}

	itemId, err := encryption.DecryptId(requestURI.ItemId)
	if err != nil {
		helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
		return
	}

	idb.DB.Where("item_id = ?", itemId).First(&item)
	if item.ItemId == 0 {
		helper.CustomResponse(c, http.StatusUnprocessableEntity, "96", "Data not found!", nil)
		return
	}

	idb.DB.Where("headlines = ?", requestForm.Headlines).First(&itemHeadlines)
	if itemHeadlines.ItemId != 0 {
		helper.CustomResponse(c, http.StatusUnprocessableEntity, "95", "Data already exists!", nil)
		return
	}

	tx := idb.DB.Begin()

	if err := tx.Error; err != nil {
		helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
		return
	}

	// item
	productId, err := encryption.DecryptId(requestForm.ProductId)
	if err != nil {
		helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
		return
	}
	storefrontId, err := encryption.DecryptId(requestForm.StorefrontId)
	if err != nil {
		helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
		return
	}

	updateParams := models.Item{
		ProductId:    productId,
		StorefrontId: storefrontId,
		Headlines:    requestForm.Headlines,
		Description:  requestForm.Description,
		MinimumOrder: requestForm.MinimumOrder,
		Price:        requestForm.Price,
		DisplayFlag:  requestForm.DisplayFlag,
	}

	if err := tx.Model(&item).Where("item_id = ?", itemId).Updates(&updateParams).Error; err != nil {
		tx.Rollback()
		helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
		return
	}

	// dimension
	tx.Select("dimension_id, item_id").Where("item_id = ?", itemId).First(&dimension)
	if dimension.DimensionId != 0 {
		var requestDimension requests.UpdateDimensionForm

		if dimension.Width != requestDimension.Width {
			updateDimensionParams := models.Dimension{
				Width: requestDimension.Width,
			}

			if err := tx.Model(&dimension).Where("dimension_id = ?", dimension.DimensionId).Update(&updateDimensionParams).Error; err != nil {
				tx.Rollback()
				helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
				return
			}
		}

		if dimension.Height != requestDimension.Height {
			updateDimensionParams := models.Dimension{
				Height: requestDimension.Height,
			}

			if err := tx.Model(&dimension).Where("dimension_id = ?", dimension.DimensionId).Update(&updateDimensionParams).Error; err != nil {
				tx.Rollback()
				helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
				return
			}
		}
	}

	// location
	tx.Select("item_x_location_id, location_id").Where("item_id = ?", itemId).First(&itemLocation)
	if itemLocation.ItemXLocationId != 0 {
		var requestLocation requests.UpdateLocationForm
		var location models.Location

		if err := c.ShouldBind(&requestLocation); err != nil {
			errMessage := helper.ErrorValidationMessage(err)
			helper.CustomResponse(c, http.StatusBadRequest, "98", errMessage, nil)
			return
		}

		tx.Where("location_id = ?", itemLocation.LocationId).First(&location)
		if location.Title != requestLocation.Title {
			locationCountryId, err := encryption.DecryptId(requestLocation.LocationCountryId)
			if err != nil {
				tx.Rollback()
				helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
				return
			}
			locationProvinceId, err := encryption.DecryptId(requestLocation.LocationProvinceId)
			if err != nil {
				tx.Rollback()
				helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
				return
			}
			locationCityId, err := encryption.DecryptId(requestLocation.LocationCityId)
			if err != nil {
				tx.Rollback()
				helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
				return
			}

			updateLocationParams := models.Location{
				LocationCountryId:  locationCountryId,
				LocationProvinceId: locationProvinceId,
				LocationCityId:     locationCityId,
				Latitude:           requestLocation.Latitude,
				Longitude:          requestLocation.Longitude,
				Title:              requestLocation.Title,
				Address:            requestLocation.Address,
				GoogleMaps:         requestLocation.GoogleMaps,
			}

			if err := tx.Model(&location).Where("location_id = ?", location.LocationId).Updates(&updateLocationParams).Error; err != nil {
				tx.Rollback()
				helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
				return
			}
		}
	}

	tx.Commit()

	helper.CustomResponse(c, http.StatusCreated, "00", "Record has been updated", nil)
	return
}

func (idb *InDB) DeleteItem(c *gin.Context) {
	var request requests.DeleteItem
	var item models.Item
	var itemLocation models.ItemXLocation
	var dimension models.Dimension

	if err := c.ShouldBindUri(&request); err != nil {
		errMessage := helper.ErrorValidationMessage(err)
		helper.CustomResponse(c, http.StatusBadRequest, "98", errMessage, nil)
		return
	}

	itemId, err := encryption.DecryptId(request.ItemId)
	if err != nil {
		helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
		return
	}

	tx := idb.DB.Begin()

	if tx.Where("item_id = ?", itemId).First(&item); item.ItemId == 0 {
		tx.Rollback()
		helper.CustomResponse(c, http.StatusUnprocessableEntity, "96", "Data not found!", nil)
		return
	}

	// dimension
	tx.Where("item_id = ?", itemId).First(&dimension)
	if dimension.DimensionId != 0 {
		if err := tx.Where("item_id = ?", itemId).Delete(&dimension).Error; err != nil {
			tx.Rollback()
			helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
			return
		}
	}

	// location
	tx.Where("item_id = ?", itemId).First(&itemLocation)
	if itemLocation.ItemXLocationId != 0 {
		var location models.Location

		if err := tx.Where("location_id = ?", itemLocation.LocationId).Delete(&location).Error; err != nil {
			tx.Rollback()
			helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
			return
		}

		if err := tx.Where("item_x_location_id = ?", itemLocation.ItemXLocationId).Delete(&itemLocation).Error; err != nil {
			tx.Rollback()
			helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
			return
		}
	}

	// item
	if err := tx.Where("item_id = ?", itemId).Delete(&item).Error; err != nil {
		tx.Rollback()
		helper.CustomResponse(c, http.StatusInternalServerError, "99", err.Error(), nil)
		return
	}
	tx.Commit()

	helper.CustomResponse(c, http.StatusOK, "00", "Record has been deleted", nil)
	return
}
