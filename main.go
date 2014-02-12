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

func renderPages(tatal int) map[int]FinalData {
	memoreyCache := QueryData()

	page := make(map[int]FinalData)

	var pagemark []int
	for i := len(memoreyCache) / tatal; i > 0; i -= 1 {
		pagemark = append(pagemark, i)
	}

	date := time.Now()

	i := len(memoreyCache) / tatal
	for ; i > 0; i -= 1 {
		var finaldata FinalData
		var useddata []UsedData
		for j := 0; j < tatal; j++ {
			temp, _ := strconv.Atoi(date.Format("20060102"))

			data, ok := memoreyCache[temp]
			if ok {
				useddata = append(useddata, zhihuDailyJson(data))
			} else {
				url := "http://news.at.zhihu.com/api/1.2/news/before/" + date.Format("20060102")

				data = getData(url)
				dateInt, _ := strconv.Atoi(date.Format("20060102"))
				writeToDB(dateInt, data)
				useddata = append(useddata, zhihuDailyJson(data))
			}
			date = date.AddDate(0, 0, -1)
		}
		finaldata.Useddata = useddata
		finaldata.Pagemark = pagemark
		page[i] = finaldata
	}

	return page
}

func initDB() {
	InitDB()
	GetBeforeData()
}

func main() {
	// initDB()
	pages := renderPages(3)

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

	// fmt.Println(memoryCache[20131212])
	return memoryCache
}

func GetBeforeData() {

	// string -> time
	date, _ := time.Parse("20060102", "20140209")
	firstDate, _ := time.Parse("20060102", "20130520")

	for ; date.After(firstDate); date = date.AddDate(0, 0, -1) {

		url := "http://news.at.zhihu.com/api/1.2/news/before/" + date.Format("20060102")

		data := getData(url)
		dateInt, _ := strconv.Atoi(date.Format("20060102"))
		writeToDB(dateInt, data)
	}
}

func TodayData() string {
	// today := time.Now().Format("20060102")
	url := "http://news.at.zhihu.com/api/1.2/news/latest"

	return getData(url)
}

func InitDB() {
	db, err := sql.Open("sqlite3", "./main.db")
	checkErr(err)
	//插入数据
	stmt, _ := db.Prepare("CREATE TABLE `datainfo` (`date` INTEGER PRIMARY KEY, `data` TEXT NULL)")
	checkErr(err)

	stmt.Exec()
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
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
