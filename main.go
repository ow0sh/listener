package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	token "github.com/ow0sh/listener/contract"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"
	"math"
	"math/big"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	log := logrus.New()
	ctx, cancel := ctxWithSig()
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
			cancel()
		}
	}()

	client, err := ethclient.Dial("https://mainnet.infura.io/v3/4740b9f6ce7f4a2f97f9158e5658151e")
	if err != nil {
		panic(err)
	}

	fmt.Println("Enter block number:")
	var blockNumStr string
	_, err = fmt.Scanln(&blockNumStr)
	if err != nil {
		panic(err)
	}

	blockNum, ok := big.NewInt(0).SetString(blockNumStr, 10)
	if !ok {
		panic(err)
	}

	block, err := client.BlockByNumber(ctx, blockNum)
	if err != nil {
		panic(err)
	}

	transfer := []byte("transfer(address,uint256)")
	transferHash := sha3.NewLegacyKeccak256()
	transferHash.Write(transfer)

	transferFrom := []byte("transferFrom(address,address,uint256)")
	transferFromHash := sha3.NewLegacyKeccak256()
	transferFromHash.Write(transferFrom)

	var wg sync.WaitGroup
	txs := block.Transactions()
	for i, tx := range txs {
		wg.Add(1)
		go func(i int, tx *types.Transaction) {
			defer wg.Done()
			from, err := client.TransactionSender(ctx, tx, block.Hash(), uint(i))
			if err != nil {
				log.Error(err)
				return
			}
			if len(tx.Data()) != 0 {
				if hexutil.Encode(transferHash.Sum(nil)[:4]) == hexutil.Encode(tx.Data()[:4]) {
					to := common.HexToAddress(hexutil.Encode(tx.Data()[4:36]))

					instance, err := token.NewToken(*tx.To(), client)
					if err != nil {
						log.Error(errors.Wrap(err, "failed to create instance"))
						return
					}

					symbol, err := instance.Symbol(nil)
					if err != nil {
						log.Error(errors.Wrap(err, "failed to get symbol"))
						return
					}

					decimals, err := instance.Decimals(nil)
					if err != nil {
						decimals = 18
					}

					num1 := big.NewInt(0).SetBytes(tx.Data()[36:])
					num2 := big.NewInt(int64(math.Pow(10, float64(decimals))))
					aflt1 := new(big.Float).SetInt(num1)
					aflt2 := new(big.Float).SetInt(num2)
					amount := new(big.Float).Quo(aflt1, aflt2)

					log.Info(fmt.Sprintf("{%v}-{%v}-{%v}-{%v}-{%v}", from, to, amount, symbol, tx.To()))
					return
				}
				if hexutil.Encode(transferFromHash.Sum(nil)[:4]) == hexutil.Encode(tx.Data()[:4]) {
					from := common.HexToAddress(hexutil.Encode(tx.Data()[4:36]))
					to := common.HexToAddress(hexutil.Encode(tx.Data()[36:68]))

					instance, err := token.NewToken(*tx.To(), client)
					if err != nil {
						log.Error(errors.Wrap(err, "failed to get token"))
						return
					}

					symbol, err := instance.Symbol(nil)
					if err != nil {
						log.Error(errors.Wrap(err, "failed to get symbol"))
						return
					}

					decimals, err := instance.Decimals(nil)
					if err != nil {
						decimals = 18
					}

					num1 := big.NewInt(0).SetBytes(tx.Data()[68:])
					num2 := big.NewInt(int64(math.Pow(10, float64(decimals))))
					aflt1 := new(big.Float).SetInt(num1)
					aflt2 := new(big.Float).SetInt(num2)
					amount := new(big.Float).Quo(aflt1, aflt2)

					log.Info(fmt.Sprintf("{%v}-{%v}-{%v}-{%v}-{%v}", from, to, amount, symbol, tx.To()))
					return
				}
				return
			}
			to := tx.To()

			num1 := tx.Value()
			num2 := big.NewInt(int64(math.Pow(10, 18)))
			aflt1 := new(big.Float).SetInt(num1)
			aflt2 := new(big.Float).SetInt(num2)
			amount := new(big.Float).Quo(aflt1, aflt2)

			log.Info(fmt.Sprintf("{%v}-{%v}-{%v}-{ETH}-{0x000000000000000000000000000000000000000}", from, to, amount))
		}(i, tx)
	}
	wg.Wait()
}

func ctxWithSig() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-ch:
				cancel()
			}
		}
	}()

	return ctx, cancel
}
