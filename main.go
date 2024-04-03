package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type basicInfo struct {
	cookie    string
	sessionId string
}

func getBasicInfo(client *http.Client) basicInfo {
	link := "https://courses.erppub.osu.edu/psc/ps/EMPLOYEE/PUB/c/COMMUNITY_ACCESS.CLASS_SEARCH.GBL"

	// 创建一个新的HTTP GET请求
	req, err := http.NewRequest("POST", link, nil)
	if err != nil {
		panic(err)
	}
	// 使用传入的http.Client执行请求
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// 获取cookie信息
	cookies := resp.Cookies()
	cookieStr := ""
	if len(cookies) >= 2 {
		cookieStr = buildCookieString(cookies[1]) + buildCookieString(cookies[2])
	}

	// 读取响应主体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// 解析HTML文档
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		panic(err)
	}

	// 查找ICSID的值
	sessionId, found := findInputValue(doc, "ICSID")
	if !found {
		fmt.Println("Unable to read ICSID")
	}

	result := basicInfo{cookie: cookieStr, sessionId: sessionId}
	return result
}

func findInputValue(n *html.Node, inputName string) (string, bool) {
	if n.Type == html.ElementNode && n.Data == "input" {
		var name, value string
		for _, a := range n.Attr {
			if a.Key == "name" && a.Val == inputName {
				name = a.Val
			} else if a.Key == "value" {
				value = a.Val
			}
		}
		if name != "" {
			return value, true
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if value, found := findInputValue(c, inputName); found {
			return value, true
		}
	}
	return "", false
}

func buildCookieString(cookie *http.Cookie) string {
	return cookie.Name + "=" + cookie.Value + "; "
}

type info struct {
	term           int
	campus         string
	courseNumber   string
	lastName       string
	fullName       string
	courseFullName string
	department     string
}

func sendRequest(action string, info info, cookie basicInfo, client *http.Client) bool {
	link := "https://courses.erppub.osu.edu/psc/ps/EMPLOYEE/PUB/c/COMMUNITY_ACCESS.CLASS_SEARCH.GBL"

	data := url.Values{}
	data.Set("ICAction", action)
	data.Set("ICSID", cookie.sessionId)

	switch action {
	case "CLASS_SRCH_WRK2_STRM$35$":
		data.Set("CLASS_SRCH_WRK2_STRM$35$", strconv.Itoa(info.term))
	case "SSR_CLSRCH_WRK_CAMPUS":
		data.Set("CLASS_SRCH_WRK2_STRM$35$", strconv.Itoa(info.term))
		data.Set("SSR_CLSRCH_WRK_CAMPUS$0", info.campus)
	case "DERIVED_CLSRCH_SSR_EXPAND_COLLAPS$149$$2":
		data.Set("CLASS_SRCH_WRK2_STRM$35$", strconv.Itoa(info.term))
		data.Set("SSR_CLSRCH_WRK_CAMPUS$0", info.campus)
		data.Set("SSR_CLSRCH_WRK_CATALOG_NBR$2", info.courseNumber)
		data.Set("SSR_CLSRCH_WRK_SSR_OPEN_ONLY$chk$4", "N")
	}

	r, _ := http.NewRequest(http.MethodPost, link, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("cookie", cookie.cookie)
	resp, _ := client.Do(r)
	//
	//defer resp.Body.Close()
	//
	//body, _ := io.ReadAll(resp.Body)
	//fmt.Println(string(body))
	if resp.StatusCode == 200 {
		return true
	} else {
		return false
	}
}

type section struct {
	sectionNumber       string
	sectionName         string
	sectionDate         string
	sectionRoom         string
	sectionInstructor   string
	meetingDates        string
	sectionAvailability string
}

func resultProcess(link string, info info, cookie basicInfo) ([]section, bool) {
	var sections []section
	c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
		r.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Headers.Set("cookie", cookie.cookie)
	})

	requestData := map[string]string{
		"ICAction":                   "CLASS_SRCH_WRK2_SSR_PB_CLASS_SRCH",
		"ICSID":                      cookie.sessionId,
		"SSR_CLSRCH_WRK_LAST_NAME$8": info.lastName,
		// Instructor Last Name exactly
		"SSR_CLSRCH_WRK_SSR_EXACT_MATCH2$8": "E",
		// Show Open Classes Only close
		"SSR_CLSRCH_WRK_SSR_OPEN_ONLY$chk$4": "N",
	}

	c.OnHTML("#win0div\\$ICField48\\$0", func(e *colly.HTMLElement) {
		e.ForEach("div[id^=win0divSSR_CLSRSLT_WRK_GROUPBOX3]", func(i int, h *colly.HTMLElement) {
			result := section{
				sectionNumber:       h.ChildText("span[id^=MTG_CLASS_NBR]"),
				sectionName:         strings.ReplaceAll(h.ChildText("span[id^=MTG_CLASSNAME]"), "\n", " "),
				sectionDate:         h.ChildText("span[id^=MTG_DAYTIME]"),
				sectionRoom:         h.ChildText("span[id^=MTG_ROOM]"),
				meetingDates:        h.ChildText("span[id^=MTG_TOPIC]"),
				sectionInstructor:   info.fullName,
				sectionAvailability: h.ChildAttr("div[id^=win0divDERIVED_CLSRCH_SSR_STATUS_LONG] > div > img", "alt"),
			}
			sections = append(sections, result)
		})
	})
	c.OnError(func(response *colly.Response, err error) {
		fmt.Println(err)
	})

	err := c.Post(link, requestData)
	if err != nil {
		fmt.Println(err)
	}
	found := false
	if len(sections) != 0 {
		found = true
	}

	go addToDatabase(sections, info)

	return sections, found

}

