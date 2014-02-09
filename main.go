package main

import (
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/render"
	// "github.com/codegangsta/martini-contrib/strip"
	"go-zhihudaily/src"
	"strconv"
	"time"
)

type UsedData struct {
	Date string
	News []interface{}
}

type FinalData struct {
	Useddata []UsedData
	Pagemark []int
}

func zhihuDailyJson(str string) UsedData {

	sj, _ := simplejson.NewJson([]byte(str))

	news, _ := sj.Get("news").Array()
	tmp, _ := time.Parse("20060102", sj.Get("date").MustString())
	date := tmp.Format("2006.01.02 Monday")

	return UsedData{Date: date, News: news}
}

func renderPages() map[int]FinalData {
	memoreyCache := mydatabase.QueryData()

	page := make(map[int]FinalData)

	var pagemark []int
	for i := len(memoreyCache) / 7; i > 0; i -= 1 {
		pagemark = append(pagemark, i)
	}

	date := time.Now()

	i := len(memoreyCache) / 7
	for ; i > 0; i -= 1 {
		var finaldata FinalData
		var useddata []UsedData
		for j := 0; j < 7; j++ {
			temp, _ := strconv.Atoi(date.Format("20060102"))
			useddata = append(useddata, zhihuDailyJson(memoreyCache[temp]))
			date = date.AddDate(0, 0, -1)
		}
		finaldata.Useddata = useddata
		finaldata.Pagemark = pagemark
		page[i] = finaldata
	}

	return page
}

func initDB() {
	mydatabase.InitDB()
	mydatabase.GetBeforeData()
}

func main() {
	fmt.Println("start main()")
	// initDB()
	pages := renderPages()

	m := martini.Classic()
	m.Use(martini.Static("static"))
	m.Use(render.Renderer())

	// todayData := zhihuDailyJson(mydatabase.TodayData())
	// finalData = []interface{}{todayData}

	m.Get("/", func(r render.Render) {
		r.HTML(200, "content", []interface{}{pages[len(pages)]})
	})

	m.Get("/date/:id", func(params martini.Params, r render.Render) {

		id, _ := strconv.Atoi(params["id"])
		fmt.Println("id", id)
		r.HTML(200, "content", []interface{}{pages[id]})
	})

	// each title
	m.Get("/url/**", func(params martini.Params, r render.Render) {

		// fmt.Println(params["_1"])

		r.HTML(200, "share_image", params["_1"])
	})

	// m.Get("/**/*.css", strip.Prefix("/**"), m.ServeHTTP)
	// end test strip

	m.Run()
}
