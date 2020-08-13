package main

import (
	"adsmall-v2/api-item/config"
	"adsmall-v2/api-item/controllers"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
	"github.com/gin-gonic/gin"
)

func main() {
	err := config.CheckRequiredEnv()
	if err != nil {
		panic(err)
	}

	db := config.DBInit()
	inDB := &controllers.InDB{DB: db}

	gin.SetMode(config.GetJSON("app.mode"))

	now := time.Now() //or time.Now().UTC()
	logFileName := "access_" + now.Format("2006-01-02") + ".log"
	logFile, err := os.OpenFile("log/"+logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	if config.GetJSON("app.mode") == gin.ReleaseMode {
		gin.DefaultWriter = io.MultiWriter(logFile)
	} else {
		gin.DefaultWriter = io.MultiWriter(logFile, os.Stdout)
	}

	router := gin.New()
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s | %s | %d | %s | %s %s | %s \n",
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			"Authorization: "+param.Request.Header.Get("Authorization"),
			param.Request.Form,
			param.ErrorMessage,
		)
	}))
	router.Use(gin.Recovery())
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	api := router.Group("/v2")

	contributionRoute := api.Group("contribution")
	contributionRoute.GET("/", inDB.GetListContribution)
	contributionRoute.GET("/:contribution_id", inDB.GetDetailContribution)
	contributionRoute.POST("/", inDB.CreateContribution)
	contributionRoute.PATCH("/:contribution_id", inDB.UpdateContribution)
	contributionRoute.DELETE("/:contribution_id", inDB.DeleteContribution)

	itemRoute := api.Group("item")
	itemRoute.PATCH("/:item_id", inDB.UpdateItem)
	itemRoute.DELETE("/:item_id", inDB.DeleteItem)

	s := &http.Server{
		Addr:           ":" + config.GetEnv("API_ITEM_PORT"),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	err = s.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
