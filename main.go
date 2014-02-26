package main

import (
	"database/sql"
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/render"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

// 20060102 15:04:05

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

func renderPages(days int, memoreyCache map[int]string) map[int]FinalData {
	pages := make(map[int]FinalData)
	var pagemark []int
	date := time.Now()

	for i := 1; i <= len(memoreyCache)/days; i += 1 {
		pagemark = append(pagemark, i)
	}

	for i := 1; i <= len(memoreyCache)/days; i += 1 {

		var finaldata FinalData
		var useddata []UsedData

		for j := 0; j < days; j++ {
			key := date.Format("20060102")

			data, ok := memoreyCache[atoi(key)]
			if !ok {
				data = getBeforeData(key)
				memoreyCache[atoi(key)] = data
			}

			useddata = append(useddata, zhihuDailyJson(data))
			date = date.AddDate(0, 0, -1)
		}
		finaldata.Useddata = useddata
		finaldata.Pagemark = pagemark
		pages[i] = finaldata
	}

	return pages
}

func updatePages(pages map[int]FinalData) {

	updatepage := 1

	oriUseddata := pages[updatepage].Useddata
	oriPagemark := pages[updatepage].Pagemark

	var finaldata FinalData
	var useddata []UsedData

	useddata = append(useddata, zhihuDailyJson(todayData()))
	index := 0
	if len(oriUseddata) > 4 {
		index = 1
	}
	useddata = append(useddata, oriUseddata[index:]...)

	finaldata.Useddata = useddata
	finaldata.Pagemark = oriPagemark

	delete(pages, updatepage)

	pages[updatepage] = finaldata
}

func atoi(s string) int {
	dateInt, _ := strconv.Atoi(s)
	return dateInt
}

func autoUpdate() map[int]FinalData {

	// init
	days := 4
	memoreyCache := QueryData()
	pages := renderPages(days, memoreyCache)

	updatePages(pages)
	lastUpdate := time.Now()

	ticker := time.NewTicker(time.Hour) // update every per hour
	go func() {
		for t := range ticker.C {

			// Tomorrow 7 am
			if tomorrowSeven(time.Now(), lastUpdate) {
				lastUpdate = t
				pages = nil
				pages = renderPages(days, memoreyCache)
				//fmt.Println("autoRenderPages at", t)
			}

			updatePages(pages)
			//fmt.Println("autoUpdate at", t)
		}
	}()

	return pages
}

func tomorrowSeven(now time.Time, lastUpdate time.Time) bool {
	return (time.Now().Format("02") > lastUpdate.Format("02")) && (time.Now().Format("15") > lastUpdate.Format("15"))
}

func main() {
	//fmt.Println("main()")

	pages := autoUpdate()

	m := martini.Classic()
	m.Use(martini.Static("static"))
	m.Use(render.Renderer())

	m.Get("/", func(r render.Render) {

		r.HTML(200, "content", []interface{}{pages[1]})
	})

	m.Get("/page/:id", func(params martini.Params, r render.Render) {

		id := atoi(params["id"])
		r.HTML(200, "content", []interface{}{pages[id]})
	})

	m.Get("/url/**", func(params martini.Params, r render.Render) {

		r.HTML(200, "share_image", params["_1"])
	})

	http.ListenAndServe("0.0.0.0:8000", m)
	m.Run()
}

// -------------------DB----------------------
func getData(url string) string {
	resp, err := http.Get(url)

	if err != nil {
		// handle error
		fmt.Println(err)
		return ""
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return string(body)
}

func getAllData() {
	date, _ := time.Parse("20060102", "20140209")
	firstDate, _ := time.Parse("20060102", "20130520")

	for ; date.After(firstDate); date = date.AddDate(0, 0, -1) {
		getBeforeData(date.Format("20060102"))
	}

}

func getBeforeData(date string) string {
	url := "http://news.at.zhihu.com/api/1.2/news/before/" + date
	data := getData(url)

	writeToDB(atoi(date), data)

	return data
}

func todayData() string {
	url := "http://news.at.zhihu.com/api/1.2/news/latest"

	return getData(url)
}

func InitDB() {
	db, err := sql.Open("sqlite3", "./main.db")
	checkErr(err)
	//插入数据
	stmt, err := db.Prepare("CREATE TABLE `datainfo` (`date` INTEGER PRIMARY KEY, `data` TEXT NULL)")
	checkErr(err)

	stmt.Exec()

	db.Close()
}

func QueryData() map[int]string {

	memoryCache := make(map[int]string)

	db, err := sql.Open("sqlite3", "./main.db")
	checkErr(err)

	rows, err := db.Query("SELECT * FROM datainfo")
	checkErr(err)

	db.Close()

	for rows.Next() {
		var date int
		var data string
		err = rows.Scan(&date, &data)
		memoryCache[date] = data
	}

	return memoryCache
}

func writeToDB(date int, data string) {

	db, err := sql.Open("sqlite3", "./main.db")
	checkErr(err)
	//插入数据
	stmt, err := db.Prepare("INSERT INTO datainfo(date, data) values(?,?)")
	checkErr(err)

	res, err := stmt.Exec(date, data)
	checkErr(err)

	id, err := res.LastInsertId()
	checkErr(err)

	fmt.Println(id)

	db.Close()
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
