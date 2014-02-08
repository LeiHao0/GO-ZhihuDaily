package main

import (
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/render"
	"go-zhihudaily/src"
)

type UsedData struct {
	Date string
	News []interface{}
}

func zhihuDailyJson(str string) UsedData {

	sj, _ := simplejson.NewJson([]byte(str))
	date := sj.Get("date").MustString()
	news, _ := sj.Get("news").Array()

	return UsedData{Date: date, News: news}
}

func main() {
	fmt.Println("start main()")

	memoreyCache := mydatabase.QueryData()

	useddata := zhihuDailyJson(memoreyCache[20140207])

	finalData := []interface{}{useddata.Date, useddata.News}

	m := martini.Classic()
	m.Use(martini.Static("static"))
	m.Use(render.Renderer())

	m.Get("/", func(r render.Render) {
		r.HTML(200, "content", finalData[1])
	})
	// m.Get("/:id", func(params martini.Params, r render.Render) {
	// 	r.HTML(200, "share_image", params["id"])
	// })

	m.Run()
}
