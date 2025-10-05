# PQC Falcon-512 签名过程详细分析

## 1. 密钥生成过程

### 1.1 随机密钥生成 (`GenPrivKey`)
```go
// 位置: crypto/keys/pqc/pqc.go:212-241
func GenPrivKey() *PrivKey {
    // 1. 初始化 Falcon 签名器
    sig := &oqs.Signature{}
    defer sig.Clean()
    
    // 2. 初始化签名器 (使用系统随机数)
    if err := sig.Init(KeyType, nil); err != nil {
        panic(fmt.Sprintf("Failed to initialize PQC signer: %v", err))
    }
    
    // 3. 生成密钥对
    publicKey, err := sig.GenerateKeyPair()
    if err != nil {
        panic(fmt.Sprintf("Failed to generate PQC key pair: %v", err))
    }
    
    // 4. 导出私钥
    secretKey := sig.ExportSecretKey()
    if secretKey == nil {
        panic("Failed to export secret key")
    }
    
    // 5. 组合存储: 私钥(1281字节) + 公钥(897字节) = 2178字节
    combined := make([]byte, 0, len(secretKey)+len(publicKey))
    combined = append(combined, secretKey...)
    combined = append(combined, publicKey...)
    
    return &PrivKey{Key: combined}
}
```

### 1.2 确定性密钥生成 (`GenPrivKeyFromSecret`)
```go
// 位置: crypto/keys/pqc/pqc.go:243-282
func GenPrivKeyFromSecret(secret []byte) *PrivKey {
    // 1. 种子填充 (最小32字节)
    if len(secret) < 32 {
        padded := make([]byte, 32)
        copy(padded, secret)
        secret = padded
    }
    
    // 2. 使用种子初始化签名器
    sig := &oqs.Signature{}
    defer sig.Clean()
    
    if err := sig.Init(KeyType, secret); err != nil {
        panic(fmt.Sprintf("Failed to initialize PQC signer from secret: %v", err))
    }
    
    // 3. 生成密钥对
    publicKey, err := sig.GenerateKeyPair()
    if err != nil {
        panic(fmt.Sprintf("Failed to generate PQC key pair from secret: %v", err))
    }
    
    // 4. 导出私钥并组合存储
    secretKey := sig.ExportSecretKey()
    combined := make([]byte, 0, len(secretKey)+len(publicKey))
    combined = append(combined, secretKey...)
    combined = append(combined, publicKey...)
    
    return &PrivKey{Key: combined}
}
```

## 2. HD 密钥派生过程

### 2.1 HD 派生实现
```go
// 位置: crypto/hd/algo.go:125-147
type pqcAlgo struct{}

func (p pqcAlgo) Name() PubKeyType {
    return PQCType // "falcon-512"
}

// 派生函数: 从助记词生成种子
func (p pqcAlgo) Derive() DeriveFn {
    return func(mnemonic string, bip39Passphrase, hdPath string) ([]byte, error) {
        // 1. 从助记词生成种子
        seed, err := bip39.NewSeedWithErrorChecking(mnemonic, bip39Passphrase)
        if err != nil {
            return nil, err
        }
        
        // 2. 计算主私钥和链码
        masterPriv, ch := ComputeMastersFromSeed(seed)
        if len(hdPath) == 0 {
            return masterPriv[:], nil
        }
        
        // 3. 根据HD路径派生私钥
        derivedKey, err := DerivePrivateKeyForPath(masterPriv, ch, hdPath)
        return derivedKey, err
    }
}

// 生成函数: 从种子生成PQC私钥
func (p pqcAlgo) Generate() GenerateFn {
    return func(bz []byte) types.PrivKey {
        return pqc.GenPrivKeyFromSecret(bz) // 调用确定性生成
    }
}
```

## 3. 密钥存储过程

### 3.1 Keyring 存储 (`writeLocalKey`)
```go
// 位置: crypto/keyring/keyring.go:759-787
func (ks keystore) writeLocalKey(name string, priv types.PrivKey, algo hd.PubKeyType) (Info, error) {
    pub := priv.PubKey()
    var info Info
    
    if algo == hd.PQCType {
        // PQC 密钥处理
        typedPriv := priv.(*pqc.PrivKey)
        aminoBytes, err := typedPriv.MarshalAmino()
        if err != nil {
            return nil, err
        }
        // 使用 base64 编码存储二进制数据
        privArmor := base64.StdEncoding.EncodeToString(aminoBytes)
        info = newLocalInfo(name, pub, privArmor, algo)
    }
    
    if err := ks.writeInfo(info); err != nil {
        return nil, err
    }
    return info, nil
}
```

