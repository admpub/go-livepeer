package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/glog"
	"github.com/livepeer/go-livepeer/eth"
	lpTypes "github.com/livepeer/go-livepeer/eth/types"
	"github.com/olekukonko/tablewriter"
)

func (w *wizard) stats(showTranscoder bool) {
	addrMap, err := w.getContractAddresses()
	if err != nil {
		glog.Errorf("Error getting contract addresses: %v", err)
		return
	}

	fmt.Println("+-----------+")
	fmt.Println("|NODE STATS|")
	fmt.Println("+-----------+")

	table := tablewriter.NewWriter(os.Stdout)
	data := [][]string{
		[]string{"Node ID", w.getNodeID()},
		[]string{"Node Addr", w.getNodeAddr()},
		[]string{"RTMP Port", w.rtmpPort},
		[]string{"HTTP Port", w.httpPort},
		[]string{"Controller Address", addrMap["Controller"].Hex()},
		[]string{"LivepeerToken Address", addrMap["LivepeerToken"].Hex()},
		[]string{"LivepeerTokenFaucet Address", addrMap["LivepeerTokenFaucet"].Hex()},
		[]string{"ETH Account", w.getEthAddr()},
		[]string{"LPT Balance", w.getTokenBalance()},
		[]string{"ETH Balance", w.getEthBalance()},
	}

	for _, v := range data {
		table.Append(v)
	}

	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.SetCenterSeparator("*")
	table.SetRowLine(true)
	table.SetColumnSeparator("|")
	table.Render()

	if showTranscoder {
		w.transcoderStats()
		w.delegatorStats()
	} else {
		w.broadcastStats()
		w.delegatorStats()
	}

	currentRound := w.currentRound()

	fmt.Printf("CURRENT ROUND: %v\n", currentRound)
}

func (w *wizard) broadcastStats() {
	fmt.Println("+-----------------+")
	fmt.Println("|BROADCASTER STATS|")
	fmt.Println("+-----------------+")

	price, transcodingOptions := w.getBroadcastConfig()

	table := tablewriter.NewWriter(os.Stdout)
	data := [][]string{
		[]string{"Deposit", w.getDeposit()},
		[]string{"Broadcast Price Per Segment", price.String()},
		[]string{"Broadcast Transcoding Options", transcodingOptions},
	}

	for _, v := range data {
		table.Append(v)
	}

	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.SetCenterSeparator("*")
	table.SetRowLine(true)
	table.SetColumnSeparator("|")
	table.Render()
}

func (w *wizard) transcoderStats() {
	t, err := w.getTranscoderInfo()
	if err != nil {
		glog.Errorf("Error getting transcoder info: %v", err)
		return
	}

	fmt.Println("+----------------+")
	fmt.Println("|TRANSCODER STATS|")
	fmt.Println("+----------------+")

	table := tablewriter.NewWriter(os.Stdout)
	data := [][]string{
		[]string{"Status", t.Status},
		[]string{"Active", strconv.FormatBool(t.Active)},
		[]string{"Delegated Stake", eth.FormatUnits(t.DelegatedStake, "LPT")},
		[]string{"Reward Cut (%)", eth.FormatPerc(t.BlockRewardCut)},
		[]string{"Fee Share (%)", eth.FormatPerc(t.FeeShare)},
		[]string{"Price Per Segment", eth.FormatUnits(t.PricePerSegment, "ETH")},
		[]string{"Pending Reward Cut (%)", eth.FormatPerc(t.PendingBlockRewardCut)},
		[]string{"Pending Fee Share (%)", eth.FormatPerc(t.PendingFeeShare)},
		[]string{"Pending Price Per Segment", eth.FormatUnits(t.PendingPricePerSegment, "ETH")},
		[]string{"Last Reward Round", t.LastRewardRound.String()},
	}

	for _, v := range data {
		table.Append(v)
	}

	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.SetCenterSeparator("*")
	table.SetRowLine(true)
	table.SetColumnSeparator("|")
	table.Render()
}

