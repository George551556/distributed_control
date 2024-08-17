package utils

import (
	"crypto/md5"
	"encoding/hex"
	"math/rand"
)

var charsets = "abcdefghijklmnopqrstuvwxyz0123456789"

// 将传入的字符串生成为一个md5码
func Str2md5(str string) string {
	str_1 := []byte(str)
	md5New := md5.New()
	md5New.Write(str_1)
	md5string := hex.EncodeToString(md5New.Sum(nil))
	return md5string
}

// 以随机数字生成一个md5值
func GetRandom_md5() string {
	temp := make([]byte, 15)
	for i := range temp {
		temp[i] = charsets[rand.Int()%len(charsets)]
	}
	temp_str := string(temp)
	return Str2md5(temp_str)
}
