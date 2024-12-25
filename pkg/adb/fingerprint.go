package adb

import (
	"encoding/hex"
	"sync"
)

// KeyInfo 存储指纹和注释信息的结构体
type KeyInfo struct {
	Fingerprint string
	Comment     string
}

// globalKeyMap 全局映射，用于存储指纹和注释信息
var globalKeyMap = make(map[string]KeyInfo)
var keyMapMutex sync.Mutex

// AddKeyInfo 添加指纹和注释信息到全局映射
func AddKeyInfo(md5hash []byte, comment string) {
	fingerprint := hex.EncodeToString(md5hash)
	keyMapMutex.Lock()
	defer keyMapMutex.Unlock()
	globalKeyMap[fingerprint] = KeyInfo{
		Fingerprint: fingerprint,
		Comment:     comment,
	}
}

// GetKeyInfo 从全局映射中检索注释信息
func GetKeyInfo(fingerprint string) (KeyInfo, bool) {
	keyMapMutex.Lock()
	defer keyMapMutex.Unlock()
	keyInfo, exists := globalKeyMap[fingerprint]
	return keyInfo, exists
}