func (w *wizard) delegatorStats() {
	d, err := w.getDelegatorInfo()
	if err != nil {
		glog.Errorf("Error getting delegator info: %v", err)
		return
	}

	fmt.Println("+---------------+")
	fmt.Println("|DELEGATOR STATS|")
	fmt.Println("+---------------+")

	table := tablewriter.NewWriter(os.Stdout)
	data := [][]string{
		[]string{"Status", d.Status},
		[]string{"Stake", d.BondedAmount.String()},
		[]string{"Collected Fees", d.Fees.String()},
		[]string{"Pending Stake", d.PendingStake.String()},
		[]string{"Pending Fees", d.PendingFees.String()},
		[]string{"Delegated Stake", d.DelegatedAmount.String()},
		[]string{"Delegate Address", d.DelegateAddress.Hex()},
		[]string{"Last Claim Round", d.LastClaimTokenPoolsSharesRound.String()},
		[]string{"Start Round", d.StartRound.String()},
		[]string{"Withdraw Round", d.WithdrawRound.String()},
	}

	for _, v := range data {
		table.Append(v)
	}

	table.SetAlignment(tablewriter.ALIGN_RIGHT)
	table.SetCenterSeparator("*")
	table.SetRowLine(true)
	table.SetColumnSeparator("|")
	table.Render()
}

func (w *wizard) getNodeID() string {
	return httpGet(fmt.Sprintf("http://%v:%v/nodeID", w.host, w.httpPort))
}

func (w *wizard) getNodeAddr() string {
	return httpGet(fmt.Sprintf("http://%v:%v/nodeAddrs", w.host, w.httpPort))
}

func (w *wizard) getContractAddresses() (map[string]common.Address, error) {
	resp, err := http.Get(fmt.Sprintf("http://%v:%v/contractAddresses", w.host, w.httpPort))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var addrMap map[string]common.Address
	err = json.Unmarshal(result, &addrMap)
	if err != nil {
		return nil, err
	}

	return addrMap, nil
}

func (w *wizard) getEthAddr() string {
	addr := httpGet(fmt.Sprintf("http://%v:%v/ethAddr", w.host, w.httpPort))
	if addr == "" {
		addr = "Unknown"
	}
	return addr
}

func (w *wizard) getTokenBalance() string {
	b := httpGet(fmt.Sprintf("http://%v:%v/tokenBalance", w.host, w.httpPort))
	if b == "" {
		b = "Unknown"
	}
	return b
}

func (w *wizard) getEthBalance() string {
	e := httpGet(fmt.Sprintf("http://%v:%v/ethBalance", w.host, w.httpPort))
	if e == "" {
		e = "Unknown"
	}
	return e
}

func (w *wizard) getDeposit() string {
	e := httpGet(fmt.Sprintf("http://%v:%v/broadcasterDeposit", w.host, w.httpPort))
	if e == "" {
		e = "Unknown"
	}
	return e
}

func (w *wizard) getBroadcastConfig() (*big.Int, string) {
	resp, err := http.Get(fmt.Sprintf("http://%v:%v/getBroadcastConfig", w.host, w.httpPort))
	if err != nil {
		glog.Errorf("Error getting broadcast config: %v", err)
		return nil, ""
	}

	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Error reading response: %v", err)
		return nil, ""
	}

	var config struct {
		MaxPricePerSegment *big.Int
		TranscodingOptions string
	}
	err = json.Unmarshal(result, &config)
	if err != nil {
		glog.Errorf("Error unmarshalling broadcast config: %v", err)
		return nil, ""
	}

	return config.MaxPricePerSegment, config.TranscodingOptions
}

func (w *wizard) getTranscoderInfo() (lpTypes.Transcoder, error) {
	resp, err := http.Get(fmt.Sprintf("http://%v:%v/transcoderInfo", w.host, w.httpPort))
	if err != nil {
		return lpTypes.Transcoder{}, err
	}

	defer resp.Body.Close()

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return lpTypes.Transcoder{}, err
	}

	var tInfo lpTypes.Transcoder
	err = json.Unmarshal(result, &tInfo)
	if err != nil {
		return lpTypes.Transcoder{}, err
	}

	return tInfo, nil
}

func (w *wizard) getDelegatorInfo() (lpTypes.Delegator, error) {
	resp, err := http.Get(fmt.Sprintf("http://%v:%v/delegatorInfo", w.host, w.httpPort))
	if err != nil {
		return lpTypes.Delegator{}, err
	}

	defer resp.Body.Close()

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return lpTypes.Delegator{}, err
	}

	var dInfo lpTypes.Delegator
	err = json.Unmarshal(result, &dInfo)
	if err != nil {
		return lpTypes.Delegator{}, err
	}

	return dInfo, nil
}
