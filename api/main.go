package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {

	dbd, err := NewDBD()
	if err != nil {
		panic(err)
	}
	cookie, err := os.ReadFile("./cookie.txt")
	if err != nil {
		panic(err)
	}
	dbd.SetCookie(string(cookie))

	type Auction struct {
		Id      int              `json:"id"`
		Price   float64          `json:"price"`
		Address string           `json:"address"`
		Detail  DBDAuctionDetail `json:"detail"`
	}
	auctions := make(map[int]Auction)

	r := gin.Default()
	r.GET("/product", func(c *gin.Context) {
		query := c.DefaultQuery("query", "")
		status := c.DefaultQuery("status", "")
		detail := c.DefaultQuery("detail", "false")
		page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		products, err := dbd.ProductSearch(query, status, page)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		// if query != "" {
		// 	ids := []int{}
		// 	for _, product := range products {
		// 		ids = append(ids, product.Id)
		// 	}

		// 	bidders, err := dbd.ProductBidder(ids...)
		// 	if err != nil {
		// 		c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
		// 		return
		// 	}

		// 	for i := range products {
		// 		id := products[i].Id
		// 		bidder, exist := bidders[id]
		// 		if !exist {
		// 			continue
		// 		}

		// 		products[i].Status = bidder.Status
		// 		products[i].SpectatorCount = bidder.SpectatorCount

		// 		products[i].CurrentPrice = bidder.CurrentPrice
		// 		products[i].CurrentBidder = bidder.CurrentBidder
		// 		products[i].CurrentBidderNickName = bidder.BidderNickName
		// 	}
		// }

		ids := []int{}
		for _, v := range products {
			ids = append(ids, v.Id)
		}

		bidders, err := dbd.ProductBidder(ids...)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		infos := []map[string]interface{}{}
		mores := []map[string]interface{}{}
		for _, v := range products {
			if bidder, exist := bidders[v.Id]; exist {
				v.Status = bidder.Status
				v.SpectatorCount = bidder.SpectatorCount

				v.CurrentPrice = bidder.CurrentPrice
				v.CurrentBidder = bidder.CurrentBidder
				v.CurrentBidderNickName = bidder.BidderNickName
			}

			price := map[string]interface{}{
				"curr":   v.CurrentPrice,
				"capped": v.CappedPrice,
			}
			if detail == "true" {
				time.Sleep(time.Second)

				detail, err := dbd.AuctionDetail(v.Id)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
					return
				}

				price = map[string]interface{}{
					"ave":    detail.HistoryPriceAve,
					"max":    detail.HistoryPriceMax,
					"min":    detail.HistoryPriceMin,
					"curr":   detail.AuctionInfo.CurrentPrice,
					"capped": v.CappedPrice,
				}
			}

			/*
				export type Auction = {
				  id: number;
				  name: string;
				  quality: string;
				  primary_pic: string;
				  status: string;
				  capped_price: number;
				  current_price: number;
				  current_bidder: string;
				  current_bidder_nickname: string;
				  start_timestamp: number;
				  end_timestamp: number;
				};
			*/

			// imageUrl: computed(
			// 	() => (primaryPic, size) => API.image_url + size + "/" + (primaryPic.startsWith("jfs") ? primaryPic : "jfs/" + primaryPic)
			//   ),

			_, isListened := auctions[v.Id]
			imageUrl := fmt.Sprintf("https://img10.360buyimg.com/%v/jfs/%v", "n4", strings.TrimPrefix(v.PrimaryPic, "jfs/"))

			infos = append(infos, map[string]interface{}{
				"id":                      v.Id,
				"name":                    v.ProductName,
				"quality":                 v.Quality,
				"primary_pic":             imageUrl,
				"status":                  v.Status,
				"listened":                isListened,
				"capped_price":            v.CappedPrice,
				"current_price":           v.CurrentPrice,
				"current_bidder":          v.CurrentBidder,
				"current_bidder_nickname": v.CurrentBidderNickName,
				"end_timestamp":           int64(v.EndTime),
				"start_timestamp":         int64(v.EndTime),
			})

			mores = append(mores, map[string]interface{}{
				"products": v,
				"detail":   detail,
				"info": map[string]interface{}{
					"price":    price,
					"id":       v.Id,
					"name":     v.ProductName,
					"status":   v.Status,
					"quality":  v.Quality,
					"listened": isListened,
					"time": map[string]interface{}{
						"end":    v.EndTime,
						"start":  v.StartTime,
						"remain": v.EndTime.Time().Sub(time.Now()),
					},
				},
			})
		}

		out := map[string]interface{}{
			"detail": infos,
			"mores":  mores,
		}

		outJson, _ := json.MarshalIndent(out, "", "  ")

		os.WriteFile("./output.json", outJson, 0644)

		c.JSON(http.StatusOK, out["detail"])
	})

	r.GET("/product/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Params.ByName("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		detail, err := dbd.AuctionDetail(id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		c.JSON(http.StatusOK, detail)
	})
	r.POST("/auction", func(c *gin.Context) {
		body := struct {
			Id      int     `json:"id"`
			Price   float64 `json:"price"`
			Address string  `json:"address"`
		}{}
		if err := c.Bind(&body); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		detail, err := dbd.AuctionDetail(body.Id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		auctions[body.Id] = Auction{
			Id:      body.Id,
			Price:   body.Price,
			Detail:  *detail,
			Address: body.Address,
		}
		c.JSON(http.StatusOK, struct{}{})
	})

	r.GET("/auction", func(c *gin.Context) {
		c.JSON(http.StatusOK, auctions)
	})

	r.GET("/auction/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Params.ByName("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		detail, exist := auctions[id]
		if !exist {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(fmt.Errorf("拍卖信息不存在")))
			return
		}

		c.JSON(http.StatusOK, detail)
	})

	r.GET("/auction/:id/didder", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Params.ByName("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}
		count, err := strconv.Atoi(c.DefaultQuery("count", "1"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		var prevBidder DBDProductBidder
		for i := 0; i < count; i++ {
			bidder, err := dbd.ProductBidder(id)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
				return
			}
			time.Sleep(100 * time.Millisecond)

			if reflect.DeepEqual(bidder[id], prevBidder) {
				fmt.Printf("%v,", i)
				continue
			}

			fmt.Println(time.Now())
			prevBidder = bidder[id]
			fmt.Printf("%v: %+v=%v\n", i, bidder[id], err)
		}

		c.JSON(http.StatusOK, prevBidder)
	})

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")

	return
}
