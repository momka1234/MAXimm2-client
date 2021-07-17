package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kpango/glg"
	"github.com/olekukonko/tablewriter"
	"io/ioutil"
	"mm2_client/config"
	"mm2_client/helpers"
	"mm2_client/services"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type MyBalanceRequest struct {
	Userpass string `json:"userpass"`
	Method   string `json:"method"`
	Coin     string `json:"coin"`
}

type MyBalanceAnswer struct {
	Address            string `json:"address"`
	Balance            string `json:"balance"`
	UnspendableBalance string `json:"unspendable_balance"`
	Coin               string `json:"coin"`
}

func NewMyBalanceCoinRequest(cfg *config.DesktopCFG) *MyBalanceRequest {
	genReq := NewGenericRequest("my_balance")
	req := &MyBalanceRequest{Userpass: genReq.Userpass, Method: genReq.Method}
	req.Coin = cfg.Coin
	return req
}

func (req *MyBalanceRequest) ToJson() string {
	b, err := json.Marshal(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return string(b)
}

func (answer *MyBalanceAnswer) ToTable() {
	if answer.Coin != "" {
		val, _, provider := services.RetrieveUSDValIfSupported(answer.Coin)
		if val != "0" {
			val = helpers.BigFloatMultiply(answer.Balance, val, 2)
		}

		data := [][]string{
			{answer.Coin, answer.Address, answer.Balance, val, answer.UnspendableBalance, provider},
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"Coin", "Address", "Balance", "Balance (USD)", "Unspendable", "Price Provider"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.AppendBulk(data) // Add Bulk Data
		table.Render()
	}
}

func ToTableMyBalanceAnswers(answers []MyBalanceAnswer) {
	var data [][]string

	total := "0"
	for _, answer := range answers {
		if answer.Coin != "" {
			val, _, provider := services.RetrieveUSDValIfSupported(answer.Coin)
			if val != "0" {
				val = helpers.BigFloatMultiply(answer.Balance, val, 2)
			}

			cur := []string{answer.Coin, answer.Address, answer.Balance, val, answer.UnspendableBalance, provider}
			total = helpers.BigFloatAdd(total, val, 2)
			data = append(data, cur)
		}
	}

	helpers.SortDoubleSlice(data, 3, false)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Coin", "Address", "Balance", "Balance (USD)", "Unspendable", "Price Provider"})
	table.SetFooter([]string{"", "", "", "Total: " + total + " $", "", ""})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.AppendBulk(data) // Add Bulk Data
	table.Render()
}

func ToSliceEmptyBalance(answers []MyBalanceAnswer, withoutTestCoin bool) []string {
	var out []string
	for _, cur := range answers {
		if v, err := strconv.ParseFloat(cur.Balance, 64); err == nil && v <= 0 {
			out = append(out, cur.Coin)
		}
		if withoutTestCoin {
			if val, ok := config.GCFGRegistry[cur.Coin]; ok {
				if val.IsTestNet || strings.Contains(val.Name, "TESTCOIN") {
					out = append(out, cur.Coin)
				}
			}
		}
	}
	return out
}

func MyBalance(coin string) (*MyBalanceAnswer, error) {
	if val, ok := config.GCFGRegistry[coin]; ok {
		req := NewMyBalanceCoinRequest(val).ToJson()
		resp, err := http.Post(GMM2Endpoint, "application/json", bytes.NewBuffer([]byte(req)))
		if err != nil {
			glg.Errorf("Err: %v", err)
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var answer = &MyBalanceAnswer{}
			decodeErr := json.NewDecoder(resp.Body).Decode(answer)
			if decodeErr != nil {
				glg.Errorf("Err: %v", err)
				return nil, decodeErr
			}
			return answer, nil
		} else {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("Err: %s\n", bodyBytes)
		}
	} else {
		err := fmt.Sprintf("coin: %s doesn't exist or is not present in the desktop configuration", coin)
		return nil, errors.New(err)
	}
	return nil, errors.New("unknown error")
}
