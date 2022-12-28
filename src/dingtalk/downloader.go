package dingtalk

import (
	"bufio"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Info struct {
	uri      *string
	token    *string
	playerId *string
	fileName *string
}

type EncodeKey struct {
	uri    *string
	key    *string
	iv     *string
	format *string
}

type EncodeData struct {
	VideoKeyId        string
	PlayerId          string
	EncryptedVideoKey string
}

var (
	info      *Info
	encodeKey *EncodeKey
	_folder   string = "tmp"
	_wg       sync.WaitGroup
)

func init() {
	os.RemoveAll(_folder)
	os.MkdirAll(_folder, 0777)
}

func Download() {
	info = &Info{}
	info.uri = flag.String("uri", "", "m3u8 uri required")
	info.token = flag.String("token", "", "user token required")
	info.playerId = flag.String("playerId", "", "player id required")
	info.fileName = flag.String("out", "movie", "result file name")
	flag.Parse()

	// res, err := buildRequest("https://api-component.yxt.com/v1/config/migrate?fileId=71bffb65-bae2-476b-88ff-e053800a238e&fileUrl=orgs%252Fdingf731889e7115715535c2f4657eb6378f%252Fknowledge%252Fvideo%252F202211%252F1bc9416915914d8c96e08d5502c3222c.mp4&originType=&enc=false", "GET")
	// if err != nil {
	// 	fmt.Println("err:", err.Error())
	// }
	// fmt.Println(string(res))
	getM3u8()
	_wg.Wait()
}

func getM3u8() {
	_wg.Add(1)
	defer _wg.Done()
	result, err := buildRequest(*info.uri, "GET")
	if err != nil {
		log.Fatalln("过去m3u8文件失败:", err.Error())
	}
	m3u8File, err := os.OpenFile("tmp.m3u8", os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		log.Fatalln("创建m3u8文件失败:", err.Error())
	}
	defer m3u8File.Close()
	m3u8File.Write(result)
	encode_key := getEncodeKey(string(result))
	if len(encode_key) > 0 {
		encodeKeyRequestUri := *encodeKey.uri + "&playerId=" + *info.playerId + "&token=" + *info.token
		keyData, err := buildRequest(encodeKeyRequestUri, "GET")
		if err != nil {
			log.Fatalln("获取加密key失败:", err.Error())
		}
		var encodeData EncodeData
		err = json.Unmarshal(keyData, &encodeData)
		if err != nil {
			log.Fatalln("解析加密key失败:", err.Error())
		}
		encodeKey.key = &encodeData.EncryptedVideoKey
	}
	tsLists := parseTs(string(result))
	tsFiles := make([]string, 0)
	for _, ts := range tsLists {
		path := downloadTsFile(ts)
		tsFiles = append(tsFiles, path)
	}
	_mergeTs(tsFiles)
}

//默认加密方式 AES-128
func getEncodeKey(body string) (key string) {
	lines := strings.Split(body, "\n")
	key = ""
	for _, line := range lines {
		if strings.Contains(line, "#EXT-X-KEY") {
			encodeKey = &EncodeKey{}
			items := strings.Split(line, ",")
			for _, item := range items {
				if strings.Index(item, "URI") == 0 {
					encodeKey.uri = &strings.Split(item, "\"")[1]
				} else if strings.Index(item, "IV") == 0 {
					encodeKey.iv = &strings.Split(item, "=")[1]
				} else if strings.Index(item, "KEYFORMAT") == 0 {
					encodeKey.format = &strings.Split(item, "=")[1]
				}
			}
			key = *encodeKey.uri
			goto END_KEY
		}
	}
END_KEY:
	return
}

func parseTs(body string) (result []string) {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if len(line) <= 0 {
			continue
		}
		result = append(result, line)
	}
	return
}

