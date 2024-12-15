package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja/ast"
	"github.com/gin-gonic/gin"
)

type Auction struct {
	Id             int              `json:"id"`
	Price          float64          `json:"price"`
	Address        string           `json:"address"`
	Detail         DBDAuctionDetail `json:"detail"`
	EndTime        UnixTime         `json:"end_time"`
	PurchasePrice  float64          `json:"purchase_price"`
	PurchaseStatus string           `json:"purchase_status"`
}

var dbd *dbdClient = nil
var auctions = make(map[int]*Auction)

var purchaseLook sync.RWMutex
var purchaseAuctions = make(map[int]*Auction)

func purchaseLoop() {
	for {
		purchaseLook.RLock()
		ids := []int{}
		for id := range purchaseAuctions {
			ids = append(ids, id)
		}
		purchaseLook.RUnlock()

		if len(ids) == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		bidders, err := dbd.ProductBidder(ids...)
		if err != nil {
			panic(err)
		}

		for id, bidder := range bidders {
			purchaseLook.RLock()
			auction := purchaseAuctions[id]
			purchaseLook.RUnlock()

			fmt.Printf("[开始出价] (最高出价：%v,当前出价 %v) %s => %v\n", auction.Price, auction.PurchasePrice, auction.Detail.AuctionInfo.ProductName, bidder)

			if bidder.Status == 3 {
				purchaseLook.Lock()
				delete(purchaseAuctions, id)
				purchaseLook.Unlock()

				auction.PurchaseStatus = "出价失败，当前商品竞拍已结束"
				fmt.Printf("[结束出价] (最高出价：%v,当前出价 %v) %s\n", auction.Price, auction.PurchasePrice, auction.Detail.AuctionInfo.ProductName)
				continue
			}

			if bidder.CurrentBidder == "98***15" && bidder.BidderNickName == "只要***一点" && bidder.BidderImage == `http://storage.360buyimg.com/i.imageUpload/393838383733342d33313335313531343539353635313632373133_big.jpg` {
				continue
			}

			if bidder.CurrentPrice+1 >= auction.Price {
				purchaseLook.Lock()
				delete(purchaseAuctions, id)
				purchaseLook.Unlock()

				auction.PurchaseStatus = "出价失败，商品当前价格过高"
				fmt.Printf("[结束出价] (最高出价：%v,当前出价 %v) %s\n", auction.Price, auction.PurchasePrice, auction.Detail.AuctionInfo.ProductName)
				continue
			}

			go func() {
				auction.PurchasePrice = bidder.CurrentPrice + 1
				if err := dbd.AuctionPrice(auction.Id, auction.Address, auction.PurchasePrice); err != nil {
					auction.PurchaseStatus = err.Error()
				} else {
					auction.PurchaseStatus = "出价成功"
				}
			}()
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func loop() {
	for {
		for i, auction := range auctions {
			remain := auction.EndTime - UnixTime(time.Now().UnixMilli())

			if remain > 1000 {
				continue
			}
			delete(auctions, i)

			fmt.Printf("[启动出价] (最高出价：%v) %s\n", auction.Price, auction.Detail.AuctionInfo.ProductName)

			purchaseLook.Lock()
			purchaseAuctions[i] = auction
			purchaseLook.Unlock()
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func replaceRegexp(body []byte, regex string, cb func(matchs [][]byte) []byte) []byte {
	callRegex, err := regexp.Compile(regex)
	if err != nil {
		panic(err)
	}

	for {
		matchIndex := callRegex.FindIndex(body)
		if matchIndex == nil {
			break
		}
		matchs := callRegex.FindSubmatch(body)
		if matchIndex == nil {
			break
		}

		replaceDate := cb(matchs)

		newBody := append([]byte{}, body[:matchIndex[0]]...)
		newBody = append(newBody, replaceDate...)
		newBody = append(newBody, body[matchIndex[1]:]...)

		if false {
			fmt.Printf("matchStrs: (%v) = (%v)\n", string(matchs[0]), string(replaceDate))
		}

		body = newBody
	}

	return body
}
func evalJSAlias(vm *goja.Runtime, call string, aliasName string, body []byte) []byte {
	return evalJS(vm, body, call, func(matchs [][]byte) string {
		v := ""
		callStr := string(matchs[0])
		if aliasName != "" {
			funcName := string(matchs[1])
			v += fmt.Sprintf("var %s = %s; ", funcName, aliasName)
		}
		v += callStr

		return v
	})
}

func evalJS(vm *goja.Runtime, body []byte, call string, cb func(matchs [][]byte) string) []byte {
	return replaceRegexp(body, call, func(matchs [][]byte) []byte {
		js := cb(matchs)

		val, err := vm.RunString(js)
		if err != nil {
			panic(err)
		}

		valJSON, err := json.Marshal(val.Export())
		if err != nil {
			panic(err)
		}

		switch v := val.Export().(type) {
		case int64:
			valJSON = []byte("0x" + strconv.FormatInt(int64(v), 16))
			if vStr := strconv.FormatInt(int64(v), 10); len(vStr) < len(valJSON) {
				valJSON = []byte(vStr)
			}
		case float64:
			if math.Floor(v) == v {
				valJSON = []byte("0x" + strconv.FormatInt(int64(v), 16))
				if vStr := strconv.FormatInt(int64(v), 10); len(vStr) < len(valJSON) {
					valJSON = []byte(vStr)
				}
			}
		}

		if false {
			fmt.Printf("matchJs = %v = %v\n", js, string(valJSON))
		}

		return valJSON
	})
}
func decodeJS(filename string, descript string, cb func(vm *goja.Runtime, body []byte) []byte) error {
	body, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("文件有问题,%v", err)
	}

	vm := goja.New()
	_, err = vm.RunString(descript)
	if err != nil {
		return fmt.Errorf("JS代码有问题,%v", err)
	}

	body = cb(vm, body)
	return os.WriteFile(filename+"_de", body, 0644)
}

var (
	descript = `

    function _4u94f(s) {
        var o = '';
        for (var i = 0; i < s.length; ) {
            var c = s.charCodeAt(i++);
            if (c > 63)
                o += String.fromCharCode(c ^ 35);
            else if (c == 35)
                o += s.charAt(i++);
            else
                o += String.fromCharCode(c);
        }
        return o;
    }

    function a060c66O() {
        var uI = ['v1fFz2f0AgvYx2n2mq', 'mc4XlJC', 'yxbWBgLJyxrPB24VEc13D3CTzM9YBs11CMXLBMnVzgvK', 'D2vI', 'yxr0CLzLCNrLEa', 'A2v5CW', 'ENHJyxnK', 'lcbHBgDVoG', 'v0vcr0XFzgvIDwDFCMvUzgvYzxjFAw5MBW', 'ChjVCgvYDhLjC0vUDw1LCMfIBgu', 'Dw5Oyw5KBgvKuMvQzwn0Aw9U', 'u3LTyM9SigLZig5VDcbHignVBNn0CNvJDg9Y', 'BwfPBI5ZAwDUi19Fzgv0zwn0Aw5N', 'D3v2oG', 'ihrVA2vUoG', 'CMvXDwvZDcb0B2TLBIbMywLSzwqGA2v5oG', 'BMv4Da', 'BNvTyMvY', 'Dg9ju09tDhjPBMC', 'AxnxzwXSs25VD25tEw1IB2W', 'kd86psHBxJTDkIKPpYG7FcqP', 'C29TzxrOAw5N', 'w29IAMvJDcbpyMPLy3rD', 'zxjYB3jZ', 'mtuUnhb4icDbCMLHBcC', 'ChjVy2vZCW', 'w3nPz25Dia', 'yxbWBgLJyxrPB24VANnVBG', 'uNzjpdD8tKmTmwC1', 'qebPDgvYyxrVCG', 'C2vHCMnO', 'AhrTBgzPBgu', 'CgfYC2vYzxjYB3i', 'C3rHDgu', 'zxHWzxjPBwvUDgfSlxDLyMDS', 'iZfHm2jJmq', 'CgfYyw1ZigLZig5VDcbHihbSywLUig9IAMvJDa', 't2jQzwn0', 'v3jVBMCGBNvTyMvYig9MihjLCgv0AxrPB25Z', 'DZi1', 'DZiZ', 'ChDKDf9Pza', 'Ahr0Chm6lY9NAxrODwiUy29Tl3PSB2LYB2nRl2nVCMuTANm', 'cqOlda0GWQdHMOdIGidIGihIGilIGipIGitIGixIGiBIGiFIGiJIGiNIGiRIGk/IGz/JGidIGkJIGkNVU78', 'mdaW', 'CNfWB25TBgTQAwHNzMvKy2jHwLLyv1zvvfnsuvbptK1ms0PjseDgrurdqKeTxZK4nZy1ndmYmtb6ExH3DNv0CW', 'v2LUzg93', 'zg9JDw1LBNrfBgvTzw50', 'C3vH', 'DgHYB3C', 'CMv0DxjUihrOAxm', 'u3rYAw5NieL0zxjHDg9Y', 'ExL5Eu1nzgrOAg1TC3ntu1m', 'D3jPDgfIBgu', 'C29YDa', 'zgL2', 'qxn5BMngDw5JDgLVBG', 'B2jQzwn0', 'q2fUj3qGy29UDMvYDcbVyMPLy3qGDg8GChjPBwL0AxzLihzHBhvL', 'DZiX', 'CgfYyw1ZignVBNrHAw5ZihjLC2vYDMvKihbHCMfTig5HBwuU', 'z2v0', 'lcbYzxrYEsbUzxH0ihrPBwuU', 'y2fUDMfZ', 'C3LTyM9S', 'C2LNBIbLBgfWC2vKihrPBwuH', 'ieL0zxjHDg9Y', 'yxn5BMnjDgvYyxrVCG', 'w29IAMvJDcb6xq', 'yNuX', 't2jQzwn0igfSCMvHzhKGAw5PDgLHBgL6zwq', 'x19Yzxf1zxn0rgvWCYb1C2uGy2fJAguGzNaSigzWoG', 'qwnJzxnZB3jZig5VDcbZDxbWB3j0zwq', 'z2v0vg9Rzw5F', 'CMvQzwn0Aw9UsgfUzgXLza', 'zgLHBNrVDxnOAs5JB20', 'Dg9mB2nHBgvtDhjPBMC', 'Bwf0y2HLCG', 'y29UC3rYDwn0', 'D3vYoG', 'mdeYmZq1nJC4owfIy2rLzMDOAwPRBg1UB3bXCNn0Dxz3EhL6qujdrevgr0HjsKTmtu5puffsu1rvvLDywvPFlq', 'igLZig5VDcbHigz1BMn0Aw9U', 'igLZig5VDcbPDgvYywjSzq', 'tM/PQPC', 'Cgf0DgvYBK1HDgnO', 'y2nU', 'rgf0zq', 'sw5JB21WyxrPyMXLihjLy2vPDMvYlca', 'Chb6Ac5Qzc5JB20', 'C2XPy2u', 'igfZigeGChjVDg90ExbL', 'AdvZDa', 'DgvZDcbLCNi', 'CMvK', 'Bg9HzcbYywmGANmGzMfPBce', 'Bwv0ywrHDgflzxK', 'yxr0CMLIDxrLihzLyZiGyxr0CLzLCNrLEdT2yxj5Aw5NihzLyZiGDMfYEwLUvgv4q29VCMrPBMf0ztT1BMLMB3jTihzLyZiGDw5PzM9YBu9MzNnLDdT2B2LKig1HAw4OkxT2yxj5Aw5uzxHdB29YzgLUyxrLpwf0Dhjwzxj0zxGRDw5PzM9YBu9MzNnLDdTNBf9qB3nPDgLVBJ12zwm0kgf0Dhjwzxj0zxGSmcWXktT9', 'lgv4ChjLC3m9', 'q2fUBM90ignVBNzLCNqGysbtEw1IB2WGDMfSDwuGDg8GysbZDhjPBMC', 'CMvWBgfJzq', 'ywXWAgfIzxq', 'CMfUzg9T', 'x19LC01VzhvSzq', 'Ahr0Chm6lY9NAxrODwiUy29Tl3PSB2LYB2nRl2nVCMuTANmVyMXVyI92mY4ZnI4Xl0Xjq0vou0u', 'B25YzwfKExn0yxrLy2HHBMDL', 'C3LTyM9SlxrVlxn0CMLUzY1YzwDPC3rYEq', 'DZeW', 'v1fFz2f0AgvYx3DNBde', 'C3bSAwnL', 'ExL5Es1nts1Kza', 'AwzYyw1L', 'x19Yzxf1zxn0qwXNB3jPDgHTigvUDKnVBgXLy3q9', 'ChrFCgLU', 'D2vIz2W', 'u3LTyM9Ska', 'u3LTyM9S', 'Dw5PzM9YBu9MzNnLDa', 'DZiW', 'C2HHBq', 'ChjLy2LZAw9Uig1LzgL1BxaGzMXVyxq7DMfYEwLUzYb2zwmYihzHCNLPBLrLEenVB3jKAw5HDgu7DM9PzcbTywLUkcKGE2DSx0zYywDdB2XVCJ12zwm0khzHCNLPBLrLEenVB3jKAw5HDguSmcWXktT9', 'qxjYyxKGsxrLCMf0B3i', 'yxbWAwq', 'kf58w14', 'lgTLEt0', 'Bg9HzgvYlNv0AwXZi2XVywrsywnty3jPChrpBMnL', 'nti0ng1Zwej2DG', 'Bg9HzgvK', 'svf6ovDt', 'C3OUAMqUy29T', 'kf58icK', 'AdvFzMLSzv92nc45lJC', 'tM90igvUB3vNAcbHCMD1BwvUDhm', 'Dg9qCMLTAxrPDMu', 'DZiY', 'y2rJx2fKB1fWB2fZBMzHnZzWzMnAtg1JzMXFu3LTyM9S', 'y29UC3rYDwn0B3i', 'ChvYzq', 'Bg9HzcbYywmGANmGC3vJy2vZCYe', 'C3rYAw5N', 'uhjVDg90ExbL', 'x19WCM90B19F', 'mta0odyYmJrfuvf5vfa', 'uMvNrxHW', 'AMf2yq', 'ig9Mia', 'Dg9Rzw4GAxmGzw1WDhK', 'D2vIz2XgCde', 'CMvQzwn0zwq', 'WQKGmJaXnc0Ymdi0ierLBMLZifb1C2HRyxjLDIaOEMXVAxjVy2SUCNuP', 'zNvSzMLSBgvK', 'rvHux3rLEhr1CMvFzMLSDgvYx2fUAxnVDhjVCgLJ', 'w25HDgL2zsbJB2rLxq', 'zxH0zw5K', 'z2v0t3DUuhjVCgvYDhLoyw1LCW', 'q29UDgvUDc1uExbL', 'ue9tva', 'ExL5Eu1nzgq', 'nc45', 'DZe4', 'ndG3mda1nwPkz2H1AG', 'CMvXDwvZDcbWyxjHBxmGzxjYB3iU', 'uhjVBwLZzq', 'iLX1zgyWnLX1zdGZnci', 'z2vUzxjHDguGA2v5igzHAwXLza', 'DZeZ', 'x19Yzxf1zxn0rgvWCYbLBMqU', 'mZm4otu4mLjQBhfrEa', 'DZe1', 'tM8GB25LihbYB21PC2uGCMvZB2X2zwq', 'Dg9tDhjPBMDuywC', 'uMvMBgvJDa', 'zw51BwvYywjSzq', 'uhjVBwLZzsbJyw4NDcbIzsbYzxnVBhzLzcbPDhnLBgy', 'mte3C2PUvu1K', 'Bg9JywXFA2v5xZm', 'vw5Oyw5KBgvKihbYB21PC2uGCMvQzwn0Aw9U', 'ntu1mZq4ofvnEgT0uG', 'twf4Aw11BsbHBgXVD2vKigLUzgv4igv4y2vLzgvK', 'q2fUBM90ihnLDcbYzwfKig9UBhKGlMXLBMD0Aa', 'AxrLCMf0B3i', 'zg9JDw1LBNqUrJ1pyMPLy3q', 'v0vcs0Lux0vyvf90zxH0DxjLx2zPBhrLCL9HBMLZB3rYB3bPyW', 'DZe0', 'x19TywTLu2LNBIWGCMvZDwX0oG', 'lcbLpq', 'DZe2', 'CMvQzwn0Aw9UAgfUzgXLza', 'Dw5Oyw5KBgvKCMvQzwn0Aw9U', 'rxjYB3i', 'x19JB2XSzwn0igvUDKnVBgXLy3q9', 'zMLSDgvY', 'zMLSztO', 'AxndB25JyxrtChjLywrHyMXL', 'r0vu', 'Ahr0Chm6lY9Jywn0DxmUAMqUy29Tl3jLCxvLC3rFywXNBW', 'y2f1C2u', 'CMvXDwvZDcbLCNjVCIWG', 'C2v0', 'qwnJzxb0', 'x19JB3jLlwPZx3nOyxjLzf9F', 'x19Yzxf1zxn0rgvWCYb1C2uGBMv3igzWlcbMCdO', 'C3LTyM9SigrLDgvJDgLVBG', 'qwDNCMvNyxrLrxjYB3i', 'DgHLBG', 'DMfSDwvpzG', 'yM9VBgvHBG', 'ywjJzgvMz2HPAMTSBw5VChfYC3r1DND4ExPbqKneruzhseLks0XntK9quvjtvfvwv1HzwG', 'tNvSBa', 'y2rJx2fKB1fWB2fZBMzHnZzWzMnAtg1JzMXFqxjYyxK', 'mJu3ntC1nhbQA0jiBG', 'tu9Ax0vyvf90zxH0DxjLx2zPBhrLCL9HBMLZB3rYB3bPyW', 'CgLU', 'zgvZy3jPChrPB24', 'ChjVDg90ExbL', 'yNuY', 'Ahr0Chm6lY9ZDg9YywDLlJm2mgj1EwLTzY5JB20VD2vIy29UDgfPBMvYl21HAw4VANmTC2vJDxjPDhKTDJmTCMfJlMPZp3y9', 'y2fUDMfZmq', 'lcbMCdO', 'CMvMzxjLCG', 'ANnVBG', 'BMfTzq', 'zw52q29SBgvJDa', 'CgfYyw1ZigLZigvTChr5', 'uhjVBwLZzs1JAgfPBIbJEwnSzq', 'lcb0B2TLBJO', 'Aw5JBhvKzxm', 'mtqXndC0ounxBgnQva', 'lcbZDg9YywDLrNa6', 'C3vJy2vZCW', 'mdm4ns0WnY0YnvqWnZOWnJOZos45otLA', 'tNvTyMvY', 'D2vIz2XgCa', 'mdeYmZq1nJC4oq', 'q2fUBM90igrLBgv0zsbWCM9Wzxj0Esa', 'q2fUj3qGC2v0ia', 'zgvMyxvSDa', 'Dgv4Dc9QyxzHC2nYAxb0', 'BwfW', 'Dw5Zy29WywjSzxm', 'igLZig5VDcbHignVBNn0CNvJDg9Y', 'yxn5BMneAxnWB3nL', 'u3rYAw5N', 'rNvUy3rPB24', 'Dw5RBM93BIbLCNjVCG', 'y2rJx2fKB1fWB2fZBMzHnZzWzMnAtg1JzMXFuhjVBwLZzq', 'EwvZ', 'BM9Uzq', 'AxnszwDPC3rLCMvKu3LTyM9S', 'lY4V', 'DZe3', 'BwvZC2fNzq', 'qxjYyxK', 'qujdrevgr0HjsKTmtu5puffsu1rvvLDywvPHyMnKzwzNAgLQA2XTBM9WCxjZDhv2D3H5EJaXmJm0nty3odKRlZ0', 'C2v0DgLUz3mUyxbWswqGBxvZDcbIzsbHig5VBI1LBxb0EsbZDhjPBMC', 'Bwf0y2G', 'x19Yzxf1zxn0qwXNB3jPDgHTt25JzsbRzxK6', 'C3bLy2LLCW', 'r2vUzxjHDg9YrNvUy3rPB24', 'AgLKzgvU', 'AgfZt3DUuhjVCgvYDhK', 'y29UzMLNDxjHyMXL', 'v1fFDMSX', 'zg9JDw1LBNq', 'B3DUs2v5CW', 'zgf0ys5Yzxn1BhqGzM9YBwf0igvYCM9YlG', 'C3rYAw5NAwz5igrLDgvJDgLVBG', 'lcbFBg9HzgvKx2nHy2HLCZO', 'CM91BMq', 'x3n0zq', 'xwq/otyW', 'CgfYyw1ZigLZigvTChr5igfMDgvYigv4y2X1zgLUzYaIDw5ZywzLiIbWyxjHBxm', 'BM9Kzq', 'x3n0AW', 'iLX1zgvHzci', 'DgLTzw91Da', 'C3LTyM9SCW', 'Bwv0ywrHDge', 'reDcruziqunjsKS', 'igLZig5VDcbHBIbVyMPLy3q', 'x19Yzxf1zxn0rgvWCYWGx19WyxjZzufSz29YAxrOBsbYzxn1Bhq6', 'y29Uy2f0', 'AgvHza', 'y29TCgXLDgu', 'DZi0', 'y3jLyxrLigLUC3rHBMnLihDPDgGGyxbWswq9', 'tw96AwXSys81lJaGxcGOlIO/kvWP', 'q2fUj3qGy2fSBcbTzxrOB2qGB24G', 'twfSzM9YBwvKifvurI04igrHDge', 'mhGXnG', 'sw5JB3jYzwn0igLUDM9JyxrPB24', 'x19Nzw5tAwDUlcbWyxjHBxntDhi6', 'v1fFzhLFywXNB19Z', 'vgHLig1LDgHVzcbKB2vZBID0igfJy2vWDcbYzwD1BgfYigv4ChjLC3nPB25Z', 'suvFufjpve8', 'AxnqCM90B3r5CgvpzG', 'x19Yzxf1zxn0rgvWCYbZDgfYDc4', 'w251BgXD', 'D2L0Ag91DfnLDhrLCG', 'Bwf0y2HbBgW', 'lcbJAgvJAYbZDg9YywDLigzWoG', 'CMv0DxjUia', 'C3bSAxq', 'jgnKy19HC2rQzMXHC3v0B3bMAhzJwKXTy2zSxW', 'nJbWEcaNtM90igeGCMvHBcbMB250jW', 'iZqYztfHmG', 'DMfSDwu', 'EgLHB3DHBMDZAgvUlMnVBq', 'Bg9JywXFA2v5xW', 'CMv0DxjU', 'BM9YBwfS', 'D2TZ', 'Bwf4', 'xsSK', 'zNvUy3rPB25jza', 'Aw5PDa', 'B2jZzxj2ywjSzq', 'D2HPDgu', 'qxn5BMnhzw5LCMf0B3jgDw5JDgLVBG', 'DMfSDwvZ', 'igLZig5VDcbHihn5BwjVBa', 'x19Yzxf1zxn0rgvWCYbMCM9TignHy2HLlcbLBMqU', 'B3aTC3LTyM9SCW', 'w14/xsO', 'mY4ZnI4X', 'xsLB', 'x19Yzxf1zxn0qwXNB3jPDgHTihjLCxvLC3qGC3vJy2vZCYeSignOzwnRig1LBw9YEsbMCdO', 'CMvWBgfJzufSBa', 'qxjNDw1LBNrZ', 'DxjS', 'sw52ywXPzcb0Aw1LihzHBhvL', 'Aw5KzxHpzG', 'C3rYAw5NAwz5', 'ChaX', 'x19Nzw5tAwDUrgvMyxvSDcWGCgfYyw1Zu3rYoG', 'sLnptG', 'zw50CMLLCW', 'v1fFzhLFDgTFCW', 'C3rYAw5NlxrVlxn5BwjVBc1YzwDPC3rYEq', 'ywXWAgfIzxrPyW', 'DZe5', 'CMv2zxjZzq', 'BgvUz3rO', 'w29IAMvJDca', 'rxzLBNq', 'DxnLig5VCM1HBfrVA2vU', 'ufiGzMXHy2TZihf1AxOGz3LToIbuvIbesIbIB3GGD2HLBJ8G4PIG', 'Bg9Hza', 'x19Yzxf1zxn0qwXNB3jPDgHTihn0yxj0lG', 'zgLZCg9Zzq', 'lcbZAwDUzwrtDhi6', 'AgfZsw5ZDgfUy2u', 'BwfPBI5ZAwDUi19FCMvXDwvZDerLChm', 'u3LTyM9SlG', 'pt09', 'C3rHy2S', 'CxvLDwvnAwnYB3rHC2S', 'DZeX', 'tMf0AxzLignYExb0BYbTB2r1BguGy291BgqGBM90igjLihvZzwqGDg8Gz2v0ihnLy3vYzsbYyw5KB20GBNvTyMvYlG', 'lIO/y2HYB21Llwv4DgvUC2LVBJPCl1WVkc4QpYLClY4QpW', 'jgnOCM9Tzv9HC3LUy1nJCMLWDeLUzM8', 'qMfKifbYB21PC2uGy29UC3rYDwn0B3i', 'DZeY', 'Dg9tDhjPBMC', 'x19Nzw5ezwzHDwX0s2v5igLUChv0pq', 'EJrYzwTSowKXDq', 'D2LUzg93', 'C2nYAxb0', 'zxH0zw5ZAw9UCZO', 'CMDIysGWlcaWlcaYmdaSidaUnsK', 'BgfZDeLUzgv4t2y', 'x19Yzxf1zxn0rgvWCYbYzxf1zxn0ihrVA2vUigzHAwXLzcWGzxjYB3i6ia'];
		var uI = [
    "yxr0CMLIDxrLihzLyZiGyxr0CLzLCNrLEdT2yxj5Aw5NihzLyZiGDMfYEwLUvgv4q29VCMrPBMf0ztT1BMLMB3jTihzLyZiGDw5PzM9YBu9MzNnLDdT2B2LKig1HAw4OkxT2yxj5Aw5uzxHdB29YzgLUyxrLpwf0Dhjwzxj0zxGRDw5PzM9YBu9MzNnLDdTNBf9qB3nPDgLVBJ12zwm0kgf0Dhjwzxj0zxGSmcWXktT9",
    "lgv4ChjLC3m9",
    "q2fUBM90ignVBNzLCNqGysbtEw1IB2WGDMfSDwuGDg8GysbZDhjPBMC",
    "CMvWBgfJzq",
    "ywXWAgfIzxq",
    "CMfUzg9T",
    "x19LC01VzhvSzq",
    "Ahr0Chm6lY9NAxrODwiUy29Tl3PSB2LYB2nRl2nVCMuTANmVyMXVyI92mY4ZnI4Xl0Xjq0vou0u",
    "B25YzwfKExn0yxrLy2HHBMDL",
    "C3LTyM9SlxrVlxn0CMLUzY1YzwDPC3rYEq",
    "DZeW",
    "v1fFz2f0AgvYx3DNBde",
    "C3bSAwnL",
    "ExL5Es1nts1Kza",
    "AwzYyw1L",
    "x19Yzxf1zxn0qwXNB3jPDgHTigvUDKnVBgXLy3q9",
    "ChrFCgLU",
    "D2vIz2W",
    "u3LTyM9Ska",
    "u3LTyM9S",
    "Dw5PzM9YBu9MzNnLDa",
    "DZiW",
    "C2HHBq",
    "ChjLy2LZAw9Uig1LzgL1BxaGzMXVyxq7DMfYEwLUzYb2zwmYihzHCNLPBLrLEenVB3jKAw5HDgu7DM9PzcbTywLUkcKGE2DSx0zYywDdB2XVCJ12zwm0khzHCNLPBLrLEenVB3jKAw5HDguSmcWXktT9",
    "qxjYyxKGsxrLCMf0B3i",
    "yxbWAwq",
    "kf58w14",
    "lgTLEt0",
    "Bg9HzgvYlNv0AwXZi2XVywrsywnty3jPChrpBMnL",
    "nti0ng1Zwej2DG",
    "Bg9HzgvK",
    "svf6ovDt",
    "C3OUAMqUy29T",
    "kf58icK",
    "AdvFzMLSzv92nc45lJC",
    "tM90igvUB3vNAcbHCMD1BwvUDhm",
    "Dg9qCMLTAxrPDMu",
    "DZiY",
    "y2rJx2fKB1fWB2fZBMzHnZzWzMnAtg1JzMXFu3LTyM9S",
    "y29UC3rYDwn0B3i",
    "ChvYzq",
    "Bg9HzcbYywmGANmGC3vJy2vZCYe",
    "C3rYAw5N",
    "uhjVDg90ExbL",
    "x19WCM90B19F",
    "mta0odyYmJrfuvf5vfa",
    "uMvNrxHW",
    "AMf2yq",
    "ig9Mia",
    "Dg9Rzw4GAxmGzw1WDhK",
    "D2vIz2XgCde",
    "CMvQzwn0zwq",
    "WQKGmJaXnc0Ymdi0ierLBMLZifb1C2HRyxjLDIaOEMXVAxjVy2SUCNuP",
    "zNvSzMLSBgvK",
    "rvHux3rLEhr1CMvFzMLSDgvYx2fUAxnVDhjVCgLJ",
    "w25HDgL2zsbJB2rLxq",
    "zxH0zw5K",
    "z2v0t3DUuhjVCgvYDhLoyw1LCW",
    "q29UDgvUDc1uExbL",
    "ue9tva",
    "ExL5Eu1nzgq",
    "nc45",
    "DZe4",
    "ndG3mda1nwPkz2H1AG",
    "CMvXDwvZDcbWyxjHBxmGzxjYB3iU",
    "uhjVBwLZzq",
    "iLX1zgyWnLX1zdGZnci",
    "z2vUzxjHDguGA2v5igzHAwXLza",
    "DZeZ",
    "x19Yzxf1zxn0rgvWCYbLBMqU",
    "mZm4otu4mLjQBhfrEa",
    "DZe1",
    "tM8GB25LihbYB21PC2uGCMvZB2X2zwq",
    "Dg9tDhjPBMDuywC",
    "uMvMBgvJDa",
    "zw51BwvYywjSzq",
    "uhjVBwLZzsbJyw4NDcbIzsbYzxnVBhzLzcbPDhnLBgy",
    "mte3C2PUvu1K",
    "Bg9JywXFA2v5xZm",
    "vw5Oyw5KBgvKihbYB21PC2uGCMvQzwn0Aw9U",
    "ntu1mZq4ofvnEgT0uG",
    "twf4Aw11BsbHBgXVD2vKigLUzgv4igv4y2vLzgvK",
    "q2fUBM90ihnLDcbYzwfKig9UBhKGlMXLBMD0Aa",
    "AxrLCMf0B3i",
    "zg9JDw1LBNqUrJ1pyMPLy3q",
    "v0vcs0Lux0vyvf90zxH0DxjLx2zPBhrLCL9HBMLZB3rYB3bPyW",
    "DZe0",
    "x19TywTLu2LNBIWGCMvZDwX0oG",
    "lcbLpq",
    "DZe2",
    "CMvQzwn0Aw9UAgfUzgXLza",
    "Dw5Oyw5KBgvKCMvQzwn0Aw9U",
    "rxjYB3i",
    "x19JB2XSzwn0igvUDKnVBgXLy3q9",
    "zMLSDgvY",
    "zMLSztO",
    "AxndB25JyxrtChjLywrHyMXL",
    "r0vu",
    "Ahr0Chm6lY9Jywn0DxmUAMqUy29Tl3jLCxvLC3rFywXNBW",
    "y2f1C2u",
    "CMvXDwvZDcbLCNjVCIWG",
    "C2v0",
    "qwnJzxb0",
    "x19JB3jLlwPZx3nOyxjLzf9F",
    "x19Yzxf1zxn0rgvWCYb1C2uGBMv3igzWlcbMCdO",
    "C3LTyM9SigrLDgvJDgLVBG",
    "qwDNCMvNyxrLrxjYB3i",
    "DgHLBG",
    "DMfSDwvpzG",
    "yM9VBgvHBG",
    "ywjJzgvMz2HPAMTSBw5VChfYC3r1DND4ExPbqKneruzhseLks0XntK9quvjtvfvwv1HzwG",
    "tNvSBa",
    "y2rJx2fKB1fWB2fZBMzHnZzWzMnAtg1JzMXFqxjYyxK",
    "mJu3ntC1nhbQA0jiBG",
    "tu9Ax0vyvf90zxH0DxjLx2zPBhrLCL9HBMLZB3rYB3bPyW",
    "CgLU",
    "zgvZy3jPChrPB24",
    "ChjVDg90ExbL",
    "yNuY",
    "Ahr0Chm6lY9ZDg9YywDLlJm2mgj1EwLTzY5JB20VD2vIy29UDgfPBMvYl21HAw4VANmTC2vJDxjPDhKTDJmTCMfJlMPZp3y9",
    "y2fUDMfZmq",
    "lcbMCdO",
    "CMvMzxjLCG",
    "ANnVBG",
    "BMfTzq",
    "zw52q29SBgvJDa",
    "CgfYyw1ZigLZigvTChr5",
    "uhjVBwLZzs1JAgfPBIbJEwnSzq",
    "lcb0B2TLBJO",
    "Aw5JBhvKzxm",
    "mtqXndC0ounxBgnQva",
    "lcbZDg9YywDLrNa6",
    "C3vJy2vZCW",
    "mdm4ns0WnY0YnvqWnZOWnJOZos45otLA",
    "tNvTyMvY",
    "D2vIz2XgCa",
    "mdeYmZq1nJC4oq",
    "q2fUBM90igrLBgv0zsbWCM9Wzxj0Esa",
    "q2fUj3qGC2v0ia",
    "zgvMyxvSDa",
    "Dgv4Dc9QyxzHC2nYAxb0",
    "BwfW",
    "Dw5Zy29WywjSzxm",
    "igLZig5VDcbHignVBNn0CNvJDg9Y",
    "yxn5BMneAxnWB3nL",
    "u3rYAw5N",
    "rNvUy3rPB24",
    "Dw5RBM93BIbLCNjVCG",
    "y2rJx2fKB1fWB2fZBMzHnZzWzMnAtg1JzMXFuhjVBwLZzq",
    "EwvZ",
    "BM9Uzq",
    "AxnszwDPC3rLCMvKu3LTyM9S",
    "lY4V",
    "DZe3",
    "BwvZC2fNzq",
    "qxjYyxK",
    "qujdrevgr0HjsKTmtu5puffsu1rvvLDywvPHyMnKzwzNAgLQA2XTBM9WCxjZDhv2D3H5EJaXmJm0nty3odKRlZ0",
    "C2v0DgLUz3mUyxbWswqGBxvZDcbIzsbHig5VBI1LBxb0EsbZDhjPBMC",
    "Bwf0y2G",
    "x19Yzxf1zxn0qwXNB3jPDgHTt25JzsbRzxK6",
    "C3bLy2LLCW",
    "r2vUzxjHDg9YrNvUy3rPB24",
    "AgLKzgvU",
    "AgfZt3DUuhjVCgvYDhK",
    "y29UzMLNDxjHyMXL",
    "v1fFDMSX",
    "zg9JDw1LBNq",
    "B3DUs2v5CW",
    "zgf0ys5Yzxn1BhqGzM9YBwf0igvYCM9YlG",
    "C3rYAw5NAwz5igrLDgvJDgLVBG",
    "lcbFBg9HzgvKx2nHy2HLCZO",
    "CM91BMq",
    "x3n0zq",
    "xwq/otyW",
    "CgfYyw1ZigLZigvTChr5igfMDgvYigv4y2X1zgLUzYaIDw5ZywzLiIbWyxjHBxm",
    "BM9Kzq",
    "x3n0AW",
    "iLX1zgvHzci",
    "DgLTzw91Da",
    "C3LTyM9SCW",
    "Bwv0ywrHDge",
    "reDcruziqunjsKS",
    "igLZig5VDcbHBIbVyMPLy3q",
    "x19Yzxf1zxn0rgvWCYWGx19WyxjZzufSz29YAxrOBsbYzxn1Bhq6",
    "y29Uy2f0",
    "AgvHza",
    "y29TCgXLDgu",
    "DZi0",
    "y3jLyxrLigLUC3rHBMnLihDPDgGGyxbWswq9",
    "tw96AwXSys81lJaGxcGOlIO/kvWP",
    "q2fUj3qGy2fSBcbTzxrOB2qGB24G",
    "twfSzM9YBwvKifvurI04igrHDge",
    "mhGXnG",
    "sw5JB3jYzwn0igLUDM9JyxrPB24",
    "x19Nzw5tAwDUlcbWyxjHBxntDhi6",
    "v1fFzhLFywXNB19Z",
    "vgHLig1LDgHVzcbKB2vZBID0igfJy2vWDcbYzwD1BgfYigv4ChjLC3nPB25Z",
    "suvFufjpve8",
    "AxnqCM90B3r5CgvpzG",
    "x19Yzxf1zxn0rgvWCYbZDgfYDc4",
    "w251BgXD",
    "D2L0Ag91DfnLDhrLCG",
    "Bwf0y2HbBgW",
    "lcbJAgvJAYbZDg9YywDLigzWoG",
    "CMv0DxjUia",
    "C3bSAxq",
    "jgnKy19HC2rQzMXHC3v0B3bMAhzJwKXTy2zSxW",
    "nJbWEcaNtM90igeGCMvHBcbMB250jW",
    "iZqYztfHmG",
    "DMfSDwu",
    "EgLHB3DHBMDZAgvUlMnVBq",
    "Bg9JywXFA2v5xW",
    "CMv0DxjU",
    "BM9YBwfS",
    "D2TZ",
    "Bwf4",
    "xsSK",
    "zNvUy3rPB25jza",
    "Aw5PDa",
    "B2jZzxj2ywjSzq",
    "D2HPDgu",
    "qxn5BMnhzw5LCMf0B3jgDw5JDgLVBG",
    "DMfSDwvZ",
    "igLZig5VDcbHihn5BwjVBa",
    "x19Yzxf1zxn0rgvWCYbMCM9TignHy2HLlcbLBMqU",
    "B3aTC3LTyM9SCW",
    "w14/xsO",
    "mY4ZnI4X",
    "xsLB",
    "x19Yzxf1zxn0qwXNB3jPDgHTihjLCxvLC3qGC3vJy2vZCYeSignOzwnRig1LBw9YEsbMCdO",
    "CMvWBgfJzufSBa",
    "qxjNDw1LBNrZ",
    "DxjS",
    "sw52ywXPzcb0Aw1LihzHBhvL",
    "Aw5KzxHpzG",
    "C3rYAw5NAwz5",
    "ChaX",
    "x19Nzw5tAwDUrgvMyxvSDcWGCgfYyw1Zu3rYoG",
    "sLnptG",
    "zw50CMLLCW",
    "v1fFzhLFDgTFCW",
    "C3rYAw5NlxrVlxn5BwjVBc1YzwDPC3rYEq",
    "ywXWAgfIzxrPyW",
    "DZe5",
    "CMv2zxjZzq",
    "BgvUz3rO",
    "w29IAMvJDca",
    "rxzLBNq",
    "DxnLig5VCM1HBfrVA2vU",
    "ufiGzMXHy2TZihf1AxOGz3LToIbuvIbesIbIB3GGD2HLBJ8G4PIG",
    "Bg9Hza",
    "x19Yzxf1zxn0qwXNB3jPDgHTihn0yxj0lG",
    "zgLZCg9Zzq",
    "lcbZAwDUzwrtDhi6",
    "AgfZsw5ZDgfUy2u",
    "BwfPBI5ZAwDUi19FCMvXDwvZDerLChm",
    "u3LTyM9SlG",
    "pt09",
    "C3rHy2S",
    "CxvLDwvnAwnYB3rHC2S",
    "DZeX",
    "tMf0AxzLignYExb0BYbTB2r1BguGy291BgqGBM90igjLihvZzwqGDg8Gz2v0ihnLy3vYzsbYyw5KB20GBNvTyMvYlG",
    "lIO/y2HYB21Llwv4DgvUC2LVBJPCl1WVkc4QpYLClY4QpW",
    "jgnOCM9Tzv9HC3LUy1nJCMLWDeLUzM8",
    "qMfKifbYB21PC2uGy29UC3rYDwn0B3i",
    "DZeY",
    "Dg9tDhjPBMC",
    "x19Nzw5ezwzHDwX0s2v5igLUChv0pq",
    "EJrYzwTSowKXDq",
    "D2LUzg93",
    "C2nYAxb0",
    "zxH0zw5ZAw9UCZO",
    "CMDIysGWlcaWlcaYmdaSidaUnsK",
    "BgfZDeLUzgv4t2y",
    "x19Yzxf1zxn0rgvWCYbYzxf1zxn0ihrVA2vUigzHAwXLzcWGzxjYB3i6ia",
    "v1fFz2f0AgvYx2n2mq",
    "mc4XlJC",
    "yxbWBgLJyxrPB24VEc13D3CTzM9YBs11CMXLBMnVzgvK",
    "D2vI",
    "yxr0CLzLCNrLEa",
    "A2v5CW",
    "ENHJyxnK",
    "lcbHBgDVoG",
    "v0vcr0XFzgvIDwDFCMvUzgvYzxjFAw5MBW",
    "ChjVCgvYDhLjC0vUDw1LCMfIBgu",
    "Dw5Oyw5KBgvKuMvQzwn0Aw9U",
    "u3LTyM9SigLZig5VDcbHignVBNn0CNvJDg9Y",
    "BwfPBI5ZAwDUi19Fzgv0zwn0Aw5N",
    "D3v2oG",
    "ihrVA2vUoG",
    "CMvXDwvZDcb0B2TLBIbMywLSzwqGA2v5oG",
    "BMv4Da",
    "BNvTyMvY",
    "Dg9ju09tDhjPBMC",
    "AxnxzwXSs25VD25tEw1IB2W",
    "kd86psHBxJTDkIKPpYG7FcqP",
    "C29TzxrOAw5N",
    "w29IAMvJDcbpyMPLy3rD",
    "zxjYB3jZ",
    "mtuUnhb4icDbCMLHBcC",
    "ChjVy2vZCW",
    "w3nPz25Dia",
    "yxbWBgLJyxrPB24VANnVBG",
    "uNzjpdD8tKmTmwC1",
    "qebPDgvYyxrVCG",
    "C2vHCMnO",
    "AhrTBgzPBgu",
    "CgfYC2vYzxjYB3i",
    "C3rHDgu",
    "zxHWzxjPBwvUDgfSlxDLyMDS",
    "iZfHm2jJmq",
    "CgfYyw1ZigLZig5VDcbHihbSywLUig9IAMvJDa",
    "t2jQzwn0",
    "v3jVBMCGBNvTyMvYig9MihjLCgv0AxrPB25Z",
    "DZi1",
    "DZiZ",
    "ChDKDf9Pza",
    "Ahr0Chm6lY9NAxrODwiUy29Tl3PSB2LYB2nRl2nVCMuTANm",
    "cqOlda0GWQdHMOdIGidIGihIGilIGipIGitIGixIGiBIGiFIGiJIGiNIGiRIGk/IGz/JGidIGkJIGkNVU78",
    "mdaW",
    "CNfWB25TBgTQAwHNzMvKy2jHwLLyv1zvvfnsuvbptK1ms0PjseDgrurdqKeTxZK4nZy1ndmYmtb6ExH3DNv0CW",
    "v2LUzg93",
    "zg9JDw1LBNrfBgvTzw50",
    "C3vH",
    "DgHYB3C",
    "CMv0DxjUihrOAxm",
    "u3rYAw5NieL0zxjHDg9Y",
    "ExL5Eu1nzgrOAg1TC3ntu1m",
    "D3jPDgfIBgu",
    "C29YDa",
    "zgL2",
    "qxn5BMngDw5JDgLVBG",
    "B2jQzwn0",
    "q2fUj3qGy29UDMvYDcbVyMPLy3qGDg8GChjPBwL0AxzLihzHBhvL",
    "DZiX",
    "CgfYyw1ZignVBNrHAw5ZihjLC2vYDMvKihbHCMfTig5HBwuU",
    "z2v0",
    "lcbYzxrYEsbUzxH0ihrPBwuU",
    "y2fUDMfZ",
    "C3LTyM9S",
    "C2LNBIbLBgfWC2vKihrPBwuH",
    "ieL0zxjHDg9Y",
    "yxn5BMnjDgvYyxrVCG",
    "w29IAMvJDcb6xq",
    "yNuX",
    "t2jQzwn0igfSCMvHzhKGAw5PDgLHBgL6zwq",
    "x19Yzxf1zxn0rgvWCYb1C2uGy2fJAguGzNaSigzWoG",
    "qwnJzxnZB3jZig5VDcbZDxbWB3j0zwq",
    "z2v0vg9Rzw5F",
    "CMvQzwn0Aw9UsgfUzgXLza",
    "zgLHBNrVDxnOAs5JB20",
    "Dg9mB2nHBgvtDhjPBMC",
    "Bwf0y2HLCG",
    "y29UC3rYDwn0",
    "D3vYoG",
    "mdeYmZq1nJC4owfIy2rLzMDOAwPRBg1UB3bXCNn0Dxz3EhL6qujdrevgr0HjsKTmtu5puffsu1rvvLDywvPFlq",
    "igLZig5VDcbHigz1BMn0Aw9U",
    "igLZig5VDcbPDgvYywjSzq",
    "tM/PQPC",
    "Cgf0DgvYBK1HDgnO",
    "y2nU",
    "rgf0zq",
    "sw5JB21WyxrPyMXLihjLy2vPDMvYlca",
    "Chb6Ac5Qzc5JB20",
    "C2XPy2u",
    "igfZigeGChjVDg90ExbL",
    "AdvZDa",
    "DgvZDcbLCNi",
    "CMvK",
    "Bg9HzcbYywmGANmGzMfPBce",
    "Bwv0ywrHDgflzxK"
];
        a060c66O = function() {
            return uI;
        }
        ;
        return a060c66O();
    }
    function a060c66n(_$O, _$n) {
        var _$p = a060c66O();
        return a060c66n = function(_$s, _$F) {
            _$s = _$s - (-0x533 * -0x3 + -0x16 * 0xbf + 0x18a);
            var _$V = _$p[_$s];
            if (a060c66n.gdEjwl === undefined) {
                var _$A = function(_$X) {
                    var _$z = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+/=';
                    var _$S = ''
                      , _$Z = '';
                    for (var _$L = 0x1 * -0x1378 + 0x1 * 0xfca + 0x13a * 0x3, _$Y, _$m, _$J = 0x82b + 0x693 + 0x66 * -0x25; _$m = _$X.charAt(_$J++); ~_$m && (_$Y = _$L % (0x986 + -0x239 * -0xc + -0x242e) ? _$Y * (-0x1b * 0xd + -0x5e * 0x11 + 0x7dd) + _$m : _$m,
                    _$L++ % (0x1f77 + -0x1 * -0x1532 + -0x1 * 0x34a5)) ? _$S += String.fromCharCode(0x1e88 + -0x2141 * -0x1 + -0x1f65 * 0x2 & _$Y >> (-(0x196d + -0x4 * -0x2b1 + 0x1 * -0x242f) * _$L & -0x6e * 0x7 + 0x1085 * -0x1 + 0x23 * 0x8f)) : 0x210 + -0x13 * -0xe7 + -0x1335) {
                        _$m = _$z.indexOf(_$m);
                    }
                    for (var _$o = -0x1 * 0x1ebc + -0x219d + 0x3c9 * 0x11, _$K = _$S.length; _$o < _$K; _$o++) {
                        _$Z += '%' + ('00' + _$S.charCodeAt(_$o).toString(-0x35b + -0x58b * -0x2 + -0x7ab)).slice(-(-0x1f87 * -0x1 + 0x16da + -0x365f));
                    }
                    return decodeURIComponent(_$Z);
                };
                a060c66n.ZMTEpS = _$A,
                _$O = arguments,
                a060c66n.gdEjwl = !![];
            }
            var _$P = _$p[0xb1 + 0x7 * 0x439 + -0x40 * 0x79].substring(-0x649 * 0x4 + -0x1051 + 0x1 * 0x2975, 0x2666 + 0x1 * -0x182f + -0xe35 * 0x1)
              , _$I = _$s + _$P
              , _$B = _$O[_$I];
            return !_$B ? (_$V = a060c66n.ZMTEpS(_$V),
            _$O[_$I] = _$V) : _$V = _$B,
            _$V;
        }
        ,
        a060c66n(_$O, _$n);
    }
	`
)

func t1() {
	var err error

	var funcI3 func(ctx context.Context, exprs ...ast.Expression)
	var funcI1 func(ctx context.Context, stmts ...ast.Statement)
	var funcI2 func(ctx context.Context, bindings ...*ast.Binding)
	var funcV1 func(ctx context.Context, varDecls ...*ast.VariableDeclaration)

	type _context struct {
		pre     string
		replace map[string]*ast.Identifier
	}
	funcI3 = func(ctx context.Context, exprs ...ast.Expression) {
		_ctx := ctx.Value("pre").(*_context)
		pre := _ctx.pre

		for _, expr := range exprs {
			switch expr := expr.(type) {
			case *ast.CallExpression:
				// fmt.Println(expr.Callee)
				funcI3(ctx, expr.Callee)
				funcI3(ctx, expr.ArgumentList...)
			case *ast.FunctionLiteral:
				name := "_anon_"
				if expr.Name != nil {
					name = expr.Name.Name.String()
				}
				pre += "[" + name + "]"
				// fmt.Println(pre, "[call]", expr.Name)
				funcI1(ctx, expr.Body)
				funcV1(ctx, expr.DeclarationList...)
			}
		}

		_ctx.pre = pre
	}

	funcI2 = func(ctx context.Context, bindings ...*ast.Binding) {
		_ctx := ctx.Value("pre").(*_context)
		pre := _ctx.pre

		for _, v := range bindings {
			_newCtx := *_ctx

			_newCtx.replace = make(map[string]*ast.Identifier)
			for k, v := range _ctx.replace {
				_newCtx.replace[k] = v
			}

			switch init := v.Initializer.(type) {
			case *ast.ArrayLiteral,
				*ast.DotExpression,
				*ast.ConditionalExpression,
				*ast.FunctionLiteral,
				*ast.UnaryExpression,
				*ast.BinaryExpression,
				*ast.ObjectLiteral,
				*ast.AssignExpression,
				*ast.RegExpLiteral,
				*ast.CallExpression:
				funcI3(ctx, init)
			case *ast.Identifier:
				funcI3(ctx, init)
				new := init.Name.String()
				old := v.Target.(*ast.Identifier).Name.String()

				_ctx.replace[old] = init
				fmt.Printf("%v %v %v = %v\n", pre, "[hit]", old, new)
			default:
				target := v.Target.(*ast.Identifier)

				pre += "." + target.Name.String()
				if v.Initializer == nil {
					fmt.Println(pre, "[binding]", target.Name.String(), "nil")
				} else {
					fmt.Println(pre, "[binding]", target.Name.String(), reflect.TypeOf(v.Initializer).String())
				}
			}

			newCtx := context.WithValue(ctx, "pre", &_newCtx)

			funcI3(newCtx, v.Initializer)
		}

		_ctx.pre = pre
	}

	funcI1 = func(ctx context.Context, stmts ...ast.Statement) {
		// _ctx := ctx.Value("pre").(*_context)

		for _, stmt := range stmts {
			switch stmt := stmt.(type) {
			case *ast.VariableStatement:
				funcI2(ctx, stmt.List...)
			case *ast.BlockStatement:
			}
		}

	}

	funcV1 = func(ctx context.Context, varDecls ...*ast.VariableDeclaration) {
		// _ctx := ctx.Value("pre").(*_context)

		for _, varDecl := range varDecls {
			funcI2(ctx, varDecl.List...)
		}
	}

	jsScript2, err := os.ReadFile("./js_security_v3_0.1.5.js_de")
	if err != nil {
		panic(err)
	}

	porj, err := goja.Parse("./js_security_v3_0.1.5.js", string(jsScript2))
	if err != nil {
		panic(err)
	}
	// ctx := context.WithValue(context.Background(), "pre", &_context{
	// 	pre:     "",
	// 	replace: make(map[string]*ast.Identifier),
	// })
	// funcI1(ctx, porj.Body...)
	// funcV1(ctx, porj.DeclarationList...)

	// projJSON, err := json.MarshalIndent(porj, "", "\t")
	// if err != nil {
	// 	panic(err)
	// }
	// os.WriteFile("./js_security_v3_0.1.5.js.ast", projJSON, 0644)
	fmt.Println(porj)
	return

	// [{0 1 98***15 只要***一点 2024-12-08T08:43:17Z 1970-01-01T00:00:00Z 2 0}] <nil>

	err = decodeJS("./js_security_v3_0.1.5.js", descript, func(vm *goja.Runtime, body []byte) []byte {
		body = evalJSAlias(vm, `_4u94f\("\S+"\)`, "", body)
		body = evalJSAlias(vm, `a060c66n\(0x[0-9abcdef]+\)`, "", body)
		body = evalJSAlias(vm, `([a-zA-Z][a-zA-Z0-9])\(0x[0-9abcdef]+\)`, "a060c66n", body)

		body = evalJS(vm, body, `([\-+]?0x[0-9abcdef]+[ \t]*[\+\-\*]?[ \t]*)+[\-+]?0x[0-9abcdef]+`, func(matchs [][]byte) string {
			js := fmt.Sprintf("eval(%s)", string(matchs[0]))

			return js
		})
		bodyStr := string(body)
		bodyStr = strings.ReplaceAll(bodyStr, "_$mv", "__log")
		// bodyStr = strings.ReplaceAll(bodyStr, "t.call(r, e[n], n, e) === {})", "false)")
		// bodyStr = strings.ReplaceAll(bodyStr, `t["call"](r, e[i], i, e) === {})`, "false)")
		// bodyStr = strings.ReplaceAll(bodyStr, "_0x201a", "__tostr_e")
		// bodyStr = strings.ReplaceAll(bodyStr, "_0x2e0a6d", "__tostr_s")
		// bodyStr = strings.ReplaceAll(bodyStr, "_0x5f424c", "__get_jstoken")
		// bodyStr = strings.ReplaceAll(bodyStr, "_0x370d2c", "__tokenStruct")
		// bodyStr = strings.ReplaceAll(bodyStr, "_0x29c61f", "__execCallback")
		// bodyStr = strings.ReplaceAll(bodyStr, "_0x13997a", "__tkKeys")
		// bodyStr = strings.ReplaceAll(bodyStr, "_0x1eda92", "__tkKeyMode")
		// bodyStr = strings.ReplaceAll(bodyStr, "_0x292020", "__localStorage")
		body = []byte(bodyStr)

		replaces := map[string]string{}
		// body = replaceRegexp(body, `(_\$[a-zA-Z][a-zA-Z0-9])\s*=\s*([_a-zA-Z0-9\.]+)([ \t]*[\n,])`, func(matchs [][]byte) []byte {
		// 	output := "{}" + string(matchs[3])
		// 	replaces[string(matchs[1])] = string(matchs[2])

		// 	return []byte(output)
		// })

		for old, new := range replaces {
			for _, v := range []string{`$`, `(`, `)`, `[`, `]`, `.`, `+`, `-`} {
				old = strings.ReplaceAll(old, v, `\`+v)
			}
			body = replaceRegexp(body, fmt.Sprintf(`%s([\[.\(])`, old), func(matchs [][]byte) []byte {
				output := new
				output += string(matchs[1])

				return []byte(output)
			})

		}

		return []byte(bodyStr)
	})
	if err != nil {
		panic(err)
	}
}

// 定义 download 方法
func download(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("Error: %s", err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error: %s", err.Error())
	}

	return string(body)
}
func main() {
	var err error

	go loop()
	go purchaseLoop()

	dbd, err = NewDBD()
	if err != nil {
		panic(err)
	}
	cookie, err := os.ReadFile("./cookie.txt")
	if err != nil {
		panic(err)
	}
	dbd.SetCookie(string(cookie))

	jsScript2, err := os.ReadFile("./js_security_v3_0.1.5.js_de")
	if err != nil {
		panic(err)
	}

	t1()
	return

	vm := goja.New()
	// 创建一个缓冲区来存储 console.log 的输出
	var outputBuffer bytes.Buffer
	// 定义一个自定义的 console 对象
	console := map[string]func(call goja.FunctionCall) goja.Value{
		"log": func(call goja.FunctionCall) goja.Value {
			// 将输出写入到缓冲区
			for _, arg := range call.Arguments {
				outputBuffer.WriteString(arg.String() + " ")
			}
			outputBuffer.WriteString("\n")
			return goja.Undefined()
		},
	}
	// 将 console 对象注入到虚拟机中
	vm.Set("console", console)
	// 将 download 方法注入到虚拟机
	vm.Set("download", download)

	_, err = vm.RunString(string(`
//模拟旧的 RegExp.$1 等全局属性行为来实现兼容
	(function () {
  // 保存原始的 RegExp.prototype.exec 方法
  const originalExec = RegExp.prototype.exec;

  // 定义兼容层，拦截 exec 调用
  RegExp.prototype.exec = function (str) {
    const result = originalExec.call(this, str); // 调用原始方法

    if (result) {
      // 更新 RegExp 全局属性
      for (let i = 1; i <= 9; i++) {
        RegExp[` + "`" + `$${i}` + "`" + `] = result[i] || ""; // 设置 $1, $2, ..., $9
      }
    } else {
      // 如果没有匹配，则清空全局属性
      for (let i = 1; i <= 9; i++) {
        RegExp[` + "`" + `$${i}` + "`" + `] = "";
      }
    }

    return result;
  };
})();


window = {document:{querySelector:null}};navigator = {userAgent:"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"}`))
	if err != nil {
		fmt.Println(outputBuffer.String())
		panic(err)
	}
	_, err = vm.RunString(string(jsScript2))
	if err != nil {
		fmt.Println(outputBuffer.String())
		panic(err)
	}

	_, err = vm.RunString(string(`
	function getJsToken() {
	        var O = new window.ParamsSign({
    "appId": "/detail/v2",
    "debug": false,
    "preRequest": false
});

O._debug = true

    	var k = {
    "functionId": "paipai.auction.current_bid_info",
    "t": 1734086442007,
    "appid": "paipai_h5",
    "body": "a090a65d64307061e1bd2c14979c0e2205a6f7b166a8e5762a7257f54fdd3070"
}
		return O.sign(k);
	}
	
	`))
	if err != nil {
		panic(err)
	}

	var getJsToken func(userAgent, url, bizId, eid string) goja.Promise
	err = vm.ExportTo(vm.Get("getJsToken"), &getJsToken)
	if err != nil {
		fmt.Println("Js函数映射到 Go 函数失败！")
		panic(err)
	}

	v := getJsToken(
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"https://paipai.jd.com/auction-detail/393024207",
		"paipai_sale_pc",
		"",
	)
	v.Result()
	v2 := v.Result().ToObject(vm)

	fmt.Println(outputBuffer.String())
	fmt.Println(v2.ToObject(vm).Export())
	// fmt.Println(dbd.AuctionDetail(393229830))
	// fmt.Println(dbd.AuctionCurrentPrice(393229882))
	// fmt.Println(dbd.AuctionPriceRecords(393229882))
	// fmt.Println(dbd.AuctionPriceInfo(393229882))
	return

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
			Id        int     `json:"id"`
			Price     float64 `json:"price"`
			PriceType float64 `json:"type"`
			Address   string  `json:"address"`
			Status    string  `json:"status"`
		}{}
		// body2, _ := io.ReadAll(c.Request.Body)
		// fmt.Println(string(body2))
		if err := c.Bind(&body); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		detail, err := dbd.AuctionDetail(body.Id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err))
			return
		}

		auction := &Auction{
			Id:      body.Id,
			Price:   0,
			Detail:  *detail,
			EndTime: detail.AuctionInfo.ActualEndTime,
			Address: body.Address,
		}
		switch body.PriceType {
		case 1: // 历史均价
			auction.Price = detail.HistoryPriceAve
		case 2: // 历史最低价
			auction.Price = detail.HistoryPriceMin
		case 3: // 历史最高价
			auction.Price = detail.HistoryPriceMax
		case 4: // 自定义出价
			auction.Price = body.Price
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(fmt.Errorf("不支持的出价方式 %v", body.PriceType)))
			return
		}

		auctions[body.Id] = auction
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
