# PQC 账户地址生成流程代码分析

## 1. 地址生成概述

PQC 账户的地址生成遵循 Cosmos SDK 的标准流程：
1. 从 PQC 公钥生成原始地址字节（20字节）
2. 使用 Bech32 编码转换为人类可读格式
3. 应用项目特定的前缀（如 `plume1`）

## 2. PQC 公钥地址生成

### 2.1 PQC PubKey Address 方法
```go
// 位置: crypto/keys/pqc/pqc.go:119-125
func (pubKey PubKey) Address() crypto.Address {
    // 使用公钥的前20字节作为地址
    addr := make([]byte, 20)
    copy(addr, pubKey.Key[:20])
    return addr
}
```

**关键特点：**
- PQC 公钥长度为 897 字节
- 地址取公钥的前 20 字节
- 返回类型为 `crypto.Address`（即 `[]byte`）

## 3. 地址类型转换

### 3.1 从公钥地址到 Account Address
```go
// 位置: crypto/keyring/info.go:71-73
func (i LocalInfo) GetAddress() types.AccAddress {
    return i.PubKey.Address().Bytes()
}
```

**流程：**
1. 调用 `PubKey.Address()` 获取原始地址字节
2. 调用 `.Bytes()` 转换为 `[]byte`
3. 包装为 `types.AccAddress` 类型

### 3.2 AccAddress 类型定义
```go
// 位置: types/address.go:127-129
type AccAddress []byte
```

## 4. Bech32 编码

### 4.1 AccAddress String 方法
```go
// 位置: types/address.go:273-287
func (aa AccAddress) String() string {
    if aa.Empty() {
        return ""
    }

    var key = conv.UnsafeBytesToStr(aa)
    accAddrMu.Lock()
    defer accAddrMu.Unlock()
    addr, ok := accAddrCache.Get(key)
    if ok {
        return addr
    }
    return cacheBech32Addr(GetConfig().GetBech32AccountAddrPrefix(), aa, accAddrCache, key)
}
```

**特点：**
- 使用缓存机制提高性能
- 调用 `cacheBech32Addr` 进行 Bech32 编码
- 使用配置的地址前缀

### 4.2 cacheBech32Addr 函数
```go
// 位置: types/address.go:663-671
func cacheBech32Addr(prefix string, addr []byte, cache *simplelru.LRU[string, string], cacheKey string) string {
    bech32Addr, err := bech32.ConvertAndEncode(prefix, addr)
    if err != nil {
        panic(err)
    }
    cache.Add(cacheKey, bech32Addr)
    return bech32Addr
}
```

**功能：**
- 使用 `bech32.ConvertAndEncode` 进行编码
- 缓存编码结果
- 返回完整的 Bech32 地址字符串

## 5. 地址前缀配置

### 5.1 默认前缀配置
```go
// 位置: types/address.go:35-74
const (
    // Bech32MainPrefix 定义主 SDK Bech32 前缀
    Bech32MainPrefix = "cosmos"
    
    // Bech32PrefixAccAddr 定义账户地址的 Bech32 前缀
    Bech32PrefixAccAddr = Bech32MainPrefix  // "cosmos"
    
    // Bech32PrefixAccPub 定义账户公钥的 Bech32 前缀
    Bech32PrefixAccPub = Bech32MainPrefix + PrefixPublic  // "cosmospub"
)
```

### 5.2 项目特定前缀设置
```go
// 位置: types/config.go:85-91
func (config *Config) SetBech32PrefixForAccount(addressPrefix, pubKeyPrefix string) {
    config.assertNotSealed()
    config.bech32AddressPrefix["account_addr"] = addressPrefix
    config.bech32AddressPrefix["account_pub"] = pubKeyPrefix
}
```

**示例配置：**
```go
config := sdk.GetConfig()
config.SetBech32PrefixForAccount("plume", "plumepub")
config.Seal()
```

## 6. 完整地址生成流程

### 6.1 流程步骤
```
1. PQC密钥生成
   └── 生成897字节的公钥

2. 地址提取
   └── pubKey.Address() 提取前20字节
   └── 返回 crypto.Address ([]byte)

3. 类型转换
   └── 转换为 types.AccAddress

4. Bech32编码
   └── AccAddress.String() 调用
   └── cacheBech32Addr() 编码
   └── bech32.ConvertAndEncode() 最终编码

5. 前缀应用
   └── 使用配置的前缀 (如 "plume")
   └── 生成最终地址 (如 "plume1pxgjd8tyzg0f8qv3pms5qswywwqnsdyaf8zeaw")
```

