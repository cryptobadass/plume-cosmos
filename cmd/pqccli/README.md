# pqccli 使用说明

`pqccli` 是基于 PQC (Falcon-512) 的命令行工具，用于生成账户、公钥签名及验签。

## 构建

```bash
go build ./cmd/pqccli
```

生成的可执行文件为 `./pqccli`。

## 命令

- 生成账户
  ```bash
  ./pqccli gen
  ```
  输出 JSON，字段说明：
  - `priv_key_hex`: 私钥字节（十六进制，内部为 sk||pk 组合，便于派生公钥）
  - `pub_key_hex`: 公钥字节（十六进制）
  - `address_hex`: 地址（SHA256(pub) 前 20 字节，十六进制）
  - `pqc_address`: 可读地址（前缀 `plume2` + 40 位 hex）
  - `key_type`: 密钥类型（`Falcon-512`）
  - `signature_alg`: 签名算法（同上）

- 签名
  ```bash
  ./pqccli sign <priv_hex> <msg_hex>
  ```
  - `priv_hex`: `gen` 输出的 `priv_key_hex`
  - `msg_hex`: 原始消息字节的十六进制（非哈希，工具内部直接对字节进行签名）
  - 输出：签名（十六进制）

- 验证签名
  ```bash
  ./pqccli verify <pub_hex> <msg_hex> <sig_hex>
  ```
  - `pub_hex`: `gen` 输出的 `pub_key_hex`
  - `msg_hex`: 与签名时相同的消息（十六进制）
  - `sig_hex`: 签名（十六进制）
  - 输出：`true` 或 `false`，并分别以退出码 `0/1` 返回

## 示例

```bash
# 1) 生成账户
./pqccli gen > acct.json
jq -r .priv_key_hex acct.json > priv.hex
jq -r .pub_key_hex  acct.json > pub.hex

# 2) 签名（消息为 "hello"）
MSG_HEX=68656c6c6f
SIG_HEX=$(./pqccli sign $(cat priv.hex) $MSG_HEX)
echo $SIG_HEX

# 3) 验签
./pqccli verify $(cat pub.hex) $MSG_HEX $SIG_HEX && echo OK || echo FAIL
```

## 注意事项

- 私钥格式为 `sk||pk`（Falcon-512：`sk` 1281 字节，`pk` 897 字节），请妥善保管。
- `msg_hex` 为原始消息的十六进制编码；若要对哈希签名，请自行先计算哈希后再传入其十六进制。
- 本工具依赖 `liboqs`（通过 `liboqs-go` 使用）。构建时若有链接警告为正常现象，不影响使用。


