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
	"time"
)

type SliceItem struct {
	Url  string
	Name string
}

const sourceDir string = "data"

var prefix string
var wg sync.WaitGroup
var taskChan chan SliceItem
var PwdKey []byte
func init() {
	PwdKey = []byte("ABCDABCDABCDABCD")
	//https://streamddo.yxt.com/orgs/dingf731889e7115715535c2f4657eb6378f/knowledge/video/202211/1bc9416915914d8c96e08d5502c3222c_1442042431_480p_enc.m3u8.0.ts
	//https://drm.media.baidubce.com/v1/tokenVideoKey?videoKeyId=job-nkukpfwy39ys5fay&playerId=pid-1-5-1&token=97cfb1a7d0265c9b869a38b53eb67e2f92954f81613d3b87ed4f388131efba07_7d2195c92f8842a586f4299a8244b1fa_1672056495
	prefix = "https://streamddo.yxt.com/orgs/dingf731889e7115715535c2f4657eb6378f/knowledge/video/202211/" // "https://dtliving-sh.dingtalk.com/live_hp/" //"https://dtliving-bj.dingtalk.com/live_hp/"
	fmt.Printf("cpu count: %v \n", runtime.NumCPU())
	taskChan = make(chan SliceItem, runtime.NumCPU())
}

func downloadSliceItem(url string, name string) {
	fmt.Printf("url:%v \n name: %v \n", url, name)
	defer wg.Done()
	var fileTitle string

	if strings.Contains(name, "?") {
		fileTitle = strings.Split(strings.Split(name, "/")[1], "?")[0]
	} else {
		fileTitle = name
	}

	folder := ""
	fmt.Println("start download file:", fileTitle)
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
	path := sourceDir + "/" + folder + "/" + fileTitle
	// file, err := os.Create(sourceDir + "/" + folder + "/" + fileTitle)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// defer file.Close()
	err = ioutil.WriteFile(path, bytes, 0777)

	// content := string(bytes)

	// writer := bufio.NewWriter(file)

	// length, err := writer.WriteString(content)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("%s, %d writed\n", fileTitle, len(bytes))
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
	fmt.Printf("result video:%v.mp4\n", out)
}

func downloadReal() {
	for {
		select {
		case task := <-taskChan:
			downloadSliceItem(task.Url, task.Name)
		case <-time.After(time.Second * 5):
			fmt.Println("all task put in")
			goto END
		}
	}
END:
	fmt.Println("stop task loop")
}

func DownloadM3u8File() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("start download m3u8 file")
	os.RemoveAll(sourceDir)
	os.Mkdir(sourceDir, 0777)
	os.Remove("out.mp4")
	go downloadReal()
	readFileContent("source.m3u8")

	wg.Wait()

	mergeTs()

}
