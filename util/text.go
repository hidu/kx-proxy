package util

import (
	"bytes"
	"io/ioutil"
	"log"
)

// LoadTxtConf 读取一个文本配置文件
// 会过滤掉空行 以及 #开头的
func LoadTxtConf(confPath string) []string {
	var lines []string
	datas, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.Fatalln(err)
	}
	ls := bytes.Split(datas, []byte("\n"))
	for _, lineBs := range ls {
		index := bytes.IndexByte(lineBs, '#')
		if index > -1 {
			lineBs = lineBs[:index]
		}
		lineBs = bytes.TrimSpace(lineBs)
		if len(lineBs) == 0 {
			continue
		}
		lines = append(lines, string(lineBs))
	}
	return lines
}
