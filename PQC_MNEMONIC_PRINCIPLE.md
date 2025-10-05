# PQC 账户助记词生成原理详解

## 1. 助记词概述

在使用 Falcon-512 生成 PQC 账户时，系统会生成一个助记词（Mnemonic）。助记词是基于 BIP-39 标准的人类可读私钥表示形式，用于安全地备份和恢复密钥。

## 2. BIP-39 标准

### 2.1 BIP-39 规范
- **标准**: Bitcoin Improvement Proposal 39
- **目的**: 将随机生成的私钥转换为人类可读的单词序列
- **安全性**: 通过熵值确保足够的随机性
- **语言**: 支持多种语言，Cosmos SDK 默认使用英语

### 2.2 助记词结构
```
[随机熵] → [助记词] → [种子] → [主密钥] → [派生密钥]
```

## 3. 助记词生成流程

### 3.1 熵值生成
```go
// 位置: crypto/keyring/types.go:38
const defaultEntropySize = 256  // 256位熵值

// 位置: crypto/keyring/keyring.go:527
entropy, err := bip39.NewEntropy(defaultEntropySize)
```

**技术细节：**
- **熵值大小**: 256位（32字节）
- **随机性**: 来自系统加密安全的随机数生成器
- **安全性**: 提供 2^256 种可能的组合

### 3.2 助记词转换
```go
// 位置: crypto/keyring/keyring.go:532
mnemonic, err := bip39.NewMnemonic(entropy)
```

**转换过程：**
1. 256位熵值 → 24个单词
2. 每个单词代表11位（2^11 = 2048个单词）
3. 24 × 11 = 264位（包含8位校验和）

### 3.3 助记词到种子
```go
// 位置: crypto/hd/algo.go:128
seed, err := bip39.NewSeedWithErrorChecking(mnemonic, bip39Passphrase)
```

**种子生成：**
- **输入**: 助记词 + 可选密码短语
- **算法**: PBKDF2 (Password-Based Key Derivation Function 2)
- **参数**: 
  - 哈希函数: HMAC-SHA512
  - 迭代次数: 2048
  - 输出长度: 512位（64字节）

## 4. 密钥派生流程

### 4.1 主密钥生成
```go
// 位置: crypto/hd/algo.go:133
masterPriv, ch := ComputeMastersFromSeed(seed)
```

**主密钥计算：**
- **输入**: 512位种子
- **输出**: 主私钥 + 链码
- **算法**: HMAC-SHA512
- **用途**: 作为所有派生密钥的根

### 4.2 HD 路径派生
```go
// 位置: crypto/hd/algo.go:137
derivedKey, err := DerivePrivateKeyForPath(masterPriv, ch, hdPath)
```

**路径示例：**
- **标准路径**: `m/44'/118'/0'/0/0`
- **含义**: 
  - `m`: 主密钥
  - `44'`: BIP-44 标准
  - `118'`: Cosmos 币种代码
  - `0'`: 账户索引
  - `0`: 外部/内部链
  - `0`: 地址索引

### 4.3 PQC 密钥生成
```go
// 位置: crypto/hd/algo.go:146
return pqc.GenPrivKeyFromSecret(bz)
```

**PQC 密钥创建：**
- **输入**: 派生的私钥字节
- **算法**: Falcon-512
- **输出**: PQC 密钥对（私钥 + 公钥）

## 5. 代码实现分析

### 5.1 助记词生成入口
```go
// 位置: crypto/keyring/keyring.go:516-547
func (ks keystore) NewMnemonic(uid string, language Language, hdPath, bip39Passphrase string, algo SignatureAlgo) (Info, string, error) {
    // 1. 验证语言支持
    if language != English {
        return nil, "", ErrUnsupportedLanguage
    }
    
    // 2. 验证算法支持
    if !ks.isSupportedSigningAlgo(algo) {
        return nil, "", ErrUnsupportedSigningAlgo
    }
    
    // 3. 生成熵值
    entropy, err := bip39.NewEntropy(defaultEntropySize)
    if err != nil {
        return nil, "", err
    }
    
    // 4. 生成助记词
    mnemonic, err := bip39.NewMnemonic(entropy)
    if err != nil {
        return nil, "", err
    }
    
    // 5. 设置默认密码短语
    if bip39Passphrase == "" {
        bip39Passphrase = DefaultBIP39Passphrase
    }
    
    // 6. 创建账户
    info, err := ks.NewAccount(uid, mnemonic, bip39Passphrase, hdPath, algo)
    if err != nil {
        return nil, "", err
    }
    
    return info, mnemonic, nil
}
```

