package links

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/xanygo/anygo/xcodec/xbase"
)

const (
	aesTable = "kxproxyb8PsyCQ4b"
)

var aesBlock cipher.Block

func init() {
	var err error
	aesBlock, err = aes.NewCipher([]byte(aesTable))
	if err != nil {
		panic(err)
	}
}

// CipherStreamWrite 对数据流进行加密
func CipherStreamWrite(skey string, encodeURL string, writer io.Writer) *cipher.StreamWriter {
	key := strMd5(fmt.Sprintf("%s#kxsw#%s", skey, encodeURL))
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	var iv [aes.BlockSize]byte
	stream := cipher.NewCTR(block, iv[:])
	return &cipher.StreamWriter{S: stream, W: writer}
}

// CipherStreamReader 对数据流进行加密
func CipherStreamReader(skey string, encodeURL string, reader io.Reader) *cipher.StreamReader {
	key := strMd5(fmt.Sprintf("%s#kxsw#%s", skey, encodeURL))
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	var iv [aes.BlockSize]byte
	stream := cipher.NewCTR(block, iv[:])
	return &cipher.StreamReader{S: stream, R: reader}
}

func strMd5(mystr string) []byte {
	h := md5.New()
	h.Write([]byte(mystr))
	return h.Sum(nil)
}

// EncryptURL 对url进行加密
func EncryptURL(srcURL string) (string, error) {
	src := []byte(srcURL)
	padLen := aes.BlockSize - (len(src) % aes.BlockSize)

	for i := 0; i < padLen; i++ {
		src = append(src, byte(padLen))
	}

	srcLen := len(src)

	encryptText := make([]byte, srcLen+aes.BlockSize)

	iv := encryptText[srcLen:]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	mode := cipher.NewCBCEncrypter(aesBlock, iv)

	mode.CryptBlocks(encryptText[:srcLen], src)
	s := xbase.Base58.EncodeToString(encryptText)
	return s, nil
}

// DecryptURL 对url进行解密
func DecryptURL(srcURL string) (string, error) {
	if srcURL == "" {
		return "", errors.New("empty url")
	}
	src, err := xbase.Base58.DecodeString(srcURL)
	if err != nil {
		log.Println("base64_decode_failed:", err.Error(), "data:", srcURL[1:])
		return "", err
	}
	if len(src) < aes.BlockSize*2 || len(src)%aes.BlockSize != 0 {
		return "", errors.New("wrong data size")
	}

	srcLen := len(src) - aes.BlockSize

	decryptText := make([]byte, srcLen)

	iv := src[srcLen:]

	mode := cipher.NewCBCDecrypter(aesBlock, iv)

	mode.CryptBlocks(decryptText, src[:srcLen])

	paddingLen := int(decryptText[srcLen-1])

	if paddingLen > 16 {
		return "", errors.New("wrong pading size")
	}

	return string(decryptText[:srcLen-paddingLen]), nil
}
