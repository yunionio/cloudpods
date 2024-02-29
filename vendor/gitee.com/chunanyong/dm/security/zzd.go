/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package security

import (
	"math/big"
)

const (
	DH_KEY_LENGTH int = 64
	/* 低7位用于保存分组加密算法中的工作模式 */
	WORK_MODE_MASK int = 0x007f
	ECB_MODE       int = 0x1
	CBC_MODE       int = 0x2
	CFB_MODE       int = 0x4
	OFB_MODE       int = 0x8
	/* 高位保存加密算法 */
	ALGO_MASK int = 0xff80
	DES       int = 0x0080
	DES3      int = 0x0100
	AES128    int = 0x0200
	AES192    int = 0x0400
	AES256    int = 0x0800
	RC4       int = 0x1000
	MD5       int = 0x1100

	// 用户名密码加密算法
	DES_CFB int = 132
	// 消息加密摘要长度
	MD5_DIGEST_SIZE int = 16

	MIN_EXTERNAL_CIPHER_ID int = 5000
)

var dhParaP = "C009D877BAF5FAF416B7F778E6115DCB90D65217DCC2F08A9DFCB5A192C593EBAB02929266B8DBFC2021039FDBD4B7FDE2B996E00008F57AE6EFB4ED3F17B6D3"
var dhParaG = "5"
var defaultIV = []byte{0x20, 0x21, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a,
	0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a,
	0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x20}
var p *big.Int
var g *big.Int

func NewClientKeyPair() (key *DhKey, err error) {
	p, _ = new(big.Int).SetString(dhParaP, 16)
	g, _ = new(big.Int).SetString(dhParaG, 16)
	dhGroup := newDhGroup(p, g)
	key, err = dhGroup.GeneratePrivateKey(nil)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func ComputeSessionKey(clientPrivKey *DhKey, serverPubKey []byte) []byte {
	serverKeyX := bytes2Bn(serverPubKey)
	clientPrivKeyX := clientPrivKey.GetX()
	sessionKeyBN := serverKeyX.Exp(serverKeyX, clientPrivKeyX, p)
	return Bn2Bytes(sessionKeyBN, 0)
}

func bytes2Bn(bnBytesSrc []byte) *big.Int {
	if bnBytesSrc == nil {
		return nil
	}
	if bnBytesSrc[0] == 0 {
		return new(big.Int).SetBytes(bnBytesSrc)
	}
	validBytesCount := len(bnBytesSrc) + 1
	bnBytesTo := make([]byte, validBytesCount)
	bnBytesTo[0] = 0
	copy(bnBytesTo[1:validBytesCount], bnBytesSrc)
	return new(big.Int).SetBytes(bnBytesTo)
}

func Bn2Bytes(bn *big.Int, bnLen int) []byte {
	var bnBytesSrc, bnBytesTemp, bnBytesTo []byte
	var leading_zero_count int
	validBytesCount := 0
	if bn == nil {
		return nil
	}
	bnBytesSrc = bn.Bytes()

	// 去除首位0
	if bnBytesSrc[0] != 0 {
		bnBytesTemp = bnBytesSrc
		validBytesCount = len(bnBytesTemp)
	} else {
		validBytesCount = len(bnBytesSrc) - 1
		bnBytesTemp = make([]byte, validBytesCount)
		copy(bnBytesTemp, bnBytesSrc[1:validBytesCount+1])
	}

	if bnLen == 0 {
		leading_zero_count = 0
	} else {
		leading_zero_count = bnLen - validBytesCount
	}
	// 如果位数不足DH_KEY_LENGTH则在前面补0
	if leading_zero_count > 0 {
		bnBytesTo = make([]byte, DH_KEY_LENGTH)
		i := 0
		for i = 0; i < leading_zero_count; i++ {
			bnBytesTo[i] = 0
		}
		copy(bnBytesTo[i:i+validBytesCount], bnBytesTemp)
	} else {
		bnBytesTo = bnBytesTemp
	}
	return bnBytesTo
}
