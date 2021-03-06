package ginlsat

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/getAlby/gin-lsat/ln"
	"github.com/getAlby/gin-lsat/lsat"
	"github.com/getAlby/gin-lsat/macaroon"
	macaroonutils "github.com/getAlby/gin-lsat/macaroon"
	"github.com/getAlby/gin-lsat/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
)

const (
	LND_CLIENT_TYPE   = "LND"
	LNURL_CLIENT_TYPE = "LNURL"
)

const (
	LSAT_TYPE_FREE = "FREE"
	LSAT_TYPE_PAID = "PAID"
)

const (
	FREE_CONTENT_MESSAGE      = "Free Content"
	PROTECTED_CONTENT_MESSAGE = "Protected Content"
	PAYMENT_REQUIRED_MESSAGE  = "Payment Required"
)

type LsatInfo struct {
	Type     string
	Preimage lntypes.Preimage
	Mac      *macaroon.MacaroonIdentifier
	Amount   int64
	Error    error
}

type GinLsatMiddleware struct {
	Amount   int64
	Response func(c *gin.Context, code int, message string)
	LNClient ln.LNClient
}

func NewLsatMiddleware(mw *GinLsatMiddleware) (*GinLsatMiddleware, error) {
	mw.Response = func(c *gin.Context, code int, message string) {
		c.JSON(code, gin.H{
			"code":    code,
			"message": message,
		})
	}
	return mw, nil
}

func InitLnClient(lnClientConfig *ln.LNClientConfig) (ln.LNClient, error) {
	var lnClient ln.LNClient
	err := godotenv.Load(".env")
	if err != nil {
		return lnClient, errors.New("Failed to load .env file")
	}

	switch lnClientConfig.LNClientType {
	case LND_CLIENT_TYPE:
		lnClient, err = ln.NewLNDclient(lnClientConfig.LNDConfig)
		if err != nil {
			return lnClient, fmt.Errorf("Error initializing LN client: %s", err.Error())
		}
	case LNURL_CLIENT_TYPE:
		lnClient = &lnClientConfig.LNURLConfig
	default:
		return lnClient, fmt.Errorf("LN Client type not recognized: %s", lnClientConfig.LNClientType)
	}

	return lnClient, nil
}

func (lsatmiddleware *GinLsatMiddleware) Handler(c *gin.Context) {

	acceptLsatField := c.Request.Header["Accept"]
	// Check if client support LSAT

	authField := c.Request.Header["Authorization"]
	mac, preimage, err := utils.ParseLsatHeader(authField)

	// If macaroon and preimage are valid
	if err == nil {
		rootKey := utils.GetRootKey()

		// Check valid LSAT and set LSAT type Paid
		err := lsat.VerifyLSAT(mac, rootKey[:], preimage)
		if err != nil {
			c.Set("LSAT", &LsatInfo{
				Type: LSAT_TYPE_PAID,
			})
		}
	} else if len(acceptLsatField) != 0 && acceptLsatField[0] == `application/vnd.lsat.v1.full+json` {
		// Generate invoice and token
		ctx := context.Background()
		lnInvoice := lnrpc.Invoice{
			Value: lsatmiddleware.Amount,
			Memo:  "LSAT",
		}
		LNClientConn := &ln.LNClientConn{
			LNClient: lsatmiddleware.LNClient,
		}
		invoice, paymentHash, err := LNClientConn.GenerateInvoice(ctx, lnInvoice)
		if err != nil {
			c.Error(err)
			c.Set("LSAT", &LsatInfo{
				Error: err,
			})
			return
		}
		macaroonString, err := macaroonutils.GetMacaroonAsString(paymentHash)
		if err != nil {
			c.Error(err)
			c.Set("LSAT", &LsatInfo{
				Error: err,
			})
			return
		}
		c.Writer.Header().Set("WWW-Authenticate", fmt.Sprintf("LSAT macaroon=%s, invoice=%s", macaroonString, invoice))
		lsatmiddleware.Response(c, http.StatusPaymentRequired, PAYMENT_REQUIRED_MESSAGE)
		c.Abort()
	} else {
		// Set LSAT type Free if client does not support LSAT
		c.Set("LSAT", &LsatInfo{
			Type: LSAT_TYPE_FREE,
		})
	}

}
