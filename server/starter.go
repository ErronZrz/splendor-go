package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.POST("/create/:game", CreateGameRouter)
	r.POST("/join/:game", JoinGameRouter)
	r.POST("/watch/:game", WatchGameRouter)
	r.POST("/start/:game/:starter", StartGameRouter)
	r.POST("/game/:game/chat", ChatRouter)
	r.POST("/game/:game/next", NextGameRouter)
	r.POST("/game/:game/:action/:target", ActionRouter)
	r.POST("/rename/:game/:name", RenamePlayerRouter)
	r.GET("/suggest", SuggestRouter)
	// r.GET("/stat/:game", StatRouter)
	// r.GET("/poll/:game", PollRouter)
	r.GET("/list", ListRouter)

	r.StaticFile("/", "./static/index.html")

	r.GET("/:str/:game", func(c *gin.Context) {
		str := c.Param("str")
		if str == "stat" {
			StatRouter(c)
		} else if str == "poll" {
			PollRouter(c)
		} else {
			c.File("./static/index.html")
		}
	})

	r.GET("/static/*filepath", func(c *gin.Context) {
		c.File("./static" + c.Param("filepath"))
	})

	InitSuggestWords()

	err := r.Run(":8080")
	if err != nil {
		fmt.Println(err)
	}
}
