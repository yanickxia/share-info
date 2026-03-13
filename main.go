package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	magicHeader    = "ENVB1"
	saltSize       = 16
	pbkdf2Iters    = 600000
	keySize        = 32
	defaultInFile  = "env.snapshot.enc.b64"
	defaultOutFile = "env.snapshot.enc.b64"
	defaultDecOut  = "env.snapshot.json"
	defaultEncType = "base64"
	passwordEnvKey = "ENV_SNAPSHOT_PASSWORD"
)

type snapshot struct {
	CollectedAt string            `json:"collected_at"`
	Hostname    string            `json:"hostname"`
	GOOS        string            `json:"goos"`
	GOARCH      string            `json:"goarch"`
	Env         map[string]string `json:"env"`
}

func main() {
	mode := flag.String("mode", "encrypt", "运行模式: encrypt 或 decrypt")
	input := flag.String("in", defaultInFile, "输入文件路径（decrypt 模式使用）")
	output := flag.String("out", "", "输出文件路径（encrypt 默认 env.snapshot.enc.b64，decrypt 默认 env.snapshot.json）")
	encType := flag.String("encoding", defaultEncType, "encrypt 输出编码: base64 或 binary")
	password := flag.String("password", "", "加密密码（可留空，改用环境变量 ENV_SNAPSHOT_PASSWORD）")
	flag.Parse()

	m := strings.ToLower(strings.TrimSpace(*mode))
	if m != "encrypt" && m != "decrypt" {
		fatalf("不支持的 mode: %s（仅支持 encrypt/decrypt）", *mode)
	}
	enc := strings.ToLower(strings.TrimSpace(*encType))
	if m == "encrypt" && enc != "base64" && enc != "binary" {
		fatalf("不支持的 encoding: %s（仅支持 base64/binary）", *encType)
	}

	pass := readPassword(*password)
	out := resolveOutput(*output, m, enc)

	switch m {
	case "encrypt":
		if err := runEncrypt(out, pass, enc); err != nil {
			fatalf("%v", err)
		}
	case "decrypt":
		if err := runDecrypt(*input, out, pass); err != nil {
			fatalf("%v", err)
		}
	}
}

func runEncrypt(out, password, encoding string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("读取主机名失败: %w", err)
	}

	snap := snapshot{
		CollectedAt: time.Now().UTC().Format(time.RFC3339),
		Hostname:    hostname,
		GOOS:        runtime.GOOS,
		GOARCH:      runtime.GOARCH,
		Env:         collectEnv(),
	}

	plaintext, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("序列化环境变量失败: %w", err)
	}

	ciphertext, err := encrypt(plaintext, password)
	if err != nil {
		return fmt.Errorf("加密失败: %w", err)
	}

	output, err := encodeCiphertext(ciphertext, encoding)
	if err != nil {
		return fmt.Errorf("编码密文失败: %w", err)
	}

	if err := os.WriteFile(out, output, 0o600); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	fmt.Printf("已写入加密快照: %s (encoding=%s)\n", out, encoding)
	return nil
}

func runDecrypt(in, out, password string) error {
	inputData, err := os.ReadFile(in)
	if err != nil {
		return fmt.Errorf("读取输入文件失败: %w", err)
	}

	ciphertext, err := decodeCiphertext(inputData)
	if err != nil {
		return fmt.Errorf("解析输入密文失败: %w", err)
	}

	plaintext, err := decrypt(ciphertext, password)
	if err != nil {
		return fmt.Errorf("解密失败: %w", err)
	}

	if !json.Valid(plaintext) {
		return errors.New("解密结果不是有效 JSON")
	}

	if err := os.WriteFile(out, plaintext, 0o600); err != nil {
		return fmt.Errorf("写入解密结果失败: %w", err)
	}

	fmt.Printf("已写入解密结果: %s\n", out)
	return nil
}

