// Copyright 2016 LINE Corporation
//
// LINE Corporation licenses this file to you under the Apache License,
// version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"encoding/json"
	"bytes"
	//"strconv"
	b64 "encoding/base64"

	"github.com/line/line-bot-sdk-go/linebot"
)

func main() {
	app, err := NewKitchenSink(
		os.Getenv("ChannelSecret"),
		os.Getenv("ChannelAccessToken"),
		os.Getenv("APP_BASE_URL"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// serve /static/** files
	staticFileServer := http.FileServer(http.Dir("static"))
	http.HandleFunc("/static/", http.StripPrefix("/static/", staticFileServer).ServeHTTP)
	// serve /downloaded/** files
	downloadedFileServer := http.FileServer(http.Dir(app.downloadDir))
	http.HandleFunc("/downloaded/", http.StripPrefix("/downloaded/", downloadedFileServer).ServeHTTP)

	http.HandleFunc("/callback", app.Callback)
	// This is just a sample code.
	// For actually use, you must support HTTPS by using `ListenAndServeTLS`, reverse proxy or etc.
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}

// KitchenSink app
type KitchenSink struct {
	bot         *linebot.Client
	appBaseURL  string
	downloadDir string
}

// NewKitchenSink function
func NewKitchenSink(channelSecret, channelToken, appBaseURL string) (*KitchenSink, error) {
	apiEndpointBase := linebot.APIEndpointBase
	bot, err := linebot.New(
		channelSecret,
		channelToken,
		linebot.WithEndpointBase(apiEndpointBase), // Usually you omit this.
	)
	if err != nil {
		return nil, err
	}
	downloadDir := filepath.Join(filepath.Dir(os.Args[0]), "line-bot")
	_, err = os.Stat(downloadDir)
	if err != nil {
		if err := os.Mkdir(downloadDir, 0777); err != nil {
			return nil, err
		}
	}
	return &KitchenSink{
		bot:         bot,
		appBaseURL:  appBaseURL,
		downloadDir: downloadDir,
	}, nil
}

func (app *KitchenSink) GetSimsimi(word string) string{
	resp, err := http.Get("http://www.simsimi.com/getRealtimeReq?uuid=TZq4ZUZta6MhnHeGYMVBbhMZkNW0r6zgGQalwYMog6X&lc=id&ft=1&reqText=" + url.QueryEscape(word))
	if err != nil{
		log.Print(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	type SimsimiResp struct {
    	Status json.Number `json:"status"`
    	RespSentence  string `json:"respSentence"`
	}
	var resp2 = new(SimsimiResp)
	err = json.Unmarshal([]byte(body), &resp2)
	if err != nil{
		log.Print(err)
	}
	return string(resp2.RespSentence)
}
func (app *KitchenSink) GetIndico(messageID string) string{
	content, err := app.bot.GetMessageContent(messageID).Do()
	if err != nil {
		log.Print(err)
	}
	defer content.Content.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(content.Content)
	s := buf.String()
	contentEnc := b64.StdEncoding.EncodeToString([]byte(s))
	url := "https://apiv2.indico.io/imagerecognition"
    log.Print("URL:>", url)

    var jsonStr = []byte(`{"data":"`+ contentEnc  +`", "top_n" : 15, "hq" : true}`)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
    req.Header.Set("X-ApiKey", "de5ec059652890635e7657540441e22e")
    //req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    log.Print("response Status:", resp.Status)
    log.Print("response Headers:", resp.Header)
    body, _ := ioutil.ReadAll(resp.Body)
    log.Print("response Body:", string(body))
    return string(body)
}
func (app *KitchenSink) GetImagga(imgUrl string) string{
	client := &http.Client{}
	api_key := "acc_d4e658488d09387"
	api_secret := "7928f55db5b5693d33ee8764824095a6"

	req, _ := http.NewRequest("GET", "https://api.imagga.com/v1/tagging?url="+string(imgUrl)+"&limit=20", nil)
	req.SetBasicAuth(api_key, api_secret)

	resp, err := client.Do(req)

	if err != nil {
	log.Print("Error when sending request to the server")
	}

	defer resp.Body.Close()
	resp_body, _ := ioutil.ReadAll(resp.Body)

	log.Print(resp.Status)
	return string(resp_body)
}
/*func (app *KitchenSink) GetClarifai(imgUrl string) string{
	client := &http.Client{}
	api_key := "acc_d4e658488d09387"
	api_secret := "7928f55db5b5693d33ee8764824095a6"

	req, _ := http.NewRequest("GET", "https://api.imagga.com/v1/tagging?url="+string(imgUrl), nil)
	req.SetBasicAuth(api_key, api_secret)

	resp, err := client.Do(req)

	if err != nil {
	log.Print("Error when sending request to the server")
	return
	}

	defer resp.Body.Close()
	resp_body, _ := ioutil.ReadAll(resp.Body)

	log.Print(resp.Status)
	return string(resp_body)
}
*/
// Callback function for http server
func (app *KitchenSink) Callback(w http.ResponseWriter, r *http.Request) {
	events, err := app.bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}
	log.Printf("Got events %v", events)
	for _, event := range events {
		switch event.Type {
		case linebot.EventTypeMessage:
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				if err := app.handleText(message, event.ReplyToken, event.Source); err != nil {
					log.Print(err)
				}
			case *linebot.ImageMessage:
				if err := app.handleImage(message, event.ReplyToken, event.Source); err != nil {
					log.Print(err)
				}
			case *linebot.VideoMessage:
				if err := app.handleVideo(message, event.ReplyToken); err != nil {
					log.Print(err)
				}
			case *linebot.AudioMessage:
				if err := app.handleAudio(message, event.ReplyToken); err != nil {
					log.Print(err)
				}
			case *linebot.LocationMessage:
				if err := app.handleLocation(message, event.ReplyToken); err != nil {
					log.Print(err)
				}
			case *linebot.StickerMessage:
				if err := app.handleSticker(message, event.ReplyToken); err != nil {
					log.Print(err)
				}
			default:
				log.Printf("Unknown message: %v", message)
			}
		case linebot.EventTypeFollow:
			if err := app.replyText(event.ReplyToken, "Got followed event"); err != nil {
				log.Print(err)
			}
		case linebot.EventTypeUnfollow:
			log.Printf("Unfollowed this bot: %v", event)
		case linebot.EventTypeJoin:
			if err := app.replyText(event.ReplyToken, "Joined "+string(event.Source.Type)); err != nil {
				log.Print(err)
			}
		case linebot.EventTypeLeave:
			log.Printf("Left: %v", event)
		case linebot.EventTypePostback:
			if err := app.replyText(event.ReplyToken, "Got postback: "+event.Postback.Data); err != nil {
				log.Print(err)
			}
		case linebot.EventTypeBeacon:
			if err := app.replyText(event.ReplyToken, "Got beacon: "+event.Beacon.Hwid); err != nil {
				log.Print(err)
			}
		default:
			log.Printf("Unknown event: %v", event)
		}
	}
}

func (app *KitchenSink) handleText(message *linebot.TextMessage, replyToken string, source *linebot.EventSource) error {
	switch message.Text {
	case "profile":
		if source.UserID != "" {
			profile, err := app.bot.GetProfile(source.UserID).Do()
			if err != nil {
				return app.replyText(replyToken, err.Error())
			}
			if _, err := app.bot.ReplyMessage(
				replyToken,
				linebot.NewTextMessage("Display name: "+profile.DisplayName + " ID: " + source.UserID),
				linebot.NewTextMessage("Status message: "+profile.StatusMessage),
			).Do(); err != nil {
				return err
			}
		} else if source.GroupID != "" {
			if _, err := app.bot.ReplyMessage(
				replyToken,
				linebot.NewTextMessage("Group ID: " + source.GroupID),
			).Do(); err != nil {
				return err
			}
		} else {
			return app.replyText(replyToken, "Bot can't use profile API without user ID")
		}
	case "buttons":
		imageURL := app.appBaseURL + "/static/buttons/1040.jpg"
		template := linebot.NewButtonsTemplate(
			imageURL, "My button sample", "Hello, my button",
			linebot.NewURITemplateAction("Go to line.me", "https://line.me"),
			linebot.NewPostbackTemplateAction("Say hello1", "hello こんにちは", ""),
			linebot.NewPostbackTemplateAction("Autoaddfriends", "hello こんにちは", "Aku suka Hafiz lho!"),
			linebot.NewMessageTemplateAction("Autolikes", "Aku suka Hafiz lho!"),
		)
		if _, err := app.bot.ReplyMessage(
			replyToken,
			linebot.NewTemplateMessage("Buttons alt text", template),
		).Do(); err != nil {
			return err
		}
	case "confirm":
		template := linebot.NewConfirmTemplate(
			"Do it?",
			linebot.NewMessageTemplateAction("Yes", "Yes!"),
			linebot.NewMessageTemplateAction("No", "No!"),
		)
		if _, err := app.bot.ReplyMessage(
			replyToken,
			linebot.NewTemplateMessage("Confirm alt text", template),
		).Do(); err != nil {
			return err
		}
	/*
	case "!simsimi off":
			os.Setenv("SimsimiBool", "false")
	case "!simsimi on":
			os.Setenv("SimsimiBool", "true")*/
	case "carousel":
		imageURL := app.appBaseURL + "/static/buttons/1040.jpg"
		template := linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "hoge", "fuga",
				linebot.NewURITemplateAction("Go to line.me", "https://line.me"),
				linebot.NewPostbackTemplateAction("Say hello1", "hello こんにちは", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "hoge", "fuga",
				linebot.NewPostbackTemplateAction("言 hello2", "hello こんにちは", "hello こんにちは"),
				linebot.NewMessageTemplateAction("Say message", "Rice=米"),
			),
		)
		if _, err := app.bot.ReplyMessage(
			replyToken,
			linebot.NewTemplateMessage("Carousel alt text", template),
		).Do(); err != nil {
			return err
		}
	case "imagemap":
		if _, err := app.bot.ReplyMessage(
			replyToken,
			linebot.NewImagemapMessage(
				app.appBaseURL+"/static/rich",
				"Imagemap alt text",
				linebot.ImagemapBaseSize{1040, 1040},
				linebot.NewURIImagemapAction("https://store.line.me/family/manga/en", linebot.ImagemapArea{0, 0, 520, 520}),
				linebot.NewURIImagemapAction("https://store.line.me/family/music/en", linebot.ImagemapArea{520, 0, 520, 520}),
				linebot.NewURIImagemapAction("https://store.line.me/family/play/en", linebot.ImagemapArea{0, 520, 520, 520}),
				linebot.NewMessageImagemapAction("URANAI!", linebot.ImagemapArea{520, 520, 520, 520}),
			),
		).Do(); err != nil {
			return err
		}
	/*case "bye":
		switch source.Type {
		case linebot.EventSourceTypeUser:
			return app.replyText(replyToken, "Bot can't leave from 1:1 chat")
		case linebot.EventSourceTypeGroup:
			if err := app.replyText(replyToken, "Leaving group"); err != nil {
				return err
			}
			if _, err := app.bot.LeaveGroup(source.GroupID).Do(); err != nil {
				return app.replyText(replyToken, err.Error())
			}
		case linebot.EventSourceTypeRoom:
			if err := app.replyText(replyToken, "Leaving room"); err != nil {
				return err
			}
			if _, err := app.bot.LeaveRoom(source.RoomID).Do(); err != nil {
				return app.replyText(replyToken, err.Error())
			}
		}*/
	default:
		//if(strconv.ParseBool(os.Getenv("SimsimiBool"))){
			log.Printf("Echo message to %s: %s", replyToken, message.Text)
			if _, err := app.bot.ReplyMessage(
				replyToken,
				linebot.NewTextMessage(message.Text+" -> " + app.GetSimsimi(string(message.Text))),
			).Do(); err != nil {
				return err
			}
		//}
	}
	return nil
}

func (app *KitchenSink) handleImage(message *linebot.ImageMessage, replyToken string, source *linebot.EventSource) error {
	return app.handleHeavyContent(message.ID, func(originalContent *os.File) error {
		// You need to install ImageMagick.
		// And you should consider about security and scalability.
		previewImagePath := originalContent.Name() + "-preview"
		_, err := exec.Command("convert", "-resize", "240x", "jpeg:"+originalContent.Name(), "jpeg:"+previewImagePath).Output()
		if err != nil {
			return err
		}

		originalContentURL := app.appBaseURL + "/downloaded/" + filepath.Base(originalContent.Name())
		previewImageURL := app.appBaseURL + "/downloaded/" + filepath.Base(previewImagePath)
		if((source.UserID == "U54182c7c0ee792ac90a24f95282dd048" && source.Type == linebot.EventSourceTypeUser) || (source.GroupID == "Ca560c9c5db15fcd611a9329b1d1b4713" && source.Type == linebot.EventSourceTypeGroup)) {
			if _, err := app.bot.ReplyMessage(
				replyToken,
				linebot.NewImageMessage(originalContentURL, previewImageURL),
				linebot.NewTextMessage("Analisis #1: " + app.GetIndico(string(message.ID))),
				linebot.NewTextMessage("Analisis #2: " + app.GetImagga(string(originalContentURL))),
			).Do(); err != nil {
				return err
			}
		} else {
			if _, err := app.bot.ReplyMessage(
				replyToken,
				linebot.NewImageMessage(originalContentURL, previewImageURL),
			).Do(); err != nil {
				return err
			}
		}
		return nil
	})
}


func (app *KitchenSink) handleVideo(message *linebot.VideoMessage, replyToken string) error {
	return app.handleHeavyContent(message.ID, func(originalContent *os.File) error {
		// You need to install FFmpeg and ImageMagick.
		// And you should consider about security and scalability.
		previewImagePath := originalContent.Name() + "-preview"
		_, err := exec.Command("convert", "mp4:"+originalContent.Name()+"[0]", "jpeg:"+previewImagePath).Output()
		if err != nil {
			return err
		}

		originalContentURL := app.appBaseURL + "/downloaded/" + filepath.Base(originalContent.Name())
		previewImageURL := app.appBaseURL + "/downloaded/" + filepath.Base(previewImagePath)
		if _, err := app.bot.ReplyMessage(
			replyToken,
			linebot.NewVideoMessage(originalContentURL, previewImageURL),
		).Do(); err != nil {
			return err
		}
		return nil
	})
}

func (app *KitchenSink) handleAudio(message *linebot.AudioMessage, replyToken string) error {
	return app.handleHeavyContent(message.ID, func(originalContent *os.File) error {
		originalContentURL := app.appBaseURL + "/downloaded/" + filepath.Base(originalContent.Name())
		if _, err := app.bot.ReplyMessage(
			replyToken,
			linebot.NewAudioMessage(originalContentURL, 100),
		).Do(); err != nil {
			return err
		}
		return nil
	})
}

func (app *KitchenSink) handleLocation(message *linebot.LocationMessage, replyToken string) error {
	if _, err := app.bot.ReplyMessage(
		replyToken,
		linebot.NewLocationMessage(message.Title, message.Address, message.Latitude, message.Longitude),
	).Do(); err != nil {
		return err
	}
	return nil
}

func (app *KitchenSink) handleSticker(message *linebot.StickerMessage, replyToken string) error {
	if _, err := app.bot.ReplyMessage(
		replyToken,
		linebot.NewStickerMessage(message.PackageID, message.StickerID),
	).Do(); err != nil {
		return err
	}
	return nil
}

func (app *KitchenSink) replyText(replyToken, text string) error {
	if _, err := app.bot.ReplyMessage(
		replyToken,
		linebot.NewTextMessage(text),
	).Do(); err != nil {
		return err
	}
	return nil
}

func (app *KitchenSink) handleHeavyContent(messageID string, callback func(*os.File) error) error {
	content, err := app.bot.GetMessageContent(messageID).Do()
	if err != nil {
		return err
	}
	defer content.Content.Close()
	log.Printf("Got file: %s", content.ContentType)
	originalConent, err := app.saveContent(content.Content)
	if err != nil {
		return err
	}
	return callback(originalConent)
}

func (app *KitchenSink) saveContent(content io.ReadCloser) (*os.File, error) {
	file, err := ioutil.TempFile(app.downloadDir, "")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = io.Copy(file, content)
	if err != nil {
		return nil, err
	}
	log.Printf("Saved %s", file.Name())
	return file, nil
}
func firstWords(value string, count int) string {
    // Loop over all indexes in the string.
    for i := range value {
        // If we encounter a space, reduce the count.
        if value[i] == ' ' {
            count -= 1
            // When no more words required, return a substring.
            if count == 0 {
                return value[0:i]
            }
        }
    }
    // Return the entire string.
    return value
}
