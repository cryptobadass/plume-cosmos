# 本地起链并使用 PQC（Falcon-512）发起交易的最小步骤

本指南展示如何使用 `plume-cosmos` 自带的 `simd` 在本地启动单节点链，并用 PQC 密钥完成一笔转账。

## 快速演示（一键脚本）

```bash
# 构建 simd（若未构建）
go build -o build/simd ./simapp/simd

# 运行一键脚本（使用 keyring-backend=test，免交互）
bash scripts/pqc/simd_pqc_demo.sh
```

脚本会：
- 生成 PQC 账户（falcon-512）和 secp256k1 收款账户
- 构造 bank send 交易（--generate-only）
- 使用 PQC 账户离线签名
- 验证签名

## 手动步骤（最小可复现）

```bash
# 1) 构建
go build -o build/simd ./simapp/simd

# 2) 变量
export HOME_DIR="$(pwd)/.plume_demo"
export CHAIN_ID="plume-demo-1"
export DENOM="uplume"
rm -rf "$HOME_DIR"

# 3) 初始化链
build/simd init plume-demo --chain-id="$CHAIN_ID" --home="$HOME_DIR"

# 4) 创建账户（PQC：falcon-512）
build/simd keys add alice --algo falcon-512 --keyring-backend=test --home="$HOME_DIR"
build/simd keys add bob   --algo falcon-512 --keyring-backend=test --home="$HOME_DIR"

ALICE=$(build/simd keys show alice -a --keyring-backend=test --home="$HOME_DIR")
BOB=$(build/simd keys show bob   -a --keyring-backend=test --home="$HOME_DIR")

# 5) 配置创世资金与验证人
build/simd add-genesis-account "$ALICE" 100000000$DENOM --home="$HOME_DIR"
build/simd add-genesis-account "$BOB"   100000000$DENOM --home="$HOME_DIR"

build/simd gentx alice 5000000$DENOM \
  --chain-id="$CHAIN_ID" --keyring-backend=test --home="$HOME_DIR"

build/simd collect-gentxs   --home="$HOME_DIR"
build/simd validate-genesis --home="$HOME_DIR"

# 6) 启动节点（保持此终端运行）
build/simd start --home="$HOME_DIR"
```

另开一个终端，发起交易并签名：

```bash
# 7) 在线发起一笔转账（PQC 签名）
build/simd tx bank send "$ALICE" "$BOB" 12345$DENOM \
  --fees 1$DENOM --chain-id="$CHAIN_ID" \
  --keyring-backend=test --home="$HOME_DIR"
```

可选：离线构造、签名、验签：

```bash
# 生成未签名交易
build/simd tx bank send "$ALICE" "$BOB" 1$DENOM \
  --fees 1$DENOM --chain-id="$CHAIN_ID" \
  --keyring-backend=test --home="$HOME_DIR" \
  --generate-only > unsigned.json

# 使用 alice（PQC）离线签名
build/simd tx sign unsigned.json --from alice \
  --chain-id="$CHAIN_ID" --keyring-backend=test --home="$HOME_DIR" \
  --offline > signed.json

# 验证签名
build/simd tx validate-signatures signed.json \
  --keyring-backend=test --home="$HOME_DIR"
```

## 说明
- 使用 `--algo falcon-512` 创建的账户会在签名/验签环节走新的 PQC（Falcon-512）实现。
- Gas 计费使用 `x/auth` 参数 `sig_verify_cost_pqc`（与 secp/ed25519 独立配置）。
- `--keyring-backend=test` 为无交互后端，适合脚本与本地测试。
