package mydatabase

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func main() {

	// QueryData()

}

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

		fmt.Println(url)
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
