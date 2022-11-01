package dingtalk

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type SliceItem struct {
	Url  string
	Name string
}

const sourceDir string = "data"

var prefix string
var wg sync.WaitGroup
var taskChan chan SliceItem
var _endSigal chan int

func init() {
	prefix = "https://dtliving-bj.dingtalk.com/live_hp/"
	taskChan = make(chan SliceItem, 10)
	_endSigal = make(chan int)
}

func downloadSliceItem(url string, name string) {
	defer wg.Done()
	fileTitle := strings.Split(strings.Split(name, "/")[1], "?")[0]
	folder := ""
	fmt.Println("start download folder:", fileTitle)
	err := os.MkdirAll(sourceDir+"/"+folder, 0777)
	if err != nil {
		log.Fatalln(err)
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	content := string(bytes)
	file, err := os.Create(sourceDir + "/" + folder + "/" + fileTitle)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	length, err := writer.WriteString(content)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("%s, %d writed\n", fileTitle, length)
}

func fixSliceUrl(path string) {
	path = strings.ReplaceAll(path, "\n", "")
	slicePath := prefix + path
	fmt.Printf("slice url:%v\n", slicePath)
	wg.Add(1)
	task := SliceItem{
		Url:  slicePath,
		Name: path,
	}
	taskChan <- task
	//downloadSliceItem(slicePath, path)
}

func readFileContent(filePath string) {
	if len(filePath) <= 0 {
		log.Fatalln(errors.New("file path error"))
	}
	fmt.Printf("start read file content with path : %s\n", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalln(err)
	}
	_reader := bufio.NewReader(file)

	for {
		line, err := _reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				panic(err)
			}
		}

		if strings.Index(line, "#EXT") != 0 {
			fixSliceUrl(line)
		}
	}
	_endSigal <- 1
}

func mergeTs() {
	fmt.Println("start merge ts files")
	_dir, err := os.Open(sourceDir)
	if err != nil {
		log.Fatalln(err)
	}
	defer _dir.Close()
	__dir, err := _dir.Readdir(0)
	if err != nil {
		log.Fatalln(err)
	}
	segements := make([]int, 0)
	_segements := make([]string, 0)
	for _, item := range __dir {
		index, _ := strconv.Atoi(strings.Split(item.Name(), ".")[0])
		segements = append(segements, index)
	}
	sort.Ints(segements)
	for _, value := range segements {
		_value := strconv.Itoa(value)
		_segements = append(_segements, sourceDir+"/"+_value+".ts")
	}
	_result := strings.Join(_segements, "|")
	toMp4(_result, "out")
}

func toMp4(src string, out string) {
	binary, err := exec.LookPath("FFMPEG")
	if err != nil {
		log.Fatalln(err)
	}

	args := []string{
		"-i",
		fmt.Sprintf("concat:%s", src),
		"-acodec",
		"copy",
		"-vcodec",
		"copy",
		"-absf",
		"aac_adtstoasc",
		fmt.Sprintf("%s.mp4", out),
	}

	cmd := exec.Command(binary, args...)
	_, err = cmd.Output()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("result video:%v.mp4", out)
}

func downloadReal() {
	for {
		select {
		case task := <-taskChan:
			downloadSliceItem(task.Url, task.Name)
		case _end := <-_endSigal:
			fmt.Printf("all task put in:%v\n", _end)
			break
			// case <-time.After(time.Second * 10):
			// 	break
		}
	}
}

func DownloadM3u8File() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("start download m3u8 file")
	os.RemoveAll(sourceDir)
	os.Mkdir(sourceDir, 0777)
	go downloadReal()
	readFileContent("source.m3u8")

	wg.Wait()

	mergeTs()

}