### 3.2 密钥序列化 (`MarshalAmino`)
```go
// 位置: crypto/keys/pqc/pqc.go:91-95
func (privKey PrivKey) MarshalAmino() ([]byte, error) {
    return privKey.Key, nil // 直接返回组合的密钥数据
}
```

## 4. 签名过程

### 4.1 Keyring 签名 (`Sign`)
```go
// 位置: crypto/keyring/keyring.go:315-366
func (ks keystore) Sign(uid string, msg []byte) ([]byte, types.PubKey, error) {
    info, err := ks.Key(uid)
    if err != nil {
        return nil, nil, err
    }
    
    var priv types.PrivKey
    
    switch i := info.(type) {
    case LocalInfo:
        if i.Algo == hd.PQCType {
            // 1. 从 base64 解码私钥
            privBytes, err := base64.StdEncoding.DecodeString(i.PrivKeyArmor)
            if err != nil {
                return nil, nil, fmt.Errorf("failed to decode PQC private key from base64: %w", err)
            }
            
            // 2. 反序列化私钥
            typedPriv := &pqc.PrivKey{}
            if err := typedPriv.UnmarshalAmino(privBytes); err != nil {
                return nil, nil, err
            }
            priv = typedPriv
        }
    }
    
    // 3. 调用 PQC 签名
    sig, err := priv.Sign(msg)
    if err != nil {
        return nil, nil, err
    }
    
    return sig, priv.PubKey(), nil
}
```

### 4.2 PQC 签名实现 (`Sign`)
```go
// 位置: crypto/keys/pqc/pqc.go:28-52
func (privKey *PrivKey) Sign(msg []byte) ([]byte, error) {
    // 1. 验证密钥长度
    if len(privKey.Key) != SecretKeyLen+PublicKeyLen {
        return nil, fmt.Errorf("invalid pqc private key size: expected %d bytes, got %d", 
            SecretKeyLen+PublicKeyLen, len(privKey.Key))
    }
    
    // 2. 提取私钥部分 (前1281字节)
    secretKey := privKey.Key[:SecretKeyLen]
    
    // 3. 初始化 Falcon 签名器
    sig := &oqs.Signature{}
    defer sig.Clean()
    
    if err := sig.Init(KeyType, secretKey); err != nil {
        return nil, fmt.Errorf("failed to initialize PQC signer: %w", err)
    }
    
    // 4. 签名消息
    signature, err := sig.Sign(msg)
    if err != nil {
        return nil, fmt.Errorf("failed to sign message: %w", err)
    }
    
    return signature, nil
}
```

## 5. 公钥提取过程

### 5.1 公钥提取 (`PubKey`)
```go
// 位置: crypto/keys/pqc/pqc.go:54-64
func (privKey *PrivKey) PubKey() types.PubKey {
    // 验证密钥长度
    if len(privKey.Key) != SecretKeyLen+PublicKeyLen {
        panic(fmt.Sprintf("invalid pqc private key size: expected %d bytes, got %d", 
            SecretKeyLen+PublicKeyLen, len(privKey.Key)))
    }
    
    // 提取公钥部分 (后897字节)
    publicKey := privKey.Key[SecretKeyLen:]
    
    return &PubKey{Key: publicKey}
}
```

## 6. 签名验证过程

### 6.1 PQC 签名验证 (`VerifySignature`)
```go
// 位置: crypto/keys/pqc/pqc.go:132-153
func (pubKey PubKey) VerifySignature(msg []byte, sig []byte) bool {
    // 1. 验证公钥和签名长度
    if len(pubKey.Key) != PublicKeyLen || len(sig) == 0 {
        return false
    }
    
    // 2. 初始化 Falcon 验证器
    verifier := &oqs.Signature{}
    defer verifier.Clean()
    
    if err := verifier.Init(KeyType, nil); err != nil {
        return false
    }
    
    // 3. 验证签名
    valid, err := verifier.Verify(msg, sig, pubKey.Key)
    if err != nil {
        return false
    }
    
    return valid
}
```

