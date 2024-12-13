package main

import "time"

type UnixTime int64

func (ut UnixTime) Time() time.Time {
	return time.Unix(int64(ut/1000), int64(ut%1000))
}
func (ut UnixTime) String() string {
	return ut.Time().Format("2006-01-02T15:04:05.999Z07:00")
}
func (ut UnixTime) MarshalText() (text []byte, err error) {
	return []byte(ut.Time().Format("2006-01-02T15:04:05.999Z07:00")), nil
}

type BidderStatus int

func (bs BidderStatus) String() string {
	switch bs {
	case 1:
		return "即将开始"
	case 2:
		return "正在进行"
	case 3:
		return "已经结束"
	default:
		return "未知状态"
	}
}
func (bs BidderStatus) MarshalText() (text []byte, err error) {
	return []byte(bs.String()), nil
}

type DBDProductInfo struct {
	Id                    int          `json:"id,omitempty"`             // ID
	Status                BidderStatus `json:"status,omitempty"`         // 状态 1 即将开始；2 正在进行
	Quality               string       `json:"quality,omitempty"`        // 质量
	PrimaryPic            string       `json:"primaryPic,omitempty"`     // 封面
	ProductName           string       `json:"productName,omitempty"`    // 产品名称
	StartTime             UnixTime     `json:"startTime,omitempty"`      // 开始时间
	EndTime               UnixTime     `json:"endTime,omitempty"`        // 结束时间
	StartPrice            float64      `json:"startPrice,omitempty"`     // 起步价
	CappedPrice           float64      `json:"cappedPrice,omitempty"`    // 封顶价
	CurrentPrice          float64      `json:"currentPrice,omitempty"`   // 当前价
	CurrentBidder         string       `json:"currentBidder,omitempty"`  // 当前竞拍打码账号
	CurrentBidderNickName string       `json:"bidderNickName,omitempty"` // 当前竞拍打码昵称
	SpectatorCount        int          `json:"spectatorCount,omitempty"` // 围观数
}

type DBDProductBidder struct {
	Id             int          `json:"auctionId,omitempty"`      // ID
	Status         BidderStatus `json:"status,omitempty"`         // 状态 3 结束
	CurrentPrice   float64      `json:"currentPrice,omitempty"`   // 当前价格
	BidderImage    string       `json:"bidderImage,omitempty"`    // 当前竞拍用户头像
	CurrentBidder  string       `json:"currentBidder,omitempty"`  // 当前竞拍打码账号
	BidderNickName string       `json:"bidderNickName,omitempty"` // 当前竞拍打码昵称
	ActualEndTime  UnixTime     `json:"actualEndTime,omitempty"`  // 实际结束时间
	SpectatorCount int          `json:"spectatorCount,omitempty"` // 观众数
	// DelayCount        int64        `json:"delayCount,omitempty"`        // 不知道是啥
	// VirtualDelayCount UnixTime     `json:"virtualDelayCount,omitempty"` // 不知道是啥
}

