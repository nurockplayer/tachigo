package contract

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"sync"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const tachiTokenABI = `[{"type":"constructor","inputs":[],"stateMutability":"nonpayable"},{"type":"function","name":"MAX_SUPPLY","inputs":[],"outputs":[{"name":"","type":"uint256","internalType":"uint256"}],"stateMutability":"view"},{"type":"function","name":"allowance","inputs":[{"name":"owner","type":"address","internalType":"address"},{"name":"spender","type":"address","internalType":"address"}],"outputs":[{"name":"","type":"uint256","internalType":"uint256"}],"stateMutability":"view"},{"type":"function","name":"approve","inputs":[{"name":"","type":"address","internalType":"address"},{"name":"","type":"uint256","internalType":"uint256"}],"outputs":[{"name":"","type":"bool","internalType":"bool"}],"stateMutability":"pure"},{"type":"function","name":"balanceOf","inputs":[{"name":"account","type":"address","internalType":"address"}],"outputs":[{"name":"","type":"uint256","internalType":"uint256"}],"stateMutability":"view"},{"type":"function","name":"burn","inputs":[{"name":"from","type":"address","internalType":"address"},{"name":"amount","type":"uint256","internalType":"uint256"}],"outputs":[],"stateMutability":"nonpayable"},{"type":"function","name":"decimals","inputs":[],"outputs":[{"name":"","type":"uint8","internalType":"uint8"}],"stateMutability":"view"},{"type":"function","name":"mint","inputs":[{"name":"to","type":"address","internalType":"address"},{"name":"amount","type":"uint256","internalType":"uint256"}],"outputs":[],"stateMutability":"nonpayable"},{"type":"function","name":"name","inputs":[],"outputs":[{"name":"","type":"string","internalType":"string"}],"stateMutability":"view"},{"type":"function","name":"owner","inputs":[],"outputs":[{"name":"","type":"address","internalType":"address"}],"stateMutability":"view"},{"type":"function","name":"renounceOwnership","inputs":[],"outputs":[],"stateMutability":"nonpayable"},{"type":"function","name":"symbol","inputs":[],"outputs":[{"name":"","type":"string","internalType":"string"}],"stateMutability":"view"},{"type":"function","name":"totalSupply","inputs":[],"outputs":[{"name":"","type":"uint256","internalType":"uint256"}],"stateMutability":"view"},{"type":"function","name":"transfer","inputs":[{"name":"","type":"address","internalType":"address"},{"name":"","type":"uint256","internalType":"uint256"}],"outputs":[{"name":"","type":"bool","internalType":"bool"}],"stateMutability":"pure"},{"type":"function","name":"transferFrom","inputs":[{"name":"","type":"address","internalType":"address"},{"name":"","type":"address","internalType":"address"},{"name":"","type":"uint256","internalType":"uint256"}],"outputs":[{"name":"","type":"bool","internalType":"bool"}],"stateMutability":"pure"},{"type":"function","name":"transferOwnership","inputs":[{"name":"newOwner","type":"address","internalType":"address"}],"outputs":[],"stateMutability":"nonpayable"},{"type":"event","name":"Approval","inputs":[{"name":"owner","type":"address","indexed":true,"internalType":"address"},{"name":"spender","type":"address","indexed":true,"internalType":"address"},{"name":"value","type":"uint256","indexed":false,"internalType":"uint256"}],"anonymous":false},{"type":"event","name":"OwnershipTransferred","inputs":[{"name":"previousOwner","type":"address","indexed":true,"internalType":"address"},{"name":"newOwner","type":"address","indexed":true,"internalType":"address"}],"anonymous":false},{"type":"event","name":"Transfer","inputs":[{"name":"from","type":"address","indexed":true,"internalType":"address"},{"name":"to","type":"address","indexed":true,"internalType":"address"},{"name":"value","type":"uint256","indexed":false,"internalType":"uint256"}],"anonymous":false},{"type":"error","name":"ERC20InsufficientAllowance","inputs":[{"name":"spender","type":"address","internalType":"address"},{"name":"allowance","type":"uint256","internalType":"uint256"},{"name":"needed","type":"uint256","internalType":"uint256"}]},{"type":"error","name":"ERC20InsufficientBalance","inputs":[{"name":"sender","type":"address","internalType":"address"},{"name":"balance","type":"uint256","internalType":"uint256"},{"name":"needed","type":"uint256","internalType":"uint256"}]},{"type":"error","name":"ERC20InvalidApprover","inputs":[{"name":"approver","type":"address","internalType":"address"}]},{"type":"error","name":"ERC20InvalidReceiver","inputs":[{"name":"receiver","type":"address","internalType":"address"}]},{"type":"error","name":"ERC20InvalidSender","inputs":[{"name":"sender","type":"address","internalType":"address"}]},{"type":"error","name":"ERC20InvalidSpender","inputs":[{"name":"spender","type":"address","internalType":"address"}]},{"type":"error","name":"OwnableInvalidOwner","inputs":[{"name":"owner","type":"address","internalType":"address"}]},{"type":"error","name":"OwnableUnauthorizedAccount","inputs":[{"name":"account","type":"address","internalType":"address"}]}]`

