package router

import (
	"github.com/crypto_custody/handler"
	"github.com/gin-gonic/gin"
)

func SetupRouter(walletHandler *handler.WalletHandler) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api/wallet")
	{
		//api.GET("/deposit/address", walletHandler.GetDepositAddress)
		//api.POST("/withdraw", walletHandler.RequestWithdraw)

		api.GET("/deposit/address", walletHandler.GetDepositAddress)
		api.GET("/deposit/history", walletHandler.GetDepositHistory)
		api.GET("/withdraw/history", walletHandler.GetWithdrawHistory)
		api.GET("/balance", walletHandler.GetBalance)
	}

	return r
}
