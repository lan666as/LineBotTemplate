// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"io/ioutil"
	"encoding/json"

	"github.com/line/line-bot-sdk-go/linebot"
)

var bot *linebot.Client

func main() {
	var err error
	bot, err = linebot.New(os.Getenv("ChannelSecret"), os.Getenv("ChannelAccessToken"))
	log.Println("Bot:", bot, " err:", err)
	http.HandleFunc("/callback", callbackHandler)
	port := os.Getenv("PORT")
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}

func getSimsimi(word string) string{
	resp, err := http.Get("http://sandbox.api.simsimi.com/request.p?key=1b4f97fa-a422-45f0-8faf-0122ddd2dc5c&lc=id&ft=1.0&text=" + url.QueryEscape(word))
	if err != nil {
		log.Print(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	type SimsimiResp struct {
    	Response string `json:"response"`
    	Id  string `json:"id"`
    	Result    string `json:"result"`
    	Msg  string `json:"msg"`
	}
	var resp2 = new(SimsimiResp)
	err = json.Unmarshal([]byte(body), &resp2)
	if err != nil{
		log.Print(err)
	}
	return string(resp2.Response)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)

	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			if profile := bot.GetProfile(event.Source.UserID).Do(){
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(profile.DisplayName+": "+message.Text+" -> "+getSimsimi(message.Text))).Do(); err != nil {
					log.Print(err)
				}
			}
			}
		}
	}
}