## 7. 交易签名验证过程

### 7.1 Ante Handler 签名验证
```go
// 位置: x/auth/ante/sigverify.go:258-301
func (svd SigVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
    // 1. 检查 PQC 交易限制
    if err := checkPQCTransactionRestrictions(ctx, svd.ak, sigTx); err != nil {
        return ctx, err
    }
    
    // 2. 遍历所有签名者
    for i, sig := range sigs {
        acc, err := GetSignerAcc(ctx, svd.ak, signerAddrs[i])
        if err != nil {
            return ctx, err
        }
        
        // 3. 获取公钥
        pubKey := acc.GetPubKey()
        
        // 4. 验证签名 (调用 Cosmos SDK 的签名验证)
        if !simulate && !ctx.IsReCheckTx() {
            err := authsigning.VerifySignature(pubKey, signerData, sig.Data, svd.signModeHandler, sigTx)
            if err != nil {
                return ctx, err
            }
        }
    }
    
    return next(ctx, tx, simulate)
}
```

### 7.2 PQC 交易限制检查
```go
// 位置: x/auth/ante/sigverify.go:547-591
func checkPQCTransactionRestrictions(ctx sdk.Context, ak AccountKeeper, sigTx authsigning.SigVerifiableTx) error {
    signerAddrs := sigTx.GetSigners()
    
    // 获取所有账户
    var accounts []types.AccountI
    for _, addr := range signerAddrs {
        acc, err := GetSignerAcc(ctx, ak, addr)
        if err != nil {
            return err
        }
        accounts = append(accounts, acc)
    }
    
    // 检查是否有 PQC 账户
    hasPQCAccount := false
    for _, acc := range accounts {
        pubKey := acc.GetPubKey()
        if pubKey != nil {
            if _, ok := pubKey.(*pqc.PubKey); ok {
                hasPQCAccount = true
                break
            }
        }
    }
    
    // 如果没有 PQC 账户，无限制
    if !hasPQCAccount {
        return nil
    }
    
    // 如果有 PQC 账户，所有账户都必须是 PQC
    for _, acc := range accounts {
        pubKey := acc.GetPubKey()
        if pubKey != nil {
            if _, ok := pubKey.(*pqc.PubKey); !ok {
                return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized,
                    "PQC accounts can only transact with other PQC accounts. Account %s uses %T key type",
                    acc.GetAddress().String(), pubKey)
            }
        }
    }
    
    return nil
}
```

## 8. 签名流程总结

### 8.1 完整签名流程
```
1. 密钥生成
   ├── 随机生成: GenPrivKey() → 使用系统随机数
   └── 确定性生成: GenPrivKeyFromSecret() → 使用种子

2. 密钥存储
   ├── MarshalAmino() → 序列化密钥数据
   ├── base64 编码 → 存储为字符串
   └── 写入 Keyring → 持久化存储

3. 签名过程
   ├── 从 Keyring 读取密钥
   ├── base64 解码
   ├── UnmarshalAmino() 反序列化
   ├── 提取私钥部分 (前1281字节)
   ├── sig.Init(KeyType, secretKey) 初始化签名器
   ├── sig.Sign(msg) 签名消息
   └── 返回签名结果

4. 验证过程
   ├── 提取公钥部分 (后897字节)
   ├── verifier.Init(KeyType, nil) 初始化验证器
   ├── verifier.Verify(msg, sig, pubKey) 验证签名
   └── 返回验证结果
```

### 8.2 密钥结构
```
PrivKey.Key (2178字节):
├── 私钥部分 (1281字节) - 用于签名
└── 公钥部分 (897字节)  - 用于验证
```

### 8.3 关键常量
- `KeyType = "Falcon-512"`
- `SecretKeyLen = 1281` (Falcon-512 私钥长度)
- `PublicKeyLen = 897`  (Falcon-512 公钥长度)
- 签名长度: 约656字节 (Falcon-512 签名长度)

### 8.4 限制条件
- PQC 账户只能与 PQC 账户进行交易
- 不支持 PQC 账户与其他类型账户的混合交易
- 签名验证使用 secp256k1 的 gas 成本作为近似值
