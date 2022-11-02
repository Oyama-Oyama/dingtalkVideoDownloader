package dingtalk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

const COUNT int = 400
const TOKEN string = "AAAAAKkWBGQRHEWF0Rs2JgxF1CDc-zGXFEBfc1j-ZzDw9DVm1V5x3TwQs_IdxESpYvnENNAR43lMqKyJAhPN9A1WAnZEuEn8Z4-Eawy_pX9PLvL4cJc3Ka9W1OoFMhS2Bje6KoODecI2H9uUQTpotKe0dDA"
const USER_AGENT string = "Mozilla/5.0 (Macintosh; Intel Mac OS X 12_6_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36 DingTalk(6.5.31-macOS-macOS-MAS-25661187) nw Channel/201200"

var source string = "" //"https://qy-api.yxt.com/v1/exams/exercise/subjects/3d49722e-9232-4a26-a024-14fc53af0dc0?limit=48&offset=0&type=0&id=3d49722e-9232-4a26-a024-14fc53af0dc0"

type PageContent struct {
	Data   []Item     `json:"datas"`
	Paging Pagingitem `json:"paging"`
}

type Pagingitem struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Pages  int `json:"pages"`
	Count  int `json:"count"`
}

type Item struct {
	Pid          string     `json:"pid"`
	Description  string     `json:"description"`
	OrgId        string     `json:"orgId"`
	ItemNum      int        `json:"itemNum"`
	Answer       string     `json:"answer"`
	Ttype        int        `json:"type"`
	IsOrder      int        `json:"isOrder"`
	Analysis     string     `json:"analysis"`
	Status       int        `json:"status"`
	Difficulty   int        `json:"difficulty"`
	KnowledgeId  string     `json:"knowledgeId"`
	Creator      string     `json:"creator"`
	CreateTime   string     `json:"createTime"`
	Updater      string     `json:"updater"`
	UpdateTime   string     `json:"updateTime"`
	Attachment   string     `json:"attachment"`
	IsCorrect    int        `json:"isCorrect"`
	SubjectItems []subItems `json:"subjectItems"`
	ErrorAnswer  string     `json:"errorAnswer"`
}

type subItems struct {
	Pid         string `json:"pid"`
	Description string `json:"description"`
	IsCorrect   int    `json:"isCorrect"`
	ItemName    string `json:"itemName"`
	Attachment  string `json:"attachment"`
	SubjectId   string `json:"subjectId"`
}

func init() {
	source = "https://qy-api.yxt.com/v1/exams/exercise/subjects/3d49722e-9232-4a26-a024-14fc53af0dc0?limit=400&offset=0&type=0&id=3d49722e-9232-4a26-a024-14fc53af0dc0"
}

func LoadExercise() {
	request, err := http.NewRequest("GET", source, nil)
	if err != nil {
		panic(err)
	}
	request.Header.Add("Token", TOKEN)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	saveToFile(body)
}

func saveToFile(body []byte) {
	_, err := os.Stat("temp.txt")
	if err != nil {
		os.Remove("temp.txt")
	}
	file, err := os.Create("temp.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	count, err := file.Write(body)
	if err != nil {
		panic(err)
	}
	fmt.Printf("write end, count:%v\n", count)
	ParseFileContent()
}

func ParseFileContent() {

	file, err := os.OpenFile("temp.txt", os.O_RDONLY, 0666)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	var _result PageContent
	err = json.Unmarshal(body, &_result)
	if err != nil {
		panic(err)
	}
	fmt.Printf("common:")
	fmt.Println(len(_result.Data))

	outFile, err := os.OpenFile("result.txt", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()
	defer os.Remove("temp.txt")
	index := 1
	var crroct string
	for _, _item := range _result.Data {
		outFile.WriteString(strconv.Itoa(index))
		outFile.WriteString(".")
		outFile.WriteString(_item.Description)
		outFile.WriteString("\n")
		for _, sub := range _item.SubjectItems {
			outFile.WriteString(sub.ItemName)
			outFile.WriteString(". ")
			outFile.WriteString(sub.Description)
			outFile.WriteString("\n")
			if sub.IsCorrect == 1 {
				crroct += sub.ItemName
				crroct += "、"
			}
		}

		outFile.WriteString("\n")
		outFile.WriteString("正确答案: ")
		outFile.WriteString(crroct)
		outFile.WriteString("\n")
		outFile.WriteString("\n")
		crroct = ""
		index++
	}
	fmt.Println("copy end")
}
