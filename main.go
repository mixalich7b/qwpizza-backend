package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"net/url"
	"bytes"
	"strconv"
	"time"
	"fmt"
	"os"
	"encoding/json"
	"io/ioutil"
)

var storage map[string]string
var shopId = "264131"

type Order struct {
	Products ProductStruct `json:"products"`
	Comment string `json:"comment"`
}

type ProductStruct struct {
	Pizza int `json:"pizza"`
	Redbull int `json:"redbull"`
}

type BillStruct struct {
	Phone string
	Amount string
	BillId string
	BillStatus string
	Comment string
}

type QWBillStructResponse struct {
	Response QWBillStructResponseContainer `json:"response"`
}

type QWBillStructResponseContainer struct {
	ResultCode int `json:"result_code"`
	Bill QWBillStruct `json:"bill"`
}

type QWBillStruct struct {
	Amount string `json:"amount"`
	BillId string `json:"bill_id"`
	BillStatus string `json:"status"`
	Comment string `json:"comment"`
}


func CalculateOrder (order Order) string {

	pizzaPrice := 2
	redbullPrice := 1

	amount := pizzaPrice * order.Products.Pizza + redbullPrice * order.Products.Redbull

	return strconv.Itoa(amount) + ".00"
}

func parseBillFromResponse(resp *http.Response, phone string) *BillStruct {
	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil
		}
		var billResponse QWBillStructResponse
		err = json.Unmarshal(body, &billResponse)
		if err != nil {
			fmt.Println("cannot parse json, error:", err)
			return nil
		}

		return &BillStruct{
			Phone:phone,
			Amount:billResponse.Response.Bill.Amount,
			BillId:billResponse.Response.Bill.BillId,
			BillStatus:billResponse.Response.Bill.BillStatus,
			Comment:billResponse.Response.Bill.Comment,
		}
	} else {
		return nil
	}
}

func QWCreateBill (b BillStruct) *BillStruct {

	apiUrl := "https://w.qiwi.com/"
	resource := "api/v2/prv/" + shopId + "/bills/" + b.BillId
	data := url.Values{}
	user := "tel:+" + b.Phone

	data.Set("user", user)
	data.Add("amount", b.Amount)
	data.Add("ccy","RUB")
	data.Add("comment",b.Comment)
	data.Add("pay_source","qw")
	data.Add("lifetime","2017-11-12T10:00:00")

	u, _ := url.ParseRequestURI(apiUrl)
	u.Path = resource
	urlStr := fmt.Sprintf("%v", u) // "https://api.com/user/"

	client := &http.Client{}
	r, _ := http.NewRequest("PUT", urlStr, bytes.NewBufferString(data.Encode())) // <-- URL-encoded payload
	r.Header.Add("Authorization", "Basic MTAxNjcxNjE6WHRJcDVGdEVKdXRKUlhLMWcwRmE=")
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, _ := client.Do(r)

	return parseBillFromResponse(resp, b.Phone)
}

func QWBillStatus (billId string, phone string) *BillStruct {

	apiUrl := "https://w.qiwi.com/"
	resource := "api/v2/prv/" + shopId + "/bills/" + billId
	data := url.Values{}

	u, _ := url.ParseRequestURI(apiUrl)
	u.Path = resource
	urlStr := fmt.Sprintf("%v", u)

	client := &http.Client{}
	r, _ := http.NewRequest("GET", urlStr, nil)
	r.Header.Add("Authorization", "Basic MTAxNjcxNjE6WHRJcDVGdEVKdXRKUlhLMWcwRmE=")
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, _ := client.Do(r)

	return parseBillFromResponse(resp, phone)
}

func main() {
	storage = make(map[string]string)

	r := gin.Default()
	r.POST("/order", func(c *gin.Context) {

		var json Order

		if c.BindJSON(&json) == nil {
			BindJSON(c,&json)

			phone := c.Request.Header.Get("Authorization")
			newBill := BillStruct{
				Phone:phone,
				Amount:CalculateOrder(json),
				BillId:"181116" + strconv.Itoa(int(time.Now().UnixNano() / int64(time.Millisecond))),
				BillStatus:"",
				Comment: json.Comment}

			createdBill := QWCreateBill(newBill)
			if createdBill != nil {
				storage[phone] = createdBill.BillId
				c.JSON(200, gin.H{
					"billId": createdBill.BillId,
					"shopId":shopId,
					"amount":createdBill.Amount,
					"status":createdBill.BillStatus,
					"comment":createdBill.Comment})
			} else {
				c.JSON(500, gin.H{
					"error":"QW error"})
			}
		} else {
			c.JSON(400, gin.H{
				"error":"bad JSON"})
		}
	})

	r.GET("/status", func(c *gin.Context) {
		phone := c.Request.Header.Get("Authorization")
		billId := storage[phone]
		if billId == "" {
			c.JSON(404, gin.H{
				"error":"user not found"})
		} else {
			bill := QWBillStatus(billId, phone)
			c.JSON(200, gin.H{
				"billId": bill.BillId,
				"shopId":shopId,
				"amount":bill.Amount,
				"status":bill.BillStatus,
				"comment":bill.Comment})
		}
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}

func BindJSON(c *gin.Context, obj interface{}) error {
	if err := binding.JSON.Bind(c.Request, obj); err != nil {
		c.Error(err).SetType(gin.ErrorTypeBind)
		return err
	}
	return nil
}


//хранить ID счета чтобы по кошельку вытаскивать