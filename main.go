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
)

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
	Comment string
}

func CalculateOrder (order Order) string {

	pizzaPrice := 2
	redbullPrice := 1

	amount := pizzaPrice * order.Products.Pizza + redbullPrice * order.Products.Redbull

	return strconv.Itoa(amount) + ".00"
}

func QWCreateBill (b BillStruct) int {

	apiUrl := "https://w.qiwi.com/"
	resource := "api/v2/prv/264131/bills/" + b.BillId
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

	//fmt.Println("url: ",urlStr)
	//fmt.Println("data:",r.Body)


	//respBytes := new(bytes.Buffer)
	//respBytes.ReadFrom(resp.Body)
	//fmt.Println("AUTH RESPONSE: ", string(respBytes.Bytes()))

	//fmt.Println(resp.StatusCode)

	return resp.StatusCode
}

func main() {
	r := gin.Default()
	r.POST("/order", func(c *gin.Context) {

		var json Order

		if c.BindJSON(&json) == nil {
			BindJSON(c,&json)

			newBill := BillStruct{
				Phone:c.Request.Header.Get("Authorization"),
				Amount:CalculateOrder(json),
				BillId:"181116" + strconv.Itoa(int(time.Now().UnixNano() / int64(time.Millisecond))),
				Comment: json.Comment}

			if QWCreateBill(newBill) == 200 {
				c.JSON(200, gin.H{
					"billId": newBill.BillId,
					"shopId":"264131",
					"amount":newBill.Amount})
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
		c.JSON(200, gin.H{
			"amount":"100.00",
			"comment":"Двойные оливки",
			"status":"paid"})
	})
	r.Run(":80")
}

func BindJSON(c *gin.Context, obj interface{}) error {
	if err := binding.JSON.Bind(c.Request, obj); err != nil {
		c.Error(err).SetType(gin.ErrorTypeBind)
		return err
	}
	return nil
}


//хранить ID счета чтобы по кошельку вытаскивать