func readPassword(passwordFlag string) string {
	pass := strings.TrimSpace(passwordFlag)
	if pass == "" {
		pass = strings.TrimSpace(os.Getenv(passwordEnvKey))
	}
	if pass == "" {
		fatalf("缺少密码：请通过 -password 或环境变量 %s 提供", passwordEnvKey)
	}
	return pass
}

func resolveOutput(flagValue, mode, enc string) string {
	if strings.TrimSpace(flagValue) != "" {
		return strings.TrimSpace(flagValue)
	}
	if mode == "decrypt" {
		return defaultDecOut
	}
	if enc == "binary" {
		return "env.snapshot.enc"
	}
	return defaultOutFile
}

func collectEnv() map[string]string {
	envMap := make(map[string]string)
	for _, kv := range os.Environ() {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		envMap[key] = value
	}
	return envMap
}

func encrypt(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("生成 salt 失败: %w", err)
	}

	key, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iters, keySize)
	if err != nil {
		return nil, fmt.Errorf("PBKDF2 派生密钥失败: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("生成 nonce 失败: %w", err)
	}

	body := gcm.Seal(nil, nonce, plaintext, []byte(magicHeader))

	buf := make([]byte, 0, len(magicHeader)+len(salt)+len(nonce)+len(body))
	buf = append(buf, []byte(magicHeader)...)
	buf = append(buf, salt...)
	buf = append(buf, nonce...)
	buf = append(buf, body...)
	return buf, nil
}

func encodeCiphertext(ciphertext []byte, encoding string) ([]byte, error) {
	switch encoding {
	case "binary":
		return ciphertext, nil
	case "base64":
		encoded := base64.StdEncoding.EncodeToString(ciphertext)
		return []byte(encoded + "\n"), nil
	default:
		return nil, fmt.Errorf("不支持的 encoding: %s", encoding)
	}
}

func decodeCiphertext(input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, errors.New("输入为空")
	}
	if bytes.HasPrefix(input, []byte(magicHeader)) {
		return bytes.Clone(input), nil
	}

	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return nil, errors.New("输入为空")
	}
	if bytes.HasPrefix(trimmed, []byte(magicHeader)) {
		return bytes.Clone(trimmed), nil
	}

	compact := compactWhitespace(trimmed)
	if len(compact) == 0 {
		return nil, errors.New("输入为空")
	}

	candidates := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range candidates {
		decoded, err := enc.DecodeString(string(compact))
		if err != nil {
			continue
		}
		if bytes.HasPrefix(decoded, []byte(magicHeader)) {
			return decoded, nil
		}
	}
	return nil, errors.New("输入既不是原始密文，也不是有效的 base64 密文")
}

func compactWhitespace(input []byte) []byte {
	buf := make([]byte, 0, len(input))
	for _, b := range input {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			buf = append(buf, b)
		}
	}
	return buf
}

func decrypt(ciphertext []byte, password string) ([]byte, error) {
	if len(ciphertext) < len(magicHeader)+saltSize {
		return nil, errors.New("密文长度不合法")
	}

	header := string(ciphertext[:len(magicHeader)])
	if header != magicHeader {
		return nil, errors.New("文件头不匹配，可能不是本工具生成的密文")
	}

	offset := len(magicHeader)
	salt := ciphertext[offset : offset+saltSize]
	offset += saltSize

	key, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iters, keySize)
	if err != nil {
		return nil, fmt.Errorf("PBKDF2 派生密钥失败: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}

	if len(ciphertext) < offset+gcm.NonceSize()+gcm.Overhead() {
		return nil, errors.New("密文数据不完整")
	}

	nonce := ciphertext[offset : offset+gcm.NonceSize()]
	body := ciphertext[offset+gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, body, []byte(magicHeader))
	if err != nil {
		return nil, errors.New("密码错误或密文已损坏")
	}
	return plaintext, nil
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "错误: "+format+"\n", args...)
	os.Exit(1)
}
