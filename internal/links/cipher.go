package links

import (
	"crypto/md5"
	"errors"

	"github.com/xanygo/anygo/ds/xsync"
	"github.com/xanygo/anygo/xattr"
	"github.com/xanygo/anygo/xcodec"
	"github.com/xanygo/anygo/xcodec/xbase"
)

func getSecretKey() string {
	return xattr.GetDefault[string]("SecretKey", "5a5253212cd65fb2926b330d84cd487b")
}

var myCipher = &xsync.OnceInit[xcodec.Cipher]{
	New: func() xcodec.Cipher {
		return &xcodec.AesOFB{
			Key: getSecretKey(),
		}
	},
}

func strMd5(mystr string) []byte {
	h := md5.New()
	h.Write([]byte(mystr))
	return h.Sum(nil)
}

// EncryptURL 对url进行加密
func EncryptURL(srcURL string) (string, error) {
	encryptText, err := myCipher.Load().Encrypt([]byte(srcURL))
	if err != nil {
		return "", err
	}
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
		return "", err
	}
	paninText, err := myCipher.Load().Decrypt(src)

	if err != nil {
		return "", err
	}
	return string(paninText), nil
}
