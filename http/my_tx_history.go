package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io/ioutil"
	"mm2_client/config"
	"mm2_client/helpers"
	"mm2_client/services"
	"net/http"
	"os"
	"strconv"
)

type MyTxHistoryAnswer struct {
	Result struct {
		Skipped      int         `json:"skipped"`
		Limit        int         `json:"limit"`
		Total        int         `json:"total"`
		CurrentBlock int         `json:"current_block,omitempty"`
		PageNumber   interface{} `json:"page_number,omitempty"`
		SyncStatus   struct {
			State string `json:"state,omitempty"`
		} `json:"sync_status,omitempty"`
		TotalPages   int `json:"total_pages"`
		Transactions []struct {
			BlockHeight   int    `json:"block_height"`
			Coin          string `json:"coin"`
			Confirmations int    `json:"confirmations"`
			FeeDetails    struct {
				Coin     string `json:"coin"`
				Gas      int    `json:"gas,omitempty"`
				GasPrice string `json:"gas_price,omitempty"`
				Amount   string `json:"amount,omitempty"`
				TotalFee string `json:"total_fee,omitempty"`
			} `json:"fee_details"`
			From            []string `json:"from"`
			InternalId      string   `json:"internal_id"`
			MyBalanceChange string   `json:"my_balance_change"`
			ReceivedByMe    string   `json:"received_by_me"`
			SpentByMe       string   `json:"spent_by_me"`
			Timestamp       int64    `json:"timestamp"`
			To              []string `json:"to"`
			TotalAmount     string   `json:"total_amount"`
			TxHash          string   `json:"tx_hash"`
			TxHex           string   `json:"tx_hex"`
		} `json:"transactions"`
	} `json:"result"`
	CoinType string
}

type MyTxHistoryRequest struct {
	Userpass string `json:"userpass"`
	Method   string `json:"method"`
	Coin     string `json:"coin"`
	Limit    int    `json:"limit"`
	//FromId     string `json:"from_id,omitempty"`
	PageNumber int  `json:"page_number,omitempty"`
	Max        bool `json:"max,omitempty"`
}

func NewMyTxHistoryRequest(coin string, defaultNbTx int, defaultPage int, max bool) *MyTxHistoryRequest {
	genReq := NewGenericRequest("my_tx_history")
	req := &MyTxHistoryRequest{Userpass: genReq.Userpass, Method: genReq.Method, Coin: coin, Limit: defaultNbTx, PageNumber: defaultPage, Max: max}
	return req
}

func (req *MyTxHistoryRequest) ToJson() string {
	b, err := json.Marshal(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return string(b)
}

func (answer *MyTxHistoryAnswer) ToTable(page int, tx int, withOriginalFiatValue bool, max bool, custom bool) {
	var data [][]string

	for _, curAnswer := range answer.Result.Transactions {
		if curAnswer.Coin != "" {
			val := "0"
			if !withOriginalFiatValue {
				val = services.RetrieveUSDValIfSupported(curAnswer.Coin)
				if val != "0" {
					val = helpers.BigFloatMultiply(curAnswer.MyBalanceChange, val, 2)
				}
			}

			totalFee := curAnswer.FeeDetails.Amount
			feeCoin := curAnswer.Coin
			if custom {
				totalFee = curAnswer.FeeDetails.TotalFee
				feeCoin = curAnswer.FeeDetails.Coin
			}

			txUrl := ""
			coin := curAnswer.Coin
			if (answer.CoinType == "ERC20" || answer.CoinType == "BEP20") &&
				(curAnswer.Coin != "BNB" && curAnswer.Coin != "BNBT" && curAnswer.Coin != "ETH" && curAnswer.Coin != "ETHR") {
				coin = coin + "-" + answer.CoinType
			}

			if cfg, ok := config.GCFGRegistry[coin]; ok {
				if cfg.ExplorerTxURL != "" {
					txUrl = cfg.ExplorerURL[0] + cfg.ExplorerTxURL + curAnswer.TxHash
				} else {
					txUrl = cfg.ExplorerURL[0] + "tx/" + curAnswer.TxHash
				}
			}

			cur := []string{curAnswer.From[0], curAnswer.To[0], curAnswer.MyBalanceChange + " (" + val + "$)", totalFee + " " + feeCoin, helpers.GetDateFromTimestamp(curAnswer.Timestamp, true), txUrl}
			data = append(data, cur)
		}
	}

	helpers.SortDoubleSliceByDate(data, 4, false)

	table := tablewriter.NewWriter(os.Stdout)
	if !custom && !max {
		table.SetFooter([]string{"", "", "Current Page", strconv.Itoa(page), "Nb Pages", strconv.Itoa(answer.Result.TotalPages)}) // Add Footer
	}
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"From", "To", "Balance Change", "Fee", "Date", "TxUrl"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.AppendBulk(data) // Add Bulk Data
	table.Render()
}

const customTxEndpoint = "https://komodo.live:3334/api/"

func MyTxHistory(coin string, defaultNbTx int, defaultPage int, withFiatValue bool, isMax bool) *MyTxHistoryAnswer {
	if _, ok := config.GCFGRegistry[coin]; ok {
		req := NewMyTxHistoryRequest(coin, defaultNbTx, defaultPage, isMax).ToJson()
		resp, err := http.Post(GMM2Endpoint, "application/json", bytes.NewBuffer([]byte(req)))
		if err != nil {
			fmt.Printf("Err: %v\n", err)
			return nil
		}
		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var answer = &MyTxHistoryAnswer{}
			decodeErr := json.NewDecoder(resp.Body).Decode(answer)
			if decodeErr != nil {
				fmt.Printf("Err: %v\n", err)
				return nil
			}
			return answer
		} else {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("Err: %s\n", bodyBytes)
		}
	} else {
		fmt.Printf("coin: %s doesn't exist or is not present in the desktop configuration\n", coin)
		return nil
	}
	return nil
}

func CustomMyTxHistory(coin string, defaultNbTx int, defaultPage int, withFiatValue bool, isMax bool, contract string,
	query string, address string, coinType string) *MyTxHistoryAnswer {
	endpoint := customTxEndpoint
	if contract != "" {
		endpoint = endpoint + "v2/" + query + "/" + contract + "/" + address
	} else {
		endpoint = endpoint + "v1/" + query + "/" + address
	}
	resp, err := http.Get(endpoint)
	if err != nil {
		fmt.Printf("Error occured: %v\n", err)
		return nil
	}
	defer resp.Body.Close()
	var cResp = new(MyTxHistoryAnswer)
	if decodeErr := json.NewDecoder(resp.Body).Decode(cResp); decodeErr != nil {
		fmt.Printf("Error occured: %v\n", decodeErr)
		return nil
	}
	cResp.CoinType = coinType
	return cResp
}