func downloadTsFile(ts string) string {
	_wg.Add(1)
	defer _wg.Done()
	url := "https://streamddo.yxt.com/orgs/dingf731889e7115715535c2f4657eb6378f/knowledge/video/202211/" + ts
	fmt.Println("url:", url)
	result, err := buildRequest(url, "GET")
	if err != nil {
		log.Fatalln("下载ts文件失败:", err.Error(), ts)
	}
	_path := _folder + "/" + ts
	tsFile, err := os.OpenFile(_path, os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		log.Fatalln("创建ts文件失败:", err.Error(), ts)
	}
	defer tsFile.Close()
	// fmt.Println("en:", string(result))
	// fmt.Println("key:", *encodeKey.key)
	// fmt.Println("iv:", *encodeKey.iv)
	_result, err := AES128Decrypt(result, []byte(*encodeKey.key), []byte(*encodeKey.iv))
	if err != nil {
		log.Fatalln("解密ts内容失败:", err.Error(), ts)
	}
	// fmt.Println("de:", string(_result))
	_, err = tsFile.WriteString(string(_result))
	if err != nil {
		log.Fatalln("写入ts文件失败:", err.Error(), ts)
	}
	return _path
}

func _mergeTs(files []string) {
	_wg.Add(1)
	defer _wg.Done()
	binary, err := exec.LookPath("FFMPEG")
	if err != nil {
		log.Fatalln(err)
	}
	src := strings.Join(files, "|")

	//ffmpeg -i "concat:1.ts|2.ts" -acodec copy out.mp3
	//ffmpeg -i "concat:1.ts|2.ts" -acodec copy -vcodec copy -absf aac_adtstoasc output.mp4
	//ffmpeg -i input.txt -acodec copy -vcodec copy -absf aac_adtstoasc output.mp4

	//  input.txt:
	// ffconcat version 1.0
	// file  0.ts
	// file  1.ts
	args := []string{
		"-i",
		fmt.Sprintf("concat:%s", src),
		"-acodec",
		"copy",
		"-vcodec",
		"copy",
		"-absf",
		"aac_adtstoasc",
		fmt.Sprintf("%s.mp4", *info.fileName),
	}

	cmd := exec.Command(binary, args...)
	fmt.Println("join cmd:", cmd)
	_, err = cmd.Output()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("result video:%v.mp4\n", *info.fileName)
}

func buildRequest(url string, method string) ([]byte, error) {
	client := &http.Client{}
	if len(method) > 0 {
		method = "GET"
	}
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 12_6_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36 DingTalk(6.5.47-macOS-macOS-MAS-26763337) nw Channel/201200")
	request.Header.Set("Connection", "keep-alive")
	request.Header.Set("Accept", "*/*")
	request.Header.Set("Accept-Encoding", "gzip")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	request.Header.Set("source", "4401")
	request.Header.Set("token", "AAAAAFH_NmoP35uOEuZDJsZm1L00yl8Nps2lwi_xt70kMTp6go1XVOeJTlzRVCh0ErjJzHW9-KEinxj80ZiYtl2iWkFRf1gFmpSC_inQE7tbrjs7BEFhUeqf5jbAIpC-dboThPuIZfa2LV4GdaThQnoJqZs")
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	_encoding := resp.Header.Get("Content-Encoding")

	if _encoding == "gzip" {
		bodyReader, _ := gzip.NewReader(resp.Body)
		bytes, err := ioutil.ReadAll(bodyReader)
		if err != nil {
			return nil, err
		}
		return bytes, nil
	} else {
		if strings.Index(url, "...ts") > 0 {
			bodyReader, _ := gzip.NewReader(resp.Body)
			fmt.Println("encode:", bodyReader)
			reader := bufio.NewReader(resp.Body)
			data := make([]byte, 512)
			reader.Read(data)
			fmt.Println("header:", resp.Header)
			fmt.Println("返回值:", string(data))
			return nil, nil
		} else {
			bytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			return bytes, nil
		}
	}
}

func AES128Decrypt(crypted, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	if len(iv) == 0 {
		iv = key
	}
	blockMode := cipher.NewCBCDecrypter(block, iv[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = pkcs5UnPadding(origData)
	return origData, nil
}

func pkcs5UnPadding(origData []byte) []byte {
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}