### 5.2 PQC 算法派生
```go
// 位置: crypto/hd/algo.go:125-141
func (p pqcAlgo) Derive() DeriveFn {
    return func(mnemonic string, bip39Passphrase, hdPath string) ([]byte, error) {
        // 1. 助记词转种子
        seed, err := bip39.NewSeedWithErrorChecking(mnemonic, bip39Passphrase)
        if err != nil {
            return nil, err
        }
        
        // 2. 计算主密钥
        masterPriv, ch := ComputeMastersFromSeed(seed)
        if len(hdPath) == 0 {
            return masterPriv[:], nil
        }
        
        // 3. 路径派生
        derivedKey, err := DerivePrivateKeyForPath(masterPriv, ch, hdPath)
        return derivedKey, err
    }
}
```

### 5.3 PQC 密钥生成
```go
// 位置: crypto/hd/algo.go:143-148
func (p pqcAlgo) Generate() GenerateFn {
    return func(bz []byte) types.PrivKey {
        return pqc.GenPrivKeyFromSecret(bz)
    }
}
```

## 6. 助记词安全性

### 6.1 熵值安全性
- **256位熵值**: 提供 2^256 种可能
- **暴力破解**: 即使使用全球所有计算资源，也需要数万亿年
- **随机性**: 来自系统加密安全的随机数生成器

### 6.2 助记词验证
- **校验和**: 8位校验和确保助记词完整性
- **错误检测**: 可以检测输入错误
- **单词列表**: 2048个标准单词，避免歧义

### 6.3 密码短语保护
- **额外安全层**: 密码短语提供额外的安全保护
- **默认值**: 空字符串（可通过环境变量设置）
- **存储**: 密码短语不存储在系统中，需要用户记忆

## 7. 助记词恢复流程

### 7.1 从助记词恢复密钥
```go
// 位置: crypto/keyring/keyring.go:549-570
func (ks keystore) NewAccount(name string, mnemonic string, bip39Passphrase string, hdPath string, algo SignatureAlgo) (Info, error) {
    // 1. 验证算法支持
    if !ks.isSupportedSigningAlgo(algo) {
        return nil, ErrUnsupportedSigningAlgo
    }
    
    // 2. 派生私钥
    derivedPriv, err := algo.Derive()(mnemonic, bip39Passphrase, hdPath)
    if err != nil {
        return nil, err
    }
    
    // 3. 生成密钥
    privKey := algo.Generate()(derivedPriv)
    
    // 4. 检查地址冲突
    address := sdk.AccAddress(privKey.PubKey().Address())
    if _, err := ks.KeyByAddress(address); err == nil {
        return nil, fmt.Errorf("account with address %s already exists", address)
    }
    
    // 5. 存储密钥
    return ks.writeLocalKey(name, privKey, algo.Name())
}
```

### 7.2 恢复验证
- **地址检查**: 确保生成的地址不与现有地址冲突
- **密钥验证**: 验证生成的密钥格式正确
- **存储验证**: 确保密钥正确存储到 keyring

## 8. 实际使用示例

### 8.1 生成 PQC 账户
```bash
# 使用 simd 生成 PQC 账户
./build/simd keys add admin --algo falcon-512 --keyring-backend test

# 输出示例：
# - name: admin
# - type: local
# - address: plume1pxgjd8tyzg0f8qv3pms5qswywwqnsdyaf8zeaw
# - pubkey: plumepub1addwnpepq...
# - mnemonic: abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art
```

### 8.2 助记词示例
```
abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art
```

**解析：**
- **单词数量**: 24个
- **熵值**: 256位
- **校验和**: 最后8位
- **安全性**: 2^256 种可能组合

## 9. 技术优势

### 9.1 兼容性
- **BIP-39 标准**: 与现有钱包兼容
- **BIP-44 路径**: 支持分层确定性钱包
- **多语言支持**: 理论上支持多种语言

### 9.2 安全性
- **高熵值**: 256位熵值提供极强安全性
- **校验和**: 内置错误检测机制
- **密码短语**: 可选的额外安全层

### 9.3 用户体验
- **人类可读**: 使用常见英语单词
- **易于备份**: 可以手写记录
- **易于恢复**: 通过助记词完全恢复密钥

## 10. 总结

### 10.1 关键特点
- **标准兼容**: 完全遵循 BIP-39 和 BIP-44 标准
- **高安全性**: 256位熵值 + 密码短语保护
- **易于使用**: 人类可读的单词序列
- **完全恢复**: 通过助记词可以完全恢复密钥

### 10.2 技术流程
```
系统熵值(256位) → 助记词(24词) → 种子(512位) → 主密钥 → HD派生 → PQC密钥
```

### 10.3 安全考虑
- **助记词保密**: 助记词等同于私钥，必须保密
- **安全存储**: 建议离线存储，避免数字形式
- **密码短语**: 建议使用强密码短语增加安全性
- **定期备份**: 确保助记词安全备份

### 10.4 代码位置总结
- **助记词生成**: `crypto/keyring/keyring.go:516-547`
- **熵值常量**: `crypto/keyring/types.go:38`
- **PQC派生**: `crypto/hd/algo.go:125-141`
- **密钥生成**: `crypto/hd/algo.go:143-148`
- **账户创建**: `crypto/keyring/keyring.go:549-570`
