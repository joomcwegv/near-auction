package main

import (
	"math/big"

	"github.com/vlmoon99/near-sdk-go/env"
	"github.com/vlmoon99/near-sdk-go/types"
)

// @contract:state
type Contract struct {
	Auctioneer    string            `json:"auctioneer"`
	EndTime       uint64            `json:"end_time"` // block time in ms
	HighestBid    string            `json:"highest_bid"` // yoctoNEAR as string
	HighestBidder string            `json:"highest_bidder"`
	Active        bool              `json:"active"`
	Bids          map[string]string `json:"bids"` // bidder -> total bid as string
}

// @contract:init
func (c *Contract) Init(auctioneer string, end_time uint64) {
	if auctioneer == "" {
		env.PanicStr("Auctioneer must be set")
	}
	c.Auctioneer = auctioneer
	c.EndTime = end_time
	c.HighestBid = "0"
	c.HighestBidder = ""
	c.Active = true
	c.Bids = make(map[string]string)
	env.LogString("Auction contract initialized!")
}

// @contract:view
func (c *Contract) GetInfo() Contract {
	return *c
}

// @contract:payable
// @contract:mutating
func (c *Contract) PlaceBid() {
	if !c.Active {
		env.PanicStr("Auction not active")
	}
	
	now := env.GetBlockTimeMs()
	if now > c.EndTime {
		env.PanicStr("Auction ended")
	}

	deposit, err := env.GetAttachedDeposit()
	if err != nil {
		env.PanicStr("Failed to get attached deposit: " + err.Error())
	}
	
	zero, _ := types.U128FromString("0")
	if deposit.Cmp(zero) <= 0 {
		env.PanicStr("Bid must be greater than 0")
	}

	sender, err := env.GetPredecessorAccountID()
	if err != nil {
		env.PanicStr("Failed to get predecessor account: " + err.Error())
	}

	prevBidStr := c.Bids[sender]
	if prevBidStr == "" {
		prevBidStr = "0"
	}
	
	prevBid, ok := new(big.Int).SetString(prevBidStr, 10)
	if !ok {
		prevBid = big.NewInt(0)
	}
	
	depInt, ok := new(big.Int).SetString(deposit.String(), 10)
	if !ok {
		depInt = big.NewInt(0)
	}
	
	newTotalBid := new(big.Int).Add(prevBid, depInt)
	c.Bids[sender] = newTotalBid.String()

	highestBid, ok := new(big.Int).SetString(c.HighestBid, 10)
	if !ok {
		highestBid = big.NewInt(0)
	}

	if newTotalBid.Cmp(highestBid) > 0 {
		c.HighestBid = newTotalBid.String()
		c.HighestBidder = sender
		env.LogString("New highest bid: " + newTotalBid.String() + " by " + sender)
	} else {
		env.LogString("Bid recorded: total bid is " + newTotalBid.String() + " by " + sender)
	}
}

// @contract:mutating
func (c *Contract) Finalize() {
	if !c.Active {
		env.PanicStr("Already finalized")
	}
	
	now := env.GetBlockTimeMs()
	if now < c.EndTime {
		env.PanicStr("Auction not finished yet")
	}

	c.Active = false

	highestBidVal, ok := new(big.Int).SetString(c.HighestBid, 10)
	if !ok {
		highestBidVal = big.NewInt(0)
	}

	if highestBidVal.Cmp(big.NewInt(0)) > 0 {
		highestBidU128, err := types.U128FromString(c.HighestBid)
		if err != nil {
			env.PanicStr("Failed to parse HighestBid: " + err.Error())
		}
		promiseId := env.PromiseBatchCreate([]byte(c.Auctioneer))
		env.PromiseBatchActionTransfer(promiseId, highestBidU128)
		env.LogString("Winner payout transfer to auctioneer initiated")
	}

	for bidder, bidStr := range c.Bids {
		if bidder == c.HighestBidder {
			continue
		}
		
		bidVal, ok := new(big.Int).SetString(bidStr, 10)
		if !ok || bidVal.Cmp(big.NewInt(0)) <= 0 {
			continue
		}
		
		bidU128, err := types.U128FromString(bidStr)
		if err != nil {
			env.PanicStr("Failed to parse bid: " + err.Error())
		}
		
		promiseId := env.PromiseBatchCreate([]byte(bidder))
		env.PromiseBatchActionTransfer(promiseId, bidU128)
		env.LogString("Refunded " + bidStr + " yoctoNEAR to " + bidder)
	}
}

// @contract:mutating
func (c *Contract) Cancel() {
	sender, err := env.GetPredecessorAccountID()
	if err != nil {
		env.PanicStr("Failed to get predecessor account: " + err.Error())
	}
	
	if sender != c.Auctioneer {
		env.PanicStr("Only auctioneer can cancel the auction")
	}
	
	if !c.Active {
		env.PanicStr("Auction is already inactive")
	}

	c.Active = false

	for bidder, bidStr := range c.Bids {
		bidVal, ok := new(big.Int).SetString(bidStr, 10)
		if !ok || bidVal.Cmp(big.NewInt(0)) <= 0 {
			continue
		}
		
		bidU128, err := types.U128FromString(bidStr)
		if err != nil {
			env.PanicStr("Failed to parse bid: " + err.Error())
		}
		
		promiseId := env.PromiseBatchCreate([]byte(bidder))
		env.PromiseBatchActionTransfer(promiseId, bidU128)
		env.LogString("Refunded " + bidStr + " yoctoNEAR to " + bidder)
	}
	env.LogString("Auction cancelled and all refunds initiated")
}

// @contract:mutating
func (c *Contract) FixAuctioneer(new_auctioneer string) {
	if c.Auctioneer != "" {
		env.PanicStr("Auctioneer already set")
	}
	c.Auctioneer = new_auctioneer
	env.LogString("Auctioneer fixed: " + new_auctioneer)
}

func main() {}
