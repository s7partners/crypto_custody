package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// SignerService 签名服务
// - 如果配置了 remoteURL，则请求远程服务
// - 否则用 localPrivKey 本地签名（仅测试用）
type SignerService struct {
	remoteURL    string
	localPrivKey *ecdsa.PrivateKey
	chainID      int64
}

// NewSignerService 创建签名服务
func NewSignerService(remoteURL string, localPrivHex string, chainID int64) (*SignerService, error) {
	var key *ecdsa.PrivateKey
	if remoteURL == "" && localPrivHex != "" {
		priv, err := crypto.HexToECDSA(strings.TrimPrefix(localPrivHex, "0x"))
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %w", err)
		}
		key = priv
	}
	return &SignerService{
		remoteURL:    remoteURL,
		localPrivKey: key,
		chainID:      chainID,
	}, nil
}

// Sign 负责对未签名交易进行签名，返回签名后的 RLP
func (s *SignerService) Sign(ctx context.Context, unsignedJSON []byte) ([]byte, error) {
	// 远程签名服务
	if s.remoteURL != "" {
		reqBody := map[string]string{"unsigned_tx": string(unsignedJSON)}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequestWithContext(ctx, "POST", s.remoteURL+"/sign", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("remote signer returned %d", resp.StatusCode)
		}
		var respObj map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&respObj); err != nil {
			return nil, err
		}
		signedHex := respObj["signed_tx"]
		return hex.DecodeString(strings.TrimPrefix(signedHex, "0x"))
	}

	// 本地签名（仅测试）
	if s.localPrivKey == nil {
		return nil, errors.New("no signer configured")
	}
	var tx types.Transaction
	if err := tx.UnmarshalJSON(unsignedJSON); err != nil {
		return nil, fmt.Errorf("unmarshal unsigned tx: %w", err)
	}
	signer := types.NewLondonSigner(new(big.Int).SetInt64(s.chainID))
	signedTx, err := types.SignTx(&tx, signer, s.localPrivKey)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := signedTx.EncodeRLP(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
