# PQC 交易流程代码分析：从发起交易到签名验证

## 1. 交易发起 - CLI 命令

### 1.1 用户执行命令
```bash
./build/simd tx bank send admin plume1p9xdv2q3623zsyykk6aecj7lpvfpt65dreqydr 1000uplume --chain-id test-chain --home ~/.simapp --fees 10000uplume --keyring-backend test --yes
```

### 1.2 Bank Send 命令实现
```go
// 位置: x/bank/client/cli/tx.go:28-64
func NewSendTxCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use: "send [from_key_or_address] [to_address] [amount]",
        Args: cobra.ExactArgs(3),
        RunE: func(cmd *cobra.Command, args []string) error {
            // 1. 设置发送者
            cmd.Flags().Set(flags.FlagFrom, args[0])
            clientCtx, err := client.GetClientTxContext(cmd)
            if err != nil {
                return err
            }

            // 2. 解析金额
            coins, err := sdk.ParseCoinsNormalized(args[2])
            if err != nil {
                return err
            }

            // 3. 创建 MsgSend 消息
            msg := &types.MsgSend{
                FromAddress: clientCtx.GetFromAddress().String(),
                ToAddress:   args[1],
                Amount:      coins,
            }
            if err := msg.ValidateBasic(); err != nil {
                return err
            }

            // 4. 生成或广播交易
            return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
        },
    }
    
    flags.AddTxFlagsToCmd(cmd)
    return cmd
}
```

## 2. 交易生成和广播

### 2.1 GenerateOrBroadcastTxCLI
```go
// 位置: client/tx/tx.go:26-31
func GenerateOrBroadcastTxCLI(clientCtx client.Context, flagSet *pflag.FlagSet, msgs ...sdk.Msg) error {
    txf := NewFactoryCLI(clientCtx, flagSet)
    return GenerateOrBroadcastTxWithFactory(clientCtx, txf, msgs...)
}

// 位置: client/tx/tx.go:35-51
func GenerateOrBroadcastTxWithFactory(clientCtx client.Context, txf Factory, msgs ...sdk.Msg) error {
    // 1. 验证所有消息
    for _, msg := range msgs {
        if err := msg.ValidateBasic(); err != nil {
            return err
        }
    }

    // 2. 根据模式决定生成或广播
    if clientCtx.GenerateOnly {
        return GenerateTx(clientCtx, txf, msgs...)
    }

    return BroadcastTx(clientCtx, txf, msgs...)
}
```

### 2.2 BroadcastTx - 核心广播流程
```go
// 位置: client/tx/tx.go:87-148
func BroadcastTx(clientCtx client.Context, txf Factory, msgs ...sdk.Msg) error {
    // 1. 准备交易工厂
    txf, err := prepareFactory(clientCtx, txf)
    if err != nil {
        return err
    }

    // 2. 模拟和计算 Gas
    if txf.SimulateAndExecute() || clientCtx.Simulate {
        _, adjusted, err := CalculateGas(clientCtx, txf, msgs...)
        if err != nil {
            return err
        }
        txf = txf.WithGas(adjusted)
    }

    // 3. 构建未签名交易
    tx, err := BuildUnsignedTx(txf, msgs...)
    if err != nil {
        return err
    }

    // 4. 用户确认 (如果不是 --yes)
    if !clientCtx.SkipConfirm {
        // 显示交易内容并等待用户确认
        out, err := clientCtx.TxConfig.TxJSONEncoder()(tx.GetTx())
        if err != nil {
            return err
        }
        
        buf := bufio.NewReader(os.Stdin)
        ok, err := input.GetConfirmation("confirm transaction before signing and broadcasting", buf, os.Stderr)
        if err != nil || !ok {
            return err
        }
    }

    // 5. 设置费用授权者
    tx.SetFeeGranter(clientCtx.GetFeeGranterAddress())
    
    // 6. 签名交易 (关键步骤)
    err = Sign(txf, clientCtx.GetFromName(), tx, true)
    if err != nil {
        return err
    }

    // 7. 编码交易
    txBytes, err := clientCtx.TxConfig.TxEncoder()(tx.GetTx())
    if err != nil {
        return err
    }

    // 8. 广播到 Tendermint 节点
    res, err := clientCtx.BroadcastTx(txBytes)
    if err != nil {
        return err
    }

    return clientCtx.PrintProto(res)
}
```

## 3. PQC 签名过程

