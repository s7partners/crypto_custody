package handler

import (
	"github.com/crypto_custody/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type WalletHandler struct {
	svc *service.WalletService
}

func NewWalletHandler(svc *service.WalletService) *WalletHandler {
	return &WalletHandler{svc: svc}
}

// GET /api/wallet/deposit/address
func (h *WalletHandler) GetDepositAddress(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("userId"), 10, 64)
	currency := c.Query("currency")

	addr, err := h.svc.GetDepositAddress(userID, currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": addr.Address})
}

// GET /api/wallet/deposit/history
func (h *WalletHandler) GetDepositHistory(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("userId"), 10, 64)
	currency := c.Query("currency")
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))

	list, total, err := h.svc.GetDepositHistory(c, userID, currency, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "records": list})
}

// GET /api/wallet/withdraw/history
func (h *WalletHandler) GetWithdrawHistory(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("userId"), 10, 64)
	currency := c.Query("currency")
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))

	list, total, err := h.svc.GetWithdrawHistory(c, userID, currency, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "records": list})
}

// GET /api/wallet/balance
func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Query("userId"), 10, 64)
	currency := c.Query("currency")

	available, frozen, err := h.svc.GetBalance(c, userID, currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"availableBalance": available, "frozenBalance": frozen})
}
