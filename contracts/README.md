## Foundry

**Foundry is a blazing fast, portable and modular toolkit for Ethereum application development written in Rust.**

Foundry consists of:

- **Forge**: Ethereum testing framework (like Truffle, Hardhat and DappTools).
- **Cast**: Swiss army knife for interacting with EVM smart contracts, sending transactions and getting chain data.
- **Anvil**: Local Ethereum node, akin to Ganache, Hardhat Network.
- **Chisel**: Fast, utilitarian, and verbose solidity REPL.

## Documentation

https://book.getfoundry.sh/

## Setup

初次 clone 後需安裝 Solidity 依賴：

```shell
cd contracts
forge install OpenZeppelin/openzeppelin-contracts@v5.6.1 --no-git
```

接著即可執行 build 與測試：

```shell
forge build
forge test
```

為避免在 monorepo 根目錄產生 nested submodule / gitlink 汙染，可在安裝後額外確認：

```shell
test ! -f ../.gitmodules
git ls-files --stage .. | grep 160000
```

上述檢查都不應輸出任何內容；`contracts/lib/` 也已列入 repo 的 `.gitignore`，因此依 README 操作不會把相依套件納入版控。

## Usage

### Build

```shell
forge build
```

### Test

```shell
forge test
```

### Format

```shell
forge fmt
```

### Gas Snapshots

```shell
forge snapshot
```

### Anvil

```shell
anvil
```

### Deploy

```bash
# 複製環境變數範本
cp .env.example .env
# 填入 DEPLOYER_PRIVATE_KEY、SEPOLIA_RPC_URL、ETHERSCAN_API_KEY
source .env

# 部署到 Sepolia（含 Etherscan 驗證）
forge script script/Deploy.s.sol:Deploy \
  --rpc-url $SEPOLIA_RPC_URL \
  --broadcast \
  --verify \
  --etherscan-api-key $ETHERSCAN_API_KEY

# 若部署時未加 --verify，事後單獨驗證
forge verify-contract <CONTRACT_ADDRESS> src/TachiToken.sol:TachiToken \
  --chain sepolia \
  --etherscan-api-key $ETHERSCAN_API_KEY
```

### Cast

```shell
cast <subcommand>
```

### Help

```shell
forge --help
anvil --help
cast --help
```
