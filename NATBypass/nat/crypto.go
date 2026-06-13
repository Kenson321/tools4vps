// 使用AES对称加密
package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"

	//	"fmt"
	"io"
)

// 加密字符串，返回BASE64字符串
func AESCbcEncryptS(secretKey, src string) string {
	plaintext := []byte(src)
	return AESCbcEncryptB(secretKey, plaintext)
}

// 加密字节，返回BASE64字符串
func AESCbcEncryptB(secretKey string, plaintext []byte) string {
	key := []byte(secretKey)
	if len(key) > 16 {
		key = key[:16]
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	blockSize := block.BlockSize()

	plaintext = Padding(plaintext, blockSize)
	if len(plaintext)%aes.BlockSize != 0 {
		panic("plaintext is not a multiple of the block size")
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))

	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	//对称加密，不同的类，相同的方法
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

// 解密BASE64字符串，返回字节
func AESCbcDecryptB(secretKey, src string) []byte {
	key := []byte(secretKey)
	if len(key) > 16 {
		key = key[:16]
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	ciphertext, _ := base64.StdEncoding.DecodeString(src)

	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]

	ciphertext = ciphertext[aes.BlockSize:]
	if len(ciphertext)%aes.BlockSize != 0 {
		panic("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	//对称加密，不同的类，相同的方法
	mode.CryptBlocks(ciphertext, ciphertext)
	ciphertext = UnPadding(ciphertext)
	return ciphertext
}

// 解密BASE64字符串，返回字符串
func AESCbcDecryptS(secretKey, src string) string {
	ciphertext := AESCbcDecryptB(secretKey, src)
	return string(ciphertext)
}

// 在最后不足blockSize的每个填充字节中填充代表填充了多少位的数字
func Padding(plainText []byte, blockSize int) []byte {
	if blockSize > 256 {
		panic("blockSize超过一个字节代表的最大数字")
	}
	padding := blockSize - len(plainText)%blockSize //整除也要填充，因为UnPadding无法判断，必须读一个代表Padding的字节
	char := []byte{byte(padding)}
	newPlain := bytes.Repeat(char, padding)
	return append(plainText, newPlain...)
}

// 从最后一个字节中获取代表填充了多少位的数字，然后删除对应数量的填充字节
func UnPadding(plainText []byte) []byte {
	length := len(plainText)
	lastChar := plainText[length-1]
	padding := int(lastChar)
	return plainText[:length-padding]
}
