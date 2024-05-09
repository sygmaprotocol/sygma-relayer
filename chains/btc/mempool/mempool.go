package mempool

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
)

type Utxo struct {
	TxID          string `json:"txid"`
	Vout          uint32 `json:"vout"`
	Value         uint64 `json:"value"`
	Confirmations int64  `json:"confirmations"`
}

type Fee struct {
	FastestFee  int64
	HalfHourFee int64
}

type MempoolAPI struct {
	url string
}

func NewMempoolAPI(url string) *MempoolAPI {
	return &MempoolAPI{
		url: url,
	}
}

func (a *MempoolAPI) RecommendedFee() (*Fee, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/fees/recommended", a.url))
	if err != nil {
		return nil, err
	}

	var fee *Fee
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &fee)
	if err != nil {
		return nil, err
	}

	return fee, nil
}

func (a *MempoolAPI) Utxos(address string) ([]Utxo, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/address/%s/utxo", a.url, address))
	if err != nil {
		return nil, err
	}

	var utxos []Utxo
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &utxos)
	if err != nil {
		return nil, err
	}
	sort.Slice(utxos, func(i int, j int) bool {
		return utxos[i].TxID < utxos[j].TxID
	})

	return utxos, nil
}