### 3.1 Sign 函数 - 交易签名入口
```go
// 位置: client/tx/tx.go:334-415
func Sign(txf Factory, name string, txBuilder client.TxBuilder, overwriteSig bool) error {
    // 1. 检查 keybase
    if txf.keybase == nil {
        return errors.New("keybase must be set prior to signing a transaction")
    }

    // 2. 获取签名模式
    signMode := txf.signMode
    if signMode == signing.SignMode_SIGN_MODE_UNSPECIFIED {
        signMode = txf.txConfig.SignModeHandler().DefaultMode()
    }

    // 3. 检查多签名者限制
    if err := checkMultipleSigners(signMode, txBuilder.GetTx()); err != nil {
        return err
    }

    // 4. 从 keybase 获取密钥信息
    key, err := txf.keybase.Key(name)
    if err != nil {
        return err
    }
    pubKey := key.GetPubKey()

    // 5. 准备签名者数据
    signerData := authsigning.SignerData{
        ChainID:       txf.chainID,
        AccountNumber: txf.accountNumber,
        Sequence:      txf.sequence,
    }

    // 6. 创建初始签名结构
    sigData := signing.SingleSignatureData{
        SignMode:  signMode,
        Signature: nil,
    }
    sig := signing.SignatureV2{
        PubKey:   pubKey,
        Data:     &sigData,
        Sequence: txf.Sequence(),
    }

    // 7. 设置签名到交易构建器
    if err := txBuilder.SetSignatures(sig); err != nil {
        return err
    }

    // 8. 生成待签名字节
    bytesToSign, err := txf.txConfig.SignModeHandler().GetSignBytes(signMode, signerData, txBuilder.GetTx())
    if err != nil {
        return err
    }

    // 9. 使用 keybase 签名 (关键步骤)
    sigBytes, _, err := txf.keybase.Sign(name, bytesToSign)
    if err != nil {
        return err
    }

    // 10. 构造最终签名结构
    sigData = signing.SingleSignatureData{
        SignMode:  signMode,
        Signature: sigBytes,
    }
    sig = signing.SignatureV2{
        PubKey:   pubKey,
        Data:     &sigData,
        Sequence: txf.Sequence(),
    }

    // 11. 设置最终签名
    if overwriteSig {
        return txBuilder.SetSignatures(sig)
    }
    prevSignatures = append(prevSignatures, sig)
    return txBuilder.SetSignatures(prevSignatures...)
}
```

### 3.2 Keyring Sign 方法 - PQC 签名实现
```go
// 位置: crypto/keyring/keyring.go:315-366
func (ks keystore) Sign(uid string, msg []byte) ([]byte, types.PubKey, error) {
    // 1. 获取密钥信息
    info, err := ks.Key(uid)
    if err != nil {
        return nil, nil, err
    }

    var priv types.PrivKey

    // 2. 处理不同类型的密钥信息
    switch i := info.(type) {
    case LocalInfo:
        if i.PrivKeyArmor == "" {
            return nil, nil, fmt.Errorf("private key not available")
        }

        // 3. PQC 密钥处理
        if i.Algo == hd.PQCType {
            // 从 base64 解码私钥
            privBytes, err := base64.StdEncoding.DecodeString(i.PrivKeyArmor)
            if err != nil {
                return nil, nil, fmt.Errorf("failed to decode PQC private key from base64: %w", err)
            }
            
            // 反序列化私钥
            typedPriv := &pqc.PrivKey{}
            if err := typedPriv.UnmarshalAmino(privBytes); err != nil {
                return nil, nil, err
            }
            priv = typedPriv
        }
        // ... 其他密钥类型处理
    }

    // 4. 调用 PQC 签名
    sig, err := priv.Sign(msg)
    if err != nil {
        return nil, nil, err
    }

    return sig, priv.PubKey(), nil
}
```

### 3.3 PQC Sign 方法 - Falcon-512 签名
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

## 4. 交易广播

### 4.1 BroadcastTx 方法
```go
// 位置: client/broadcast.go:51-67
func (ctx Context) BroadcastTx(txBytes []byte) (res *sdk.TxResponse, err error) {
    switch ctx.BroadcastMode {
    case flags.BroadcastSync:
        res, err = ctx.BroadcastTxSync(txBytes)
    case flags.BroadcastAsync:
        res, err = ctx.BroadcastTxAsync(txBytes)
    case flags.BroadcastBlock:
        res, err = ctx.BroadcastTxCommit(txBytes)
    default:
        return nil, fmt.Errorf("unsupported return type %s; supported types: sync, async, block", ctx.BroadcastMode)
    }
    return res, err
}
```

## 5. 节点处理 - Ante Handler 验证