type DBDAuctionDetail struct {
	AuctionInfo struct {
		Id             int      `json:"id,omitempty"`             // ID
		ProductName    string   `json:"productName,omitempty"`    // 产品名称
		CurrentPrice   float64  `json:"currentPrice,omitempty"`   // 当前价格
		CurrentBidder  string   `json:"bidder,omitempty"`         // 当前竞拍打码账号
		BidderNickName string   `json:"bidderNickName,omitempty"` // 当前竞拍打码昵称
		ActualEndTime  UnixTime `json:"actualEndTime,omitempty"`  // 实际结束时间
		Creater        string   `json:"creater,omitempty"`        // 竞拍创建人
		Created        UnixTime `json:"created,omitempty"`        // 竞拍创建时间
		Modified       UnixTime `json:"modified,omitempty"`       // 竞拍修改时间
	} `json:"auctionInfo,omitempty"` // 拍卖信息
	FreightArea     string `json:"freightArea,omitempty"`     // 收货地址内部编码
	FreightAreaText string `json:"freightAreaText,omitempty"` // 收货地址

	HistoryPriceAve float64 `json:"historyPriceAve,omitempty"` // 平均成交价格 - 原始返回不存在，需要自己算
	HistoryPriceMax float64 `json:"historyPriceMax,omitempty"` // 最高成交价格
	HistoryPriceMin float64 `json:"historyPriceMin,omitempty"` // 最低成交价格
	HistoryRecord   []struct {
		UserNickname string   `json:"userNickname,omitempty"` // 用户昵称
		UserImage    string   `json:"userImage,omitempty"`    // 用户头像
		EndTime      UnixTime `json:"endTime,omitempty"`      // 拍卖结束时间
		OfferPrice   float64  `json:"offerPrice,omitempty"`   // 最终拍卖价格
	} `json:"historyRecord,omitempty"` // 历史成交记录

	IsAreaLimit bool `json:"isAreaLimit,omitempty"` // 是否存在区域限制
	// AreaLimitDetail string `json:"areaLimitDetail,omitempty"` // 区域限制详情，不知道内容格式

	SpectatorCount int `json:"spectatorCount,omitempty"` // 围观数

	HasStock        bool   `json:"hasStock,omitempty"`        // 不知道
	HasAuctionStock bool   `json:"hasAuctionStock,omitempty"` // 不知道
	StockCheckArea  string `json:"stockCheckArea,omitempty"`  // 不知道干啥用的，值好像是这个 freightArea
}

type DBDAuctionAddress struct {
	CurrentPrice    float64 `json:"currentPrice,omitempty"`    // 当前价格
	FreightArea     string  `json:"freightArea,omitempty"`     // 收货地址内部编码
	StockCheckArea  string  `json:"stockCheckArea,omitempty"`  // 不知道干啥用的，值好像是这个 freightArea
	FreightAreaText string  `json:"freightAreaText,omitempty"` // 收货地址

	HasStock        bool `json:"hasStock,omitempty"`        // 不知道
	HasAuctionStock bool `json:"hasAuctionStock,omitempty"` // 不知道

	IsAreaLimit bool `json:"isAreaLimit,omitempty"` // 是否存在区域限制
	// AreaLimitDetail string `json:"areaLimitDetail,omitempty"` // 区域限制详情，不知道内容格式
}

type DBDAuctionPriceRecord struct {
	Id           int      `json:"auctionId,omitempty"`    // ID = null
	Status       int      `json:"status,omitempty"`       // 状态 1 = 当前价格，4 = 价格过期
	UserName     string   `json:"userName,omitempty"`     // 当前竞拍打码账号
	UserNickname string   `json:"userNickname,omitempty"` // 当前竞拍打码昵称
	Created      UnixTime `json:"created,omitempty"`      // 出价创建时间
	Modified     UnixTime `json:"modified,omitempty"`     // 出价修改时间

	OfferPrice   float64 `json:"offerPrice,omitempty"`   // 价格
	CurrentPrice float64 `json:"currentPrice,omitempty"` // 当前价格

}

type DBDAuctionCurrentPrice struct {
	Id             int          `json:"auctionId,omitempty"`      // ID
	Status         BidderStatus `json:"status,omitempty"`         // 状态 2 正在进行 3 结束
	CurrentPrice   float64      `json:"currentPrice,omitempty"`   // 当前价格
	CurrentBidder  string       `json:"currentBidder,omitempty"`  // 当前竞拍打码账号
	BidderImage    string       `json:"bidderImage,omitempty"`    // 当前竞拍用户头像
	BidderNickName string       `json:"bidderNickName,omitempty"` // 当前竞拍打码昵称
	ActualEndTime  UnixTime     `json:"actualEndTime,omitempty"`  // 实际结束时间
	SpectatorCount int          `json:"spectatorCount,omitempty"` // 观众数
	// OfferPrice        float64      `json:"offerPrice,omitempty"`        // 不知道是啥
	// DelayCount        int64        `json:"delayCount,omitempty"`        // 不知道是啥
	// VirtualDelayCount UnixTime     `json:"virtualDelayCount,omitempty"` // 不知道是啥

}
