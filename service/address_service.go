package service

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip39"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ==========================
// 数据库模型
// ==========================
type HDWallet struct {
	ID              uint   `gorm:"primaryKey"`
	Mnemonic        string `gorm:"type:text"`
	SeedFingerprint string `gorm:"size:64"`
	CoinType        int
	Addresses       []Address `gorm:"foreignKey:WalletID"`
}

type Address struct {
	ID             uint   `gorm:"primaryKey"`
	WalletID       uint   `gorm:"index"`
	DerivationPath string `gorm:"size:255"`
	Address        string `gorm:"uniqueIndex;size:64"`
	Used           bool   `gorm:"default:false"`
	UserID         *uint
}

// ==========================
// 主逻辑 - 简化版本
// ==========================
func main() {
	// 连接数据库
	dsn := "host=localhost port=5432 user=postgres password=123456 dbname=wallet sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 自动迁移表
	if err := db.AutoMigrate(&HDWallet{}, &Address{}); err != nil {
		log.Fatal("自动迁移失败:", err)
	}

	// === Step 1: 生成助记词 & 种子 ===
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		log.Fatal("生成熵失败:", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		log.Fatal("生成助记词失败:", err)
	}
	seed := bip39.NewSeed(mnemonic, "")

	fmt.Println("助记词:", mnemonic)

	// === Step 2: 存 HDWallet ===
	seedHash := crypto.Keccak256(seed)
	hd := HDWallet{
		Mnemonic:        mnemonic,
		SeedFingerprint: hex.EncodeToString(seedHash[:8]),
		CoinType:        60,
	}
	result := db.Create(&hd)
	if result.Error != nil {
		log.Fatal("创建钱包失败:", result.Error)
	}

	// === Step 3: 生成主私钥 ===
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		log.Fatal("生成主密钥失败:", err)
	}

	// === Step 4: 根据 BIP44 路径派生密钥 ===
	deriveKey := func(parent *hdkeychain.ExtendedKey, index uint32) *hdkeychain.ExtendedKey {
		child, err := parent.Derive(index)
		if err != nil {
			log.Fatal("派生失败:", err)
		}
		return child
	}

	purpose := deriveKey(masterKey, hdkeychain.HardenedKeyStart+44)
	coinType := deriveKey(purpose, hdkeychain.HardenedKeyStart+60)
	account := deriveKey(coinType, hdkeychain.HardenedKeyStart+0)
	change := deriveKey(account, 0)

	// === Step 5: 批量生成地址池 ===
	numAddresses := 10
	var addresses []Address

	for i := 0; i < numAddresses; i++ {
		addressIndex, err := change.Derive(uint32(i))
		if err != nil {
			log.Fatal("派生 addressIndex 失败:", err)
		}

		// 1. 获取公钥对象
		pubKey, err := addressIndex.ECPubKey()
		if err != nil {
			log.Fatal("获取公钥失败:", err)
		}

		// 2. 转换为压缩公钥字节
		publicKeyBytes := pubKey.SerializeCompressed()

		// 3. 转换为 ECDSA 公钥 (以太坊用)
		publicKeyECDSA, err := crypto.DecompressPubkey(publicKeyBytes)
		if err != nil {
			log.Fatal("解压缩公钥失败:", err)
		}

		// 4. 生成以太坊地址
		ethAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

		derivationPath := fmt.Sprintf("m/44'/60'/0'/0/%d", i)

		addresses = append(addresses, Address{
			WalletID:       hd.ID,
			DerivationPath: derivationPath,
			Address:        ethAddress.Hex(),
			Used:           false,
		})

		fmt.Printf("[%d] 地址: %s, 路径: %s\n", i, ethAddress.Hex(), derivationPath)
		fmt.Printf("     公钥: %x\n", publicKeyBytes)
	}

	// === Step 6: 批量插入地址 ===
	result = db.Create(&addresses)
	if result.Error != nil {
		log.Fatal("创建地址失败:", result.Error)
	}

	fmt.Printf("成功生成 %d 个地址并写入数据库！\n", numAddresses)
}