type TachiToken struct {
	address common.Address
	abi     abi.ABI
	client  *ethclient.Client
	mu      sync.Mutex
}

func NewTachiToken(address common.Address, client *ethclient.Client) (*TachiToken, error) {
	parsedABI, err := abi.JSON(strings.NewReader(tachiTokenABI))
	if err != nil {
		return nil, fmt.Errorf("parse TachiToken ABI: %w", err)
	}

	return &TachiToken{
		address: address,
		abi:     parsedABI,
		client:  client,
	}, nil
}

func (t *TachiToken) Mint(ctx context.Context, toAddr common.Address, amount *big.Int, signerKey *ecdsa.PrivateKey) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client == nil {
		return "", fmt.Errorf("eth client is nil")
	}
	if signerKey == nil {
		return "", fmt.Errorf("signer key is nil")
	}
	if amount == nil || amount.Sign() <= 0 {
		return "", fmt.Errorf("amount must be greater than zero")
	}

	fromAddr := crypto.PubkeyToAddress(signerKey.PublicKey)
	data, err := t.abi.Pack("mint", toAddr, amount)
	if err != nil {
		return "", fmt.Errorf("pack mint calldata: %w", err)
	}

	chainID, err := t.client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("get chain ID: %w", err)
	}

	nonce, err := t.client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return "", fmt.Errorf("get pending nonce: %w", err)
	}

	tipCap, err := t.client.SuggestGasTipCap(ctx)
	if err != nil {
		return "", fmt.Errorf("suggest gas tip cap: %w", err)
	}

	header, err := t.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("get latest header: %w", err)
	}
	if header.BaseFee == nil {
		return "", fmt.Errorf("latest header missing base fee")
	}

	feeCap := new(big.Int).Mul(header.BaseFee, big.NewInt(2))
	feeCap.Add(feeCap, tipCap)

	callMsg := ethereum.CallMsg{
		From:      fromAddr,
		To:        &t.address,
		GasFeeCap: feeCap,
		GasTipCap: tipCap,
		Data:      data,
	}
	gasLimit, err := t.client.EstimateGas(ctx, callMsg)
	if err != nil {
		return "", fmt.Errorf("estimate gas: %w", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
		Gas:       gasLimit,
		To:        &t.address,
		Data:      data,
	})

	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), signerKey)
	if err != nil {
		return "", fmt.Errorf("sign mint tx: %w", err)
	}

	if err := t.client.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("send mint tx: %w", err)
	}
	receipt, err := bind.WaitMined(ctx, t.client, signedTx)
	if err != nil {
		return "", fmt.Errorf("wait mint receipt: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return "", fmt.Errorf("mint tx failed: %s", signedTx.Hash().Hex())
	}

	return signedTx.Hash().Hex(), nil
}

func (t *TachiToken) Burn(ctx context.Context, fromAddr common.Address, amount *big.Int, signerKey *ecdsa.PrivateKey) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client == nil {
		return "", fmt.Errorf("eth client is nil")
	}
	if signerKey == nil {
		return "", fmt.Errorf("signer key is nil")
	}
	if amount == nil || amount.Sign() <= 0 {
		return "", fmt.Errorf("amount must be greater than zero")
	}

	fromSignerAddr := crypto.PubkeyToAddress(signerKey.PublicKey)
	data, err := t.abi.Pack("burn", fromAddr, amount)
	if err != nil {
		return "", fmt.Errorf("pack burn calldata: %w", err)
	}

	chainID, err := t.client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("get chain ID: %w", err)
	}

	nonce, err := t.client.PendingNonceAt(ctx, fromSignerAddr)
	if err != nil {
		return "", fmt.Errorf("get pending nonce: %w", err)
	}

	tipCap, err := t.client.SuggestGasTipCap(ctx)
	if err != nil {
		return "", fmt.Errorf("suggest gas tip cap: %w", err)
	}

	header, err := t.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("get latest header: %w", err)
	}
	if header.BaseFee == nil {
		return "", fmt.Errorf("latest header missing base fee")
	}

	feeCap := new(big.Int).Mul(header.BaseFee, big.NewInt(2))
	feeCap.Add(feeCap, tipCap)

	callMsg := ethereum.CallMsg{
		From:      fromSignerAddr,
		To:        &t.address,
		GasFeeCap: feeCap,
		GasTipCap: tipCap,
		Data:      data,
	}
	gasLimit, err := t.client.EstimateGas(ctx, callMsg)
	if err != nil {
		return "", fmt.Errorf("estimate gas: %w", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
		Gas:       gasLimit,
		To:        &t.address,
		Data:      data,
	})

	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), signerKey)
	if err != nil {
		return "", fmt.Errorf("sign burn tx: %w", err)
	}

	if err := t.client.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("send burn tx: %w", err)
	}
	receipt, err := bind.WaitMined(ctx, t.client, signedTx)
	if err != nil {
		return "", fmt.Errorf("wait burn receipt: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return "", fmt.Errorf("burn tx failed: %s", signedTx.Hash().Hex())
	}

	return signedTx.Hash().Hex(), nil
}
