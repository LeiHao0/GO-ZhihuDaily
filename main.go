/*
	My dear friend,

	When I wrote this, God and I knew what it meant.

	Now, God only knows.

	CC @Artwalk
*/

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shxsun/go-sh"
)

// Golang FormatTime: 20060102 15:04:05

type UsedData struct {
	Date      string
	MainPages []MainPage
}

type MainPage struct {
	Id         int // story id
	Title      string
	ShareImage string // download img
}

type FinalData struct {
	Useddata []UsedData
	Pagemark []string
}

var IMG = "static/img/"

//------------------------------------Main------------------------------------------

func main() {

	autoUpdate()

	router := gin.Default()

	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")

	router.LoadHTMLGlob("templates/*")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", []interface{}{getPage(1)})
	})

	router.GET("/page/:id", func(c *gin.Context) {
		s := strings.Trim(c.Param("id"), " .)(")
		id := atoi(s)
		c.HTML(http.StatusOK, "index.tmpl", []interface{}{getPage(id)})
	})

	router.GET("/api/:id", func(c *gin.Context) {
		s := strings.Trim(c.Param("id"), " .)(")
		id := atoi(s)
		c.JSON(http.StatusOK, []interface{}{getPage(id)})
	})

	router.Run(":8080")
}

//------------------------------------Pages------------------------------------------

func zhihuDailyJson(str string) UsedData {

	sj, _ := simplejson.NewJson([]byte(str))

	tmp, _ := time.Parse("20060102", sj.Get("date").MustString())
	date := tmp.Format("2006.01.02 Monday")

	news, _ := sj.Get("news").Array()

	var mainpages []MainPage
	var shareimageurl, shareimage, title string

	for _, a := range news {
		m := a.(map[string]interface{})

		url := m["url"].(string)
		id := atoi(url[strings.LastIndexAny(url, "/")+1:])

		if m["share_image"] != nil {
			shareimageurl = m["share_image"].(string)
		} else { // new api do not provide share_imag
			title = m["title"].(string)
			shareimageurl = m["image"].(string)
		}

		shareimage = shareImgUrlToFilename(shareimageurl)
		mainpages = append(mainpages, MainPage{id, title, shareimage})
	}

	return UsedData{Date: date, MainPages: mainpages}
}

func renderPages(days int) {

	var newMainPages []MainPage
	var finaldata FinalData

	date := time.Now()

	if date.Format("MST") == "UTC" {
		date = date.Add(time.Hour * 8)
	}

	memoreyCache := QueryDateData()
	memoreyCacheLen := len(memoreyCache) / days
	flag := memoreyCacheLen
	if flag <= 0 {
		// first init & download all data
		getAllData()

		memoreyCache = QueryDateData()
		memoreyCacheLen = len(memoreyCache) / days
	}

	for i := 1; i <= memoreyCacheLen; i += 1 {

		var pagemark []string
		if i-10 > 1 {
			pagemark = append(pagemark, "1    ...  ")
		}
		for k, j := 0, i-10; k <= 20; k++ {
			if j > 0 && j <= memoreyCacheLen {
				s := itoa(j)
				if j == i {
					s = "( " + s + " )"
				}
				pagemark = append(pagemark, s)
			}
			j++
		}
		if i < memoreyCacheLen-10 {
			pagemark = append(pagemark, "  ...    "+itoa(memoreyCacheLen))
		}

		var useddata []UsedData

		if i == 1 && date.Format("15") > "07" {
			if str := todayData(); str != "" {
				todaydata := zhihuDailyJson(str)
				useddata = append(useddata, todaydata)

				newMainPages = append(newMainPages, todaydata.MainPages...)
			}
		}

		for j := 0; j < days; j++ {
			key := date.Format("20060102")

			data, ok := memoreyCache[atoi(key)] // get from db
			if !ok {                            // get from zhihu
				// no Response， skip this day
				if data = getBeforeData(key); data == "" {
					break
				}
			}

			beforeday := zhihuDailyJson(data)

			// remove /*flag*/ comment if you want to download all images
			if /*flag <= 0 || (*/ i == 1 && j == 0 /*)*/ {
				newMainPages = append(newMainPages, beforeday.MainPages...)
			} // end
			useddata = append(useddata, beforeday)

			date = date.AddDate(0, 0, -1)
		}

		finaldata.Pagemark = pagemark
		finaldata.Useddata = useddata

		if pageJson, err := json.Marshal(finaldata); err == nil {
			page := string(pageJson)
			writeToPageDB(i, page)
		}
	}

	memoreyCache = nil

	downloadDayShareImg(newMainPages)
}

