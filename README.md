# Gin-LSAT

A middleware for [Gin](https://github.com/gin-gonic/gin) framework that uses [LSAT](https://lsat.tech/) (a protocol standard for authentication and paid APIs) and provides handler functions to accept microtransactions before serving ad-free content or any paid APIs.

## Installation

Assuming you've installed Go and Gin 

1. Run this:

```
go get github.com/DhananjayPurohit/gin-lsat
```

2. Create `.env` file (refer `.env_example`) and configure `LND_ADDRESS` and `MACAROON_HEX` for LND client or `LNURL_ADDRESS` for LNURL client, `LN_CLIENT_TYPE` (out of LND, LNURL) and `ROOT_KEY` (for minting macaroons).  

## Usage

### Simple JSON response
[This example](https://github.com/DhananjayPurohit/gin-lsat/blob/main/examples/main.go) shows how to use Gin-LSAT for serving simple JSON response:-

```
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/DhananjayPurohit/gin-lsat/ginlsat"
	"github.com/DhananjayPurohit/gin-lsat/ln"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{
			"code":    http.StatusAccepted,
			"message": "Free content",
		})
	})

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Failed to load .env file")
	}
	lnClient, err := ginlsat.InitLnClient(&ln.LNClientConfig{
		LNClientType: os.Getenv("LN_CLIENT_TYPE"),
		LNDConfig: ln.LNDoptions{
			Address:     os.Getenv("LND_ADDRESS"),
			MacaroonHex: os.Getenv("MACAROON_HEX"),
		},
		LNURLConfig: ln.LNURLoptions{
			Address: os.Getenv("LNURL_ADDRESS"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	lsatmiddleware, err := ginlsat.NewLsatMiddleware(&ginlsat.GinLsatMiddleware{
		Amount:   5,
		LNClient: lnClient,
	})
	if err != nil {
		log.Fatal(err)
	}

	router.Use(lsatmiddleware.Handler)

	router.GET("/protected", func(c *gin.Context) {
		lsatInfo := c.Value("LSAT").(*ginlsat.LsatInfo)
		if lsatInfo.Type == ginlsat.LSAT_TYPE_FREE {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusAccepted,
				"message": "Free content",
			})
		} else if lsatInfo.Type == ginlsat.LSAT_TYPE_PAID {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusAccepted,
				"message": "Protected content",
			})
		} else {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusInternalServerError,
				"message": fmt.Sprint(lsatInfo.Error),
			})
		}
	})

	router.Run("localhost:8080")
}
```

## Testing

Run `go test` to run tests.
