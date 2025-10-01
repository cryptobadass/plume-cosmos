package main

import (
	"fmt"
	"log"

	"github.com/cosmos/cosmos-sdk/crypto/keys/pqc"
)

func main() {
	fmt.Println("🧪 测试Plume-Cosmos PQC功能")
	fmt.Println("=" + string(make([]byte, 40)) + "=")

	// 测试PQC密钥生成
	fmt.Println("\n🔑 测试PQC密钥生成:")
	privKey := pqc.GenPrivKey()
	pubKey := privKey.PubKey()

	fmt.Printf("   私钥长度: %d 字节\n", len(privKey.Bytes()))
	fmt.Printf("   公钥长度: %d 字节\n", len(pubKey.Bytes()))
	fmt.Printf("   地址: %x\n", pubKey.Address())

	// 测试签名验证
	fmt.Println("\n✍️  测试签名验证:")
	message := []byte("Hello, Plume-Cosmos PQC!")

	signature, err := privKey.Sign(message)
	if err != nil {
		log.Printf("签名失败: %v", err)
		return
	}

	fmt.Printf("   签名长度: %d 字节\n", len(signature))

	valid := pubKey.VerifySignature(message, signature)
	fmt.Printf("   验证结果: %v\n", valid)

	// 测试PQC地址
	if pqcPubKey, ok := pubKey.(*pqc.PubKey); ok {
		fmt.Printf("   PQC地址: %s\n", pqcPubKey.PqcAddress)
	}

	fmt.Println("\n✅ PQC功能测试完成!")
}

