package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/render"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// 20060102 15:04:05

type UsedData struct {
	Date      string
	MainPages []MainPage
}

type MainPage struct {
	Id    int
	Title string
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

	var mainpages []MainPage

	fout, _ := os.OpenFile(SHAREIMAGE, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)

	for _, a := range news {
		m := a.(map[string]interface{})

		shareimage := m["share_image"].(string)

		fout.WriteString(shareimage + "\n")

		url := m["url"].(string)
		id := atoi(url[strings.LastIndexAny(url, "/")+1:])

		title := m["title"].(string)

		mainpages = append(mainpages, MainPage{id, title})
	}

	defer fout.Close()

	return UsedData{Date: date, MainPages: mainpages}
}

func renderPages(days int) map[int]FinalData {

	os.Remove(SHAREIMAGE)

	pages := make(map[int]FinalData)
	var pagemark []int
	date := time.Now()

	memoreyCache := QueryData()

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

func autoUpdate() map[int]FinalData {

	// init
	days := 4
	pages := renderPages(days)

	ticker := time.NewTicker(time.Hour) // update every per hour
	go func() {
		for t := range ticker.C {
			fmt.Println("renderPages at ", t)
			pages = renderPages(days)
		}
	}()

	return pages
}

var IMG = "static/img/"
var SHAREIMAGE = "shareimage.txt"

func main() {

	pages := autoUpdate()

	downloadAll()

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

	http.ListenAndServe("0.0.0.0:8000", m)
	m.Run()
}

// -------------Download---------------

func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func downloadAll() {
	os.MkdirAll(IMG, 0755)

	var imgurl []string

	file, err := os.Open(SHAREIMAGE)
	defer file.Close()
	if nil == err {
		buff := bufio.NewReader(file)

		for {
			url, err := buff.ReadString('\n')
			url = strings.TrimSpace(url)
			if err != nil || io.EOF == err {
				break
			}
			//fmt.Println(strings.Replace(url, "http://d0.zhimg.com/", "", 1))

			//download(url)
			imgurl = append(imgurl, url)
		}
	}

	muiltDownload(imgurl, 8)
}

func muiltDownload(urls []string, threads int) {
	if threads == 1 {
		go func() {
			for _, url := range urls {
				download(url)
			}
			//fmt.Println(len(urls))
		}()
	} else {
		threads /= 2
		mid := len(urls) / 2
		//fmt.Println(threads, mid)
		muiltDownload(urls[:mid], threads)
		muiltDownload(urls[mid:], threads)
	}

}

func download(url string) {

	str := strings.Replace(url, "http://d0.zhimg.com/", "", 1)
	index := strings.LastIndexAny(str, "/")

	if index > -1 {
		filename := strings.Replace(str, "/", "_", 1)
		//fmt.Println(path, filename)

		if !Exist(IMG + filename) {

			resp, err := http.Get(url)
			checkErr(err)

			defer resp.Body.Close()

			file, err := os.Create(IMG + filename)
			checkErr(err)

			io.Copy(file, resp.Body)

			fmt.Println("download: " + url)
		} else {
			fmt.Println("skip: " + url)
		}
	}

}

// -------------------DB----------------------

func getData(url string) string {
	resp, err := http.Get(url)
	checkErr(err)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return string(body)
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
