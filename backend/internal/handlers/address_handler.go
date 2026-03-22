package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type AddressHandler struct {
	addr *services.AddressService
}

func NewAddressHandler(addr *services.AddressService) *AddressHandler {
	return &AddressHandler{addr: addr}
}

// List godoc
// @Summary      List shipping addresses
// @Tags         addresses
// @Produce      json
// @Success      200  {object}  Response{data=AddressesResponse}
// @Security     BearerAuth
// @Router       /users/me/addresses [get]
func (h *AddressHandler) List(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)

	addrs, err := h.addr.List(userID)
	if err != nil {
		internal(c)
		return
	}
	ok(c, gin.H{"addresses": addrs})
}

// Create godoc
// @Summary      Create a shipping address
// @Tags         addresses
// @Accept       json
// @Produce      json
// @Param        body body services.AddressInput true "Address payload"
// @Success      201  {object}  Response{data=AddressResponse}
// @Failure      400  {object}  Response
// @Security     BearerAuth
// @Router       /users/me/addresses [post]
func (h *AddressHandler) Create(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)

	var input services.AddressInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	addr, err := h.addr.Create(userID, input)
	if err != nil {
		internal(c)
		return
	}
	created(c, gin.H{"address": addr})
}

// Update godoc
// @Summary      Update a shipping address
// @Tags         addresses
// @Accept       json
// @Produce      json
// @Param        id   path string true "Address UUID"
// @Param        body body services.AddressInput true "Address payload"
// @Success      200  {object}  Response{data=AddressResponse}
// @Failure      400  {object}  Response
// @Failure      404  {object}  Response
// @Security     BearerAuth
// @Router       /users/me/addresses/{id} [put]
func (h *AddressHandler) Update(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)
	addrID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid address id")
		return
	}

	var input services.AddressInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, err.Error())
		return
	}

	addr, err := h.addr.Update(userID, addrID, input)
	if err != nil {
		switch err {
		case services.ErrAddressNotFound:
			notFound(c, "address not found")
		default:
			internal(c)
		}
		return
	}
	ok(c, gin.H{"address": addr})
}

// Delete godoc
// @Summary      Delete a shipping address
// @Tags         addresses
// @Produce      json
// @Param        id path string true "Address UUID"
// @Success      200  {object}  Response{data=MessageResponse}
// @Failure      400  {object}  Response
// @Failure      404  {object}  Response
// @Security     BearerAuth
// @Router       /users/me/addresses/{id} [delete]
func (h *AddressHandler) Delete(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)
	addrID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid address id")
		return
	}

	if err := h.addr.Delete(userID, addrID); err != nil {
		notFound(c, "address not found")
		return
	}
	ok(c, gin.H{"message": "address deleted"})
}

// SetDefault godoc
// @Summary      Set an address as default
// @Tags         addresses
// @Produce      json
// @Param        id path string true "Address UUID"
// @Success      200  {object}  Response{data=AddressResponse}
// @Failure      400  {object}  Response
// @Failure      404  {object}  Response
// @Security     BearerAuth
// @Router       /users/me/addresses/{id}/default [put]
func (h *AddressHandler) SetDefault(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, _ := uuid.Parse(claims.UserID)
	addrID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		badRequest(c, "invalid address id")
		return
	}

	addr, err := h.addr.SetDefault(userID, addrID)
	if err != nil {
		notFound(c, "address not found")
		return
	}
	ok(c, gin.H{"address": addr})
}
