package main

import (
	"log"
	"time"

	"github.com/deepch/vdk/av"
	"github.com/deepch/vdk/format/mp4f"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

func serveHTTP() {
	router := gin.Default()
	router.GET("/ws/stream", func(c *gin.Context) {
		handler := websocket.Handler(ws)
		handler.ServeHTTP(c.Writer, c.Request)
	})
	router.Static("/web", "./web")
	router.StaticFile("/favicon.ico", "./web/icons/favicon.ico")
	err := router.Run(Config.Server.HTTPPort)
	if err != nil {
		log.Fatalln("Start HTTP Server error", err)
	}
}
func ws(ws *websocket.Conn) {
	defer ws.Close()
	suuid := ws.Request().FormValue("suuid")
	log.Println("Request", suuid)
	if !Config.ext(suuid) {
		log.Println("Stream Not Found")
		return
	}
	Config.RunIFNotRun(suuid)
	ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
	cuuid, ch := Config.clAd(suuid)
	defer Config.clDe(suuid, cuuid)
	codecs := Config.coGe(suuid)
	if codecs == nil {
		log.Println("Codecs Error")
		return
	}
	for i, codec := range codecs {
		if codec.Type().IsAudio() && codec.Type() != av.AAC {
			log.Println("Track", i, "Audio Codec Work Only AAC")
		}
	}
	muxer := mp4f.NewMuxer(nil)
	err := muxer.WriteHeader(codecs)
	if err != nil {
		log.Println("muxer.WriteHeader", err)
		return
	}
	meta, init := muxer.GetInit(codecs)
	err = websocket.Message.Send(ws, append([]byte{9}, meta...))
	if err != nil {
		log.Println("websocket.Message.Send", err)
		return
	}
	err = websocket.Message.Send(ws, init)
	if err != nil {
		return
	}
	var start bool
	go func() {
		for {
			var message string
			err := websocket.Message.Receive(ws, &message)
			if err != nil {
				ws.Close()
				return
			}
		}
	}()
	noVideo := time.NewTimer(10 * time.Second)
	var timeLine = make(map[int8]time.Duration)
	for {
		select {
		case <-noVideo.C:
			log.Println("noVideo")
			return
		case pck := <-ch:
			if pck.IsKeyFrame {
				noVideo.Reset(10 * time.Second)
				start = true
			}
			if !start {
				continue
			}
			timeLine[pck.Idx] += pck.Duration
			pck.Time = timeLine[pck.Idx]
			ready, buf, _ := muxer.WritePacket(pck, false)
			if ready {
				err = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err != nil {
					return
				}
				err := websocket.Message.Send(ws, buf)
				if err != nil {
					return
				}
			}
		}
	}
}