### 5.1 SigVerificationDecorator AnteHandle
```go
// 位置: x/auth/ante/sigverify.go:238-301
func (svd SigVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
    // 1. 获取可签名交易接口
    sigTx, ok := tx.(authsigning.SigVerifiableTx)
    if !ok {
        return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type")
    }

    // 2. 获取签名列表
    sigs, err := sigTx.GetSignaturesV2()
    if err != nil {
        return ctx, err
    }

    signerAddrs := sigTx.GetSigners()

    // 3. 检查 PQC 交易限制
    if err := checkPQCTransactionRestrictions(ctx, svd.ak, sigTx); err != nil {
        return ctx, err
    }

    // 4. 验证每个签名
    for i, sig := range sigs {
        // 获取账户
        acc, err := GetSignerAcc(ctx, svd.ak, signerAddrs[i])
        if err != nil {
            return ctx, err
        }

        // 获取公钥
        pubKey := acc.GetPubKey()
        if !simulate && pubKey == nil {
            return ctx, sdkerrors.Wrap(sdkerrors.ErrInvalidPubKey, "pubkey on account is not set")
        }

        // 准备签名者数据
        genesis := ctx.BlockHeight() == 0
        chainID := ctx.ChainID()
        var accNum uint64
        if !genesis {
            accNum = acc.GetAccountNumber()
        }
        signerData := authsigning.SignerData{
            ChainID:       chainID,
            AccountNumber: accNum,
            Sequence:      acc.GetSequence(),
        }

        // 验证签名
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

### 5.2 PQC 交易限制检查
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

### 5.3 PQC 签名验证
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

## 6. 消息处理 - Bank Module

### 6.1 MsgSend 处理
```go
// 位置: x/bank/keeper/msg_server.go:26-76
func (k msgServer) Send(goCtx context.Context, msg *types.MsgSend) (*types.MsgSendResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)

    // 1. 检查发送是否启用
    if err := k.IsSendEnabledCoins(ctx, msg.Amount...); err != nil {
        return nil, err
    }

    // 2. 解析地址
    from, err := sdk.AccAddressFromBech32(msg.FromAddress)
    if err != nil {
        return nil, err
    }
    to, err := sdk.AccAddressFromBech32(msg.ToAddress)
    if err != nil {
        return nil, err
    }

    // 3. 检查权限
    allowListCache := make(map[string]AllowedAddresses)
    if !k.IsInDenomAllowList(ctx, from, msg.Amount, allowListCache) {
        return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to send funds", msg.FromAddress)
    }

    if k.BlockedAddr(to) || !k.IsInDenomAllowList(ctx, to, msg.Amount, allowListCache) {
        return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", msg.ToAddress)
    }

    // 4. 执行转账
    err = k.SendCoins(ctx, from, to, msg.Amount)
    if err != nil {
        return nil, err
    }

    // 5. 发送事件
    defer func() {
        for _, a := range msg.Amount {
            if a.Amount.IsInt64() {
                telemetry.SetGaugeWithLabels(
                    []string{"tx", "msg", "send"},
                    float32(a.Amount.Int64()),
                    []metrics.Label{telemetry.NewLabel("denom", a.Denom)},
                )
            }
        }
    }()

    ctx.EventManager().EmitEvent(
        sdk.NewEvent(
            sdk.EventTypeMessage,
            sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
        ),
    )

    return &types.MsgSendResponse{}, nil
}
```

## 7. 完整流程总结

### 7.1 交易流程
```
1. 用户执行命令
   └── ./build/simd tx bank send admin ta0 1000uplume

2. CLI 命令处理
   └── x/bank/client/cli/tx.go:NewSendTxCmd()
   └── 创建 MsgSend 消息
   └── 调用 tx.GenerateOrBroadcastTxCLI()

3. 交易生成和广播
   └── client/tx/tx.go:BroadcastTx()
   └── 构建未签名交易
   └── 调用 Sign() 签名

4. PQC 签名过程
   └── client/tx/tx.go:Sign()
   └── crypto/keyring/keyring.go:Sign()
   └── crypto/keys/pqc/pqc.go:Sign()
   └── 使用 Falcon-512 算法签名

5. 交易广播
   └── client/broadcast.go:BroadcastTx()
   └── 发送到 Tendermint 节点

6. 节点验证
   └── x/auth/ante/sigverify.go:AnteHandle()
   └── 检查 PQC 交易限制
   └── 验证 PQC 签名

7. 消息处理
   └── x/bank/keeper/msg_server.go:Send()
   └── 执行实际转账操作
```

### 7.2 关键代码位置
- **CLI 命令**: `x/bank/client/cli/tx.go:28-64`
- **交易广播**: `client/tx/tx.go:87-148`
- **签名入口**: `client/tx/tx.go:334-415`
- **Keyring 签名**: `crypto/keyring/keyring.go:315-366`
- **PQC 签名**: `crypto/keys/pqc/pqc.go:28-52`
- **签名验证**: `x/auth/ante/sigverify.go:238-301`
- **消息处理**: `x/bank/keeper/msg_server.go:26-76`
