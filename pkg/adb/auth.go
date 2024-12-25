package adb

import (
	"bytes"
	"crypto/md5"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

// Auth 实现ADB认证
type Auth struct{}

func NewAuth() *Auth {
	return &Auth{}
}

// RSAPublicKey ADB RSA公钥结构
type RSAPublicKey struct {
	Len      uint32   // n[]的长度（uint32_t数量）
	N0inv    uint32   // -1 / n[0] mod 2^32
	N        []uint32 // 模数（小端序数组）
	RR       []uint32 // R^2（小端序数组）
	Exponent uint32   // 指数（3或65537）
	Comment  string   // 公钥注释
}

// ParsePublicKey 解析公钥数据
func ParsePublicKey(data []byte) (*rsa.PublicKey, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("invalid public key: empty data")
	}

	// 解析base64数据和注释
	parts := bytes.Split(data, []byte{0})
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid public key format")
	}

	// 解码base64数据
	keyData, err := base64.StdEncoding.DecodeString(string(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}

	// 获取注释（如果存在）
	comment := ""
	if len(parts) > 1 {
		comment = strings.TrimSpace(string(parts[1]))
	}

	return parsePublicKeyFromStruct(keyData, comment)
}

// parsePublicKeyFromStruct 从二进制结构解析公钥
func parsePublicKeyFromStruct(data []byte, comment string) (*rsa.PublicKey, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("invalid public key")
	}

	// 读取长度
	offset := 0
	lens := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	// 验证数据长度
	expectedLen := 4 + 4 + (lens * 4) + (lens * 4) + 4
	if len(data) != int(expectedLen) {
		return nil, fmt.Errorf("invalid public key length")

	}

	// 跳过n0inv
	offset += 4

	// 读取模数n
	nBytes := make([]byte, lens*4)
	copy(nBytes, data[offset:offset+int(lens*4)])
	reverseBytes(nBytes) // 转换字节序
	offset += int(lens * 4)

	// 跳过RR
	offset += int(lens * 4)

	// 读取指数
	e := binary.LittleEndian.Uint32(data[offset:])
	if e != 3 && e != 65537 {
		return nil, fmt.Errorf("invalid exponent %d, only 3 and 65537 are supported", e)
	}

	// 创建RSA公钥
	n := new(big.Int).SetBytes(nBytes)
	key := &rsa.PublicKey{
		N: n,
		E: int(e),
	}

	// 计算指纹
	md5hash := md5.New()
	md5hash.Write(data)
	fingerprint := hex.EncodeToString(md5hash.Sum(nil))

	// 将指纹和注释添加到key的扩展信息中
	// 注意：在Go中，我们需要另外的方式来存储这些信息
	// 这里可以使用全局映射或创建包装类型
	AddKeyInfo([]byte(comment), fingerprint)
	return key, nil
}

// reverseBytes 反转字节序列
func reverseBytes(b []byte) {
	for i := 0; i < len(b)/2; i++ {
		j := len(b) - i - 1
		b[i], b[j] = b[j], b[i]
	}
}

// ExtendedRSAPublicKey 包装RSA公钥，包含额外信息
type ExtendedRSAPublicKey struct {
	*rsa.PublicKey
	Fingerprint string
	Comment     string
}

// NewExtendedRSAPublicKey 创建扩展的RSA公钥
func NewExtendedRSAPublicKey(key *rsa.PublicKey, fingerprint, comment string) *ExtendedRSAPublicKey {
	return &ExtendedRSAPublicKey{
		PublicKey:   key,
		Fingerprint: fingerprint,
		Comment:     comment,
	}
}

// VerifySignature 验证签名
func (k *ExtendedRSAPublicKey) VerifySignature(data, signature []byte) bool {
	// 实现签名验证逻辑
	// 这里需要根据ADB协议的具体要求实现
	// 通常涉及RSA验证和特定的填充方案
	return false
}

// GetFingerprint 获取公钥指纹
func (k *ExtendedRSAPublicKey) GetFingerprint() string {
	return k.Fingerprint
}

// GetComment 获取公钥注释
func (k *ExtendedRSAPublicKey) GetComment() string {
	return k.Comment
}

// ParsePrivateKey 解析私钥数据
func ParsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	// 实现私钥解析逻辑
	// 这通常涉及PEM解码和PKCS#1/PKCS#8格式解析
	return nil, fmt.Errorf("not implemented")
}

// GenerateKey 生成新的密钥对
func GenerateKey(bits int) (*rsa.PrivateKey, error) {
	// 实现密钥生成逻辑
	return nil, fmt.Errorf("not implemented")
}
