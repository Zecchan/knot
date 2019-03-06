package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/eaciit/toolkit"
	"github.com/zecchan/knot/knot.v1"
)

var (
	//appViewsPath = "/Users/ariefdarmawan/goapp/src/github.com/eaciit/knot/example/hello/views/"
	appViewsPath = func() string {
		d, _ := os.Getwd()
		return d
	}() + "/../rtchat/views/"
)

type Chat struct {
}

func (h *Chat) Index(r *knot.WebContext) interface{} {
	r.Config.ViewName = "chat.html"
	return (toolkit.M{}).Set("message", "This data is sent to knot controller method")
}

func (h *Chat) Assets(r *knot.WebContext) interface{} {
	r.Config.OutputType = knot.OutputByte
	var path = appViewsPath + "assets/" + r.Query("name")
	//r.Writer.Header().Set("Content-Disposition", "attachment; filename="+r.Query("name"))

	_, e := os.Stat(path)
	if e == nil {
		ext := filepath.Ext(path)
		ext = ext[1:len(ext)]
		r.Writer.Header().Set("Content-Type", "text/"+ext)

		f, err := os.Open(path)
		if err != nil {
			fmt.Println(err.Error())
		}

		io.Copy(r.Writer, f)

		return ""
	}
	return "File not exists"
}

func (h *Chat) Ws(r *knot.WebContext) interface{} {
	fmt.Println("Client is connecting to WebSocket: " + r.Request.Host)
	err := r.WebSocket.Upgrade(r, h.handleChatClientStream)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("Client has disconnected from WebSocket: " + r.Request.Host)
	return ""
}

func (h *Chat) handleChatClientStream(context *knot.WebSocketContext) error {
	fmt.Println("Client is connected to WebSocket: " + context.WebContext.Request.Host)

	// Detector channel
	var dc = make(chan error, 2)

	// Read broadcast goroutine
	go func() {
		for !context.ExitDesired {
			bcData, ok := context.PopBroadcastData()
			for ok {
				// Check data context
				if bcData.Context == "ChatData" {
					chatData, ok := bcData.Data.(ChatData)
					if ok {
						fmt.Println("MSG: <= " + chatData.Data)
						err := context.Socket.WriteJSON(chatData)
						if err != nil {
							// Check if the error is caused by a closed connection
							if context.WebSocket.IsConnectionClosedError(err) {
								dc <- err
								return
							}
						}
					}
				}

				// Exhaust the broadcast queue
				bcData, ok = context.PopBroadcastData()
			}
			// Give other goroutine processing time
			time.Sleep(100)
		}
		dc <- nil
	}()

	// Read user input goroutine
	go func() {
		for !context.ExitDesired {
			chatData := ChatData{}
			// this call is blocking, so this goroutine does not need time.Sleep

			err := context.Socket.ReadJSON(&chatData)
			chatData.Timestamp = time.Now().Format("2006-01-02 15:04")
			if err == nil {
				fmt.Println("MSG: UID#" + chatData.UID + " (" + chatData.Username + ") => " + chatData.Data)
				context.WebSocket.Broadcast("ChatData", chatData, context)
			}
			// Check if the error is caused by a closed connection
			if context.WebSocket.IsConnectionClosedError(err) {
				dc <- err
				return
			}
		}
		dc <- nil
	}()

	// wait for those two goroutine to end
	_ = <-dc
	_ = <-dc

	return nil
}

type ChatData struct {
	Timestamp string
	Username  string
	Data      string
	UID       string
}

func main() {
	app := knot.NewApp("")
	app.ViewsPath = appViewsPath
	app.Register(&Chat{})
	app.LayoutTemplate = "_template.html"
	app.DefaultOutputType = knot.OutputTemplate

	knot.RegisterApp(app)
	knot.StartContainer(&knot.AppContainerConfig{
		Address: "localhost:8080",
	})
}

// try to access http://localhost:1234/hello/index