func addToDatabase(sections []section, info info) {
	for _, data := range sections {
		number, _ := strconv.Atoi(data.sectionNumber)
		if checkSectionIdExist(number) {
			err := updatePsgSectionInstructor(number, data.sectionInstructor)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			err := insertPsg(
				dbStruct{sectionId: number,
					courseName:          info.courseFullName,
					department:          info.department,
					sectionDate:         data.sectionDate,
					sectionRoom:         data.sectionRoom,
					sectionInstructor:   info.fullName,
					meetingPeriod:       data.meetingDates,
					sectionAvailability: data.sectionAvailability,
					term:                info.term,
					termText:            parseTerm(info.term),
				})
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func updateSameSubject(info info, cookie basicInfo, client *http.Client) {
	link := "https://courses.erppub.osu.edu/psc/ps/EMPLOYEE/PUB/c/COMMUNITY_ACCESS.CLASS_SEARCH.GBL"
	sendRequest("CLASS_SRCH_WRK2_STRM$35$", info, cookie, client)
	sendRequest("SSR_CLSRCH_WRK_CAMPUS", info, cookie, client)
	sendRequest("DERIVED_CLSRCH_SSR_EXPAND_COLLAPS$149$$2", info, cookie, client)
	go resultProcess(link, info, cookie)

}

type Instructor struct {
	Model  string `json:"model"`
	PK     int    `json:"pk"`
	Fields struct {
		FirstName  string   `json:"first_name"`
		LastName   string   `json:"last_name"`
		Department []string `json:"department"`
	} `json:"fields"`
}

func getDepartment(chosen string, term int, campus string, courseNumber string) {
	jsonFile, err := os.Open("instructors.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer jsonFile.Close()

	// 读取文件内容
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// 初始化Instructor切片
	var instructors []Instructor

	// 解析JSON数据到结构体切片
	json.Unmarshal(byteValue, &instructors)
	client := &http.Client{}

	// 遍历instructors切片，筛选department为"CSE"的记录
	//count := 1
	//cookies := getBasicInfo()
	for _, instructor := range instructors {
		for _, dept := range instructor.Fields.Department {
			if dept == chosen {
				go updateSameSubject(info{lastName: instructor.Fields.LastName, term: term, campus: campus, courseNumber: courseNumber, fullName: instructor.Fields.FirstName + " " + instructor.Fields.LastName, courseFullName: chosen + " " + courseNumber, department: chosen}, getBasicInfo(client), client)
			}
		}
	}
}
func main() {
	getDepartment("CSE", 1248, "COL", "2421")
	//updateSameSubject(info{lastName: "JONES", term: 1242, campus: "COL", courseNumber: 2421, fullName: "Janis Jones"}, getBasicInfo())
}