### 6.2 代码调用链
```
PQC账户生成
    ↓
pubKey.Address()           // crypto/keys/pqc/pqc.go:119-125
    ↓
i.PubKey.Address().Bytes() // crypto/keyring/info.go:71-73
    ↓
AccAddress.String()        // types/address.go:273-287
    ↓
cacheBech32Addr()          // types/address.go:663-671
    ↓
bech32.ConvertAndEncode()  // types/bech32/bech32.go
    ↓
最终Bech32地址
```

## 7. 地址验证

### 7.1 地址格式验证
```go
// 位置: types/address.go:137-155
func VerifyAddressFormat(bz []byte) error {
    verifier := GetConfig().GetAddressVerifier()
    if verifier != nil {
        return verifier(bz)
    }

    if len(bz) == 0 {
        return sdkerrors.Wrap(sdkerrors.ErrUnknownAddress, "addresses cannot be empty")
    }

    if len(bz) > address.MaxAddrLen {
        return sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "address max length is %d, got %d", address.MaxAddrLen, len(bz))
    }

    return nil
}
```

### 7.2 地址解码
```go
// 位置: types/address.go:167-186
func AccAddressFromBech32(address string) (addr AccAddress, err error) {
    if len(strings.TrimSpace(address)) == 0 {
        return AccAddress{}, errors.New("empty address string is not allowed")
    }

    bech32PrefixAccAddr := GetConfig().GetBech32AccountAddrPrefix()

    bz, err := GetFromBech32(address, bech32PrefixAccAddr)
    if err != nil {
        return nil, err
    }

    err = VerifyAddressFormat(bz)
    if err != nil {
        return nil, err
    }

    return AccAddress(bz), nil
}
```

## 8. 实际示例

### 8.1 PQC 地址生成示例
```go
// 1. PQC公钥 (897字节)
pubKey := &pqc.PubKey{Key: [897]byte{...}}

// 2. 提取地址 (前20字节)
addr := pubKey.Address()  // []byte, 20字节

// 3. 转换为AccAddress
accAddr := types.AccAddress(addr)

// 4. Bech32编码
addrStr := accAddr.String()  // "plume1pxgjd8tyzg0f8qv3pms5qswywwqnsdyaf8zeaw"
```

### 8.2 地址结构分析
```
原始地址字节: [20字节]
    ↓
Bech32编码: 
    - 前缀: "plume"
    - 数据: [20字节] 的 base32 编码
    - 校验和: 6字节
    ↓
最终地址: "plume1pxgjd8tyzg0f8qv3pms5qswywwqnsdyaf8zeaw"
```

## 9. 地址缓存机制

### 9.1 缓存配置
```go
// 位置: types/address.go:88-101
func init() {
    var err error
    // 缓存大小: 61k 条目
    // 键: 32字节，值: 50-70字节
    // 总内存: ~11 MB
    if accAddrCache, err = simplelru.NewLRU[string, string](60000, nil); err != nil {
        panic(err)
    }
}
```

### 9.2 缓存使用
```go
// 位置: types/address.go:273-287
func (aa AccAddress) String() string {
    var key = conv.UnsafeBytesToStr(aa)
    accAddrMu.Lock()
    defer accAddrMu.Unlock()
    addr, ok := accAddrCache.Get(key)
    if ok {
        return addr  // 缓存命中
    }
    return cacheBech32Addr(GetConfig().GetBech32AccountAddrPrefix(), aa, accAddrCache, key)
}
```

## 10. 总结

### 10.1 关键特点
- **地址长度**: 20字节（与 secp256k1 相同）
- **生成方式**: 取 PQC 公钥前 20 字节
- **编码格式**: Bech32
- **前缀可配置**: 支持项目特定前缀
- **缓存优化**: LRU 缓存提高性能

### 10.2 技术优势
1. **兼容性**: 与现有 Cosmos SDK 地址系统完全兼容
2. **安全性**: 20字节地址提供足够的熵
3. **可读性**: Bech32 编码提供人类可读格式
4. **性能**: 缓存机制优化编码性能
5. **灵活性**: 支持自定义地址前缀

### 10.3 代码位置总结
- **PQC地址生成**: `crypto/keys/pqc/pqc.go:119-125`
- **类型转换**: `crypto/keyring/info.go:71-73`
- **Bech32编码**: `types/address.go:273-287`
- **缓存机制**: `types/address.go:663-671`
- **前缀配置**: `types/config.go:85-91`
