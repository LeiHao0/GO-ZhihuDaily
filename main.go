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

		if i == 1 {
			useddata = append(useddata, zhihuDailyJson(todayData()))
		}

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

func atoi(s string) int {
	dateInt, _ := strconv.Atoi(s)
	return dateInt
}

func main() {

	memoreyCache := QueryData()
	days := 4

	pages := renderPages(days, memoreyCache)

	m := martini.Classic()
	m.Use(martini.Static("static"))
	m.Use(render.Renderer())

	lastUpdate := time.Now()

	m.Get("/", func(r render.Render) {
		if time.Since(lastUpdate) > (time.Hour * 2) {
			lastUpdate = time.Now()
			pages = renderPages(days, memoreyCache)
		}
		r.HTML(200, "content", []interface{}{pages[1]})
	})

	m.Get("/date/:id", func(params martini.Params, r render.Render) {

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
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return string(body)

}

func QueryData() map[int]string {

	memoryCache := make(map[int]string)

	db, err := sql.Open("sqlite3", "./main.db")
	checkErr(err)

	rows, err := db.Query("SELECT * FROM datainfo")
	checkErr(err)

	for rows.Next() {
		var date int
		var data string
		err = rows.Scan(&date, &data)
		memoryCache[date] = data
	}

	return memoryCache
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
