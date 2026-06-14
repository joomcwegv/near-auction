package main

import (
    "encoding/json"
    "strconv"
    sdk "github.com/vlmoon99/near-sdk-go"
)

// Auction state structure
type Auction struct {
    Auctioneer    string            `json:"auctioneer"`
    EndTime       uint64            `json:"end_time"`
    HighestBid    sdk.Uint128       `json:"highest_bid"`
    HighestBidder string            `json:"highest_bidder"`
    Active        bool              `json:"active"`
    Bids          map[string]sdk.Uint128 `json:"bids"`
}

var (
    stateKey = []byte("state")
)

// Helper to load state, creates empty if none
func loadState() (*Auction, error) {
    raw, err := sdk.StorageRead(stateKey)
    if err != nil {
        return nil, err
    }
    if raw == nil {
        return &Auction{Active: false, Bids: map[string]sdk.Uint128{}}, nil
    }
    var a Auction
    if err := json.Unmarshal(raw, &a); err != nil {
        return nil, err
    }
    return &a, nil
}

func saveState(a *Auction) error {
    b, err := json.Marshal(a)
    if err != nil {
        return err
    }
    return sdk.StorageWrite(stateKey, b)
}

// Init args
type InitArgs struct {
    EndTime    uint64 `json:"end_time"`
    Auctioneer  string `json:"auctioneer"`
}

// @contract:init
func Init(args InitArgs) {
    if args.Auctioneer == "" {
        sdk.Panic("auctioneer must be set")
    }
    state := Auction{
        Auctioneer:    args.Auctioneer,
        EndTime:       args.EndTime,
        HighestBid:    sdk.NewUint128(0),
        HighestBidder: "",
        Active:        true,
        Bids:          map[string]sdk.Uint128{},
    }
    if err := saveState(&state); err != nil {
        sdk.Panic(err.Error())
    }
    sdk.LogString("Auction initialized")
}

// View info
type Info struct {
    Auctioneer    string `json:"auctioneer"`
    EndTime       uint64 `json:"end_time"`
    HighestBid    string `json:"highest_bid"`
    HighestBidder string `json:"highest_bidder"`
    Active        bool   `json:"active"`
}

// @contract:view
func GetInfo() Info {
    s, err := loadState()
    if err != nil {
        sdk.Panic(err.Error())
    }
    return Info{
        Auctioneer:    s.Auctioneer,
        EndTime:       s.EndTime,
        HighestBid:    s.HighestBid.String(),
        HighestBidder: s.HighestBidder,
        Active:        s.Active,
    }
}

// Place bid args
type BidArgs struct {
    Bid sdk.Uint128 `json:"bid"`
}

// @contract:payable [min_deposit=1]
// @contract:mutating
func PlaceBid(_ BidArgs) {
    state, err := loadState()
    if err != nil {
        sdk.Panic(err.Error())
    }
    if !state.Active {
        sdk.Panic("auction not active")
    }
    if sdk.BlockTimestamp() > state.EndTime {
        sdk.Panic("auction ended")
    }
    
    deposit := sdk.AttachedDeposit()
    sender := sdk.SignerAccountID()
    
    prev := state.Bids[sender]
    newTotal := prev.Add(deposit)
    state.Bids[sender] = newTotal
    
    if newTotal.GreaterThan(state.HighestBid) {
        state.HighestBid = newTotal
        state.HighestBidder = sender
        sdk.LogString("new highest bid: " + newTotal.String())
    }
    if err := saveState(state); err != nil {
        sdk.Panic(err.Error())
    }
}

// Finalize auction (anyone can call after end)
type FinalizeArgs struct{}

// @contract:mutating
func Finalize(_ FinalizeArgs) {
    state, err := loadState()
    if err != nil {
        sdk.Panic(err.Error())
    }
    if !state.Active {
        sdk.Panic("already finalized")
    }
    if sdk.BlockTimestamp() < state.EndTime {
        sdk.Panic("auction not finished")
    }
    state.Active = false
    // Pay winner amount to auctioneer
    if state.HighestBid.GreaterThan(sdk.NewUint128(0)) {
        sdk.Transfer(state.Auctioneer, state.HighestBid)
        sdk.LogString("winner payout to auctioneer")
    }
    // Refund others
    for addr, amount := range state.Bids {
        if addr == state.HighestBidder {
            continue
        }
        if amount.GreaterThan(sdk.NewUint128(0)) {
            sdk.Transfer(addr, amount)
        }
    }
    if err := saveState(state); err != nil {
        sdk.Panic(err.Error())
    }
    sdk.LogString("auction finalized")
}

// Cancel (only auctioneer)
type CancelArgs struct{}

// @contract:mutating
func Cancel(_ CancelArgs) {
    state, err := loadState()
    if err != nil {
        sdk.Panic(err.Error())
    }
    if sdk.SignerAccountID() != state.Auctioneer {
        sdk.Panic("only auctioneer can cancel")
    }
    if !state.Active {
        sdk.Panic("already closed")
    }
    // Refund all bidders
    for addr, amount := range state.Bids {
        if amount.GreaterThan(sdk.NewUint128(0)) {
            sdk.Transfer(addr, amount)
        }
    }
    state.Active = false
    if err := saveState(state); err != nil {
        sdk.Panic(err.Error())
    }
    sdk.LogString("auction cancelled and refunds issued")
}

// @contract:mutating
func FixAuctioneer(new_auctioneer string) {
    state, err := loadState()
    if err != nil {
        sdk.Panic(err.Error())
    }
    if state.Auctioneer != "" {
        sdk.Panic("Auctioneer already set!")
    }
    state.Auctioneer = new_auctioneer
    if err := saveState(state); err != nil {
        sdk.Panic(err.Error())
    }
    sdk.LogString("Auctioneer fixed: " + new_auctioneer)
}

// @contract:mutating
func Withdraw() {
    state, err := loadState()
    if err != nil {
        sdk.Panic(err.Error())
    }
    if sdk.SignerAccountID() != state.Auctioneer {
        sdk.Panic("Only auctioneer can withdraw")
    }
    balance := sdk.AccountBalance()
    if balance.GreaterThan(sdk.NewUint128(0)) {
        sdk.Transfer(state.Auctioneer, balance)
        sdk.LogString("All funds withdrawn by auctioneer")
    }
}

func main() {
    // near-go automatically builds WASM from this file.
}
