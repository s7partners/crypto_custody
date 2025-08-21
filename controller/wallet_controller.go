package controller

import (
	"github.com/crypto_custody/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type WalletController struct {
	WalletService *service.WalletService
}

// GET /api/wallet/deposit/address
func (c *WalletController) GetDepositAddress(ctx *gin.Context) {
	userIdStr := ctx.Query("userId")
	currency := ctx.Query("currency")
	userId, err := strconv.ParseUint(userIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid userId"})
		return
	}

	addr, err := c.WalletService.GetDepositAddress(userId, currency)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"address": addr.Address})
}

// GET /api/wallet/deposit/history
func (c *WalletController) GetDepositHistory(ctx *gin.Context) {
	userIdStr := ctx.Query("userId")
	currency := ctx.Query("currency")
	pageStr := ctx.Query("page")
	sizeStr := ctx.Query("size")

	userId, _ := strconv.ParseUint(userIdStr, 10, 64)
	page, _ := strconv.Atoi(pageStr)
	size, _ := strconv.Atoi(sizeStr)

	records, total, err := c.WalletService.GetDepositHistory(ctx, userId, currency, page, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"total": total, "records": records})
}

// GET /api/wallet/withdraw/history
func (c *WalletController) GetWithdrawHistory(ctx *gin.Context) {
	userIdStr := ctx.Query("userId")
	currency := ctx.Query("currency")
	pageStr := ctx.Query("page")
	sizeStr := ctx.Query("size")

	userId, _ := strconv.ParseUint(userIdStr, 10, 64)
	page, _ := strconv.Atoi(pageStr)
	size, _ := strconv.Atoi(sizeStr)

	records, total, err := c.WalletService.GetWithdrawHistory(ctx, userId, currency, page, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"total": total, "records": records})
}

// GET /api/wallet/balance
func (c *WalletController) GetBalance(ctx *gin.Context) {
	userIdStr := ctx.Query("userId")
	currency := ctx.Query("currency")

	userId, _ := strconv.ParseUint(userIdStr, 10, 64)
	available, frozen, err := c.WalletService.GetBalance(ctx, userId, currency)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"availableBalance": available,
		"frozenBalance":    frozen,
	})
}
