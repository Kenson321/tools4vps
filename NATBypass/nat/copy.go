// 负责加密后写入和读取后解密
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
)

const KEY = "password"          //TODO
const BUFFER_SIZE = 1024 * 1024 //必须小于2^32-1，1mb
const BUFFER_SIZE_BYTE = 4      //int32占4字节

// 加密plaintext并写入dst
func writeEncrypt(dst io.Writer, plaintext []byte) {
	// 加密的字节
	es := AESCbcEncryptB(KEY, plaintext)
	b := []byte(es)
	// 加密的字节的长度
	l := int32(len(b))
	lb := bytes.NewBuffer([]byte{})
	err := binary.Write(lb, binary.BigEndian, &l)
	if err != nil {
		log.Println(err)
	}
	// 先写加密的字节的长度，再写加密的字节
	_, err = dst.Write(lb.Bytes())
	if err != nil {
		log.Println(err)
	}
	_, err = dst.Write(b)
	if err != nil {
		log.Println(err)
	}
}

// 从src中一段一段读取，然后按段加密写入dst
// 异步和单字节读取都是为了提高性能
func CopyToEncrypt(dst io.Writer, src io.Reader) (written int64, err error) {
	// 异步从src读取并写入缓存
	readCount := 0
	readTotal := -1
	readCh := make(chan byte, BUFFER_SIZE)
	go func() {
		buf1 := make([]byte, 1)
		for {
			nr, er := src.Read(buf1)
			if nr > 0 {
				if nr != 1 {
					log.Panic("读取非按单个字节读")
				}
				readCh <- buf1[0]
				readCount++
			}
			if er != nil {
				if er != io.EOF {
					err = er
					log.Println(er)
				}
				readTotal = readCount
				break
			}
		}
	}()

	// 从缓存读取一段然后加密写入dst
	writeBuf := [BUFFER_SIZE]byte{}
	writeCnt := 0
	writeTotal := 0
	for readTotal == -1 || writeTotal < readTotal {
		byte1 := <-readCh
		writeBuf[writeCnt] = byte1
		writeCnt++
		writeTotal++
		if writeCnt == BUFFER_SIZE || writeTotal == readCount { //如果到达缓存上限，或者读阻塞导致写速度追上了读速度
			writeEncrypt(dst, writeBuf[0:writeCnt])
			writeCnt = 0
		}
	}

	return int64(writeTotal), err
}

// 从src中读取，并按加密的逻辑进行反向解密
func readDecrypt(src io.Reader) (text []byte, readed int, err error) {
	//读取代表长度的前4个字节
	bufL := make([]byte, BUFFER_SIZE_BYTE)
	//	nr, er := src.Read(bufL)
	nr, er := io.ReadFull(src, bufL)
	if er != nil {
		err = er
		return
	}
	if nr != BUFFER_SIZE_BYTE {
		log.Panic("首字节非数字")
	}
	bb := bytes.NewBuffer(bufL)
	var size int32
	er = binary.Read(bb, binary.BigEndian, &size)
	if er != nil {
		log.Panic(er, "首字节非数字")
	}

	//按长度读取完整的加密字符串进行解密
	bufB := make([]byte, size)
	nr, er = io.ReadFull(src, bufB)
	//	nr, er = src.Read(bufB)
	if er != nil {
		if er == io.EOF {
			log.Panic("无指定长度的字节EOF")
		}
		err = er
		return
	}
	if int32(nr) != size {
		log.Panic(fmt.Sprintf("无指定长度的字节 %d %d", nr, size))
	}

	text = AESCbcDecryptB(KEY, string(bufB))
	readed = len(text)
	return
}

// 从src中读取加密内容并解密写入dst中
func CopyFromDecrypt(dst io.Writer, src io.Reader) (written int64, err error) {
	for {
		buf, nr, er := readDecrypt(src)
		if nr > 0 {
			nw, ew := dst.Write(buf)
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