func autoUpdate() {

	// init
	days := 2
	renderPages(days)

	ticker := time.NewTicker(time.Hour) // update every per hour
	go func() {
		for t := range ticker.C {
			fmt.Println("renderPages at ", t)
			renderPages(days)
		}
	}()

}

func getPage(index int) FinalData {
	var finaldata FinalData
	data := QueryPageData(index)
	json.Unmarshal([]byte(data), &finaldata)

	return finaldata
}

// ----------------------------Download----------------------------------------------

func getAllData() {
	date := time.Now()
	firstDate, _ := time.Parse("20060102", "20130520")

	for ; date.After(firstDate); date = date.AddDate(0, 0, -1) {
		getBeforeData(date.Format("20060102"))
	}

}

func Exist(filename string) bool {

	err := syscall.Access(filename, syscall.F_OK)
	return err == nil
}

func download(url string) {
	filename := shareImgUrlToFilename(url)
	index := strings.LastIndexAny(filename, "_")

	if index > -1 && !Exist(IMG+"croped/"+filename) {

		if resp, err := http.Get(url); err == nil {
			defer resp.Body.Close()

			if file, err := os.Create(IMG + filename); err == nil {
				io.Copy(file, resp.Body)
				cropImage(filename)
			}
		}
	}
}

func downloadDayShareImg(mainpages []MainPage) {

	// 8 thread
	nbConcurrentGet := 8
	urls := make(chan string, nbConcurrentGet)
	//var wg sync.WaitGroup
	for i := 0; i < nbConcurrentGet; i++ {
		go func() {
			for url := range urls {
				download(url)
				//wg.Done()
			}
		}()
	}
	for _, mainpage := range mainpages {
		//wg.Add(1)
		urls <- fmt.Sprintf(filenameToShareImgUrl(mainpage.ShareImage))
	}

	//wg.Wait()
}

func cropImage(filename string) {
	session := sh.NewSession()
	session.Command("convert", filename, "-resize", "440>", "-crop", "x275+0+0", "croped/"+filename, sh.Dir(IMG)).Run()
	session.Command("rm", IMG+filename).Run()
}

func getData(url string) string {

	if resp, err := http.Get(url); err == nil {
		defer resp.Body.Close()
		if body, err := ioutil.ReadAll(resp.Body); err == nil {
			return string(body)
		}
	}

	return ""
}

// --------------------------------DataBase------------------------------------------

func getBeforeData(date string) string {
	url := "http://news.at.zhihu.com/api/1.2/news/before/" + date
	data := getData(url)

	writeToDateDB(atoi(date), data)

	return data
}

func todayData() string {
	url := "http://news.at.zhihu.com/api/1.2/news/latest"

	return getData(url)
}

func QuerryData(table string) *sql.Rows {
	db, err := sql.Open("sqlite3", "./main.db")
	checkErr(err)

	rows, err := db.Query("SELECT * FROM " + table)
	checkErr(err)

	db.Close()

	return rows
}

func QueryDateData() map[int]string {

	rows := QuerryData("dateinfo")

	var date int
	var data string

	memoryCache := make(map[int]string)
	for rows.Next() {
		if err := rows.Scan(&date, &data); err == nil {
			memoryCache[date] = data
		}
	}

	return memoryCache
}

func QueryPageData(index int) string {

	rows := QuerryData("pageinfo")

	var id int
	var data string
	page := ""

	for rows.Next() {
		if err := rows.Scan(&id, &data); err == nil && id == index {
			page = data
		}
	}

	return page
}

func writeToDB(table string, id int, data string) {

	db, err := sql.Open("sqlite3", "./main.db")
	checkErr(err)

	stmt, err := db.Prepare("REPLACE INTO " + table + "(id, data) values(?,?)")
	checkErr(err)

	res, err := stmt.Exec(id, data)
	checkErr(err)

	index, err := res.LastInsertId()
	checkErr(err)

	fmt.Println(index)

	db.Close()
}

func writeToDateDB(date int, data string) {
	writeToDB("dateinfo", date, data)
}

func writeToPageDB(index int, data string) {
	writeToDB("pageinfo", index, data)
}

// -----------------------------------Tools------------------------------------------
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func atoi(s string) int {
	dateInt, _ := strconv.Atoi(s)
	return dateInt
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func idToUrl(id int) string {
	return "http://daily.zhihu.com/api/1.2/news/" + itoa(id)
}

func filenameToShareImgUrl(filename string) string {
	url := strings.Replace(filename, "_", "/", -1)
	url = strings.Replace(url, "-", ":", -1)

	return url
}

func shareImgUrlToFilename(shareImgUrl string) string {
	filename := strings.Replace(shareImgUrl, "/", "_", -1)
	filename = strings.Replace(filename, ":", "-", -1)

	return filename
}
