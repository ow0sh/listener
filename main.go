package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	token "github.com/ow0sh/listener/contract"
	"github.com/ow0sh/listener/crypto"
	"github.com/sirupsen/logrus"
	"math/big"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	log := logrus.New()
	// Creating context, which will be done after syscall.SIGTERM || syscall.SIGINT
	ctx, cancel := ctxWithSig()
	defer func() {
		// Regain control of panicking goroutine, if goroutine is panicking, recover will
		// capture the value given to panic, logging it and cancelling the context
		// Maybe not necessary in this code, but I used to use it
		if err := recover(); err != nil {
			log.Error(err)
			cancel()
		}
	}()

	client, err := ethclient.Dial("http://65.108.9.125:8543/")
	if err != nil {
		panic(err)
	}

	// Getting block number from input
	fmt.Println("Enter block number:")
	var blockNumStr string
	_, err = fmt.Scanln(&blockNumStr)
	if err != nil {
		panic(err)
	}

	// Converting it to *big.Int
	blockNum, ok := big.NewInt(0).SetString(blockNumStr, 10)
	if !ok {
		panic(err)
	}

	// Getting block from ETH blockchain
	block, err := client.BlockByNumber(ctx, blockNum)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	// Getting transactions from block, and iterate through each of them
	txs := block.Transactions()
	for i, tx := range txs {
		// Creating goroutine for each transaction
		wg.Add(1)
		go func(i int, tx *types.Transaction) {
			defer wg.Done()
			// Getting sender of transaction
			from, err := client.TransactionSender(ctx, tx, block.Hash(), uint(i))
			if err != nil {
				log.Error(err)
				return
			}

			// Checking if it's simple transfer or interaction with smart contract
			if len(tx.Data()) != 0 {
				// transfer(address,uint256) == MethodID
				if crypto.IsTransfer(tx.Data()) {
					// Getting recipient of transaction
					to := common.HexToAddress(hexutil.Encode(tx.Data()[4:36]))

					// Getting instance of ERC20 contract
					instance, err := token.UnwrapContract(*tx.To(), client)
					if err != nil {
						log.Error(err)
						return
					}
					decimals, err := instance.Decimals()
					if err != nil {
						decimals = 18
					}
					symbol, err := instance.Symbol()
					if err != nil {
						log.Error(err)
						return
					}

					// Dividing amount of transaction by token's decimals
					amount := divideBigIntByDecimals(big.NewInt(0).SetBytes(tx.Data()[36:]), decimals)

					log.Info(fmt.Sprintf("{%v}-{%v}-{%v}-{%v}-{%v}", from, to, amount, symbol, tx.To()))
					return
				}

				//// transferFrom(address,address,uint256) == MethodID
				if crypto.IsTransferFrom(tx.Data()) {
					// Getting sender and recipient of transaction from data
					from := common.HexToAddress(hexutil.Encode(tx.Data()[4:36]))
					to := common.HexToAddress(hexutil.Encode(tx.Data()[36:68]))

					// Getting instance of ERC20 contract
					instance, err := token.UnwrapContract(*tx.To(), client)
					if err != nil {
						log.Error(err)
						return
					}
					decimals, err := instance.Decimals()
					if err != nil {
						decimals = 18
					}
					symbol, err := instance.Symbol()
					if err != nil {
						log.Error(err)
						return
					}

					// Dividing amount of transaction by token's decimals
					amount := divideBigIntByDecimals(big.NewInt(0).SetBytes(tx.Data()[68:]), decimals)

					log.Info(fmt.Sprintf("{%v}-{%v}-{%v}-{%v}-{%v}", from, to, amount, symbol, tx.To()))
					return
				}
				return
			}
			to := tx.To()

			// Dividing amount of transaction by token's decimals
			amount := divideBigIntByDecimals(tx.Value(), 18)

			log.Info(fmt.Sprintf("{%v}-{%v}-{%v}-{ETH}-{0x000000000000000000000000000000000000000}", from, to, amount))
		}(i, tx)
	}
	wg.Wait()
}

func ctxWithSig() (context.Context, func()) {
	// Creating context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	// Creating chan with os.Signal type
	ch := make(chan os.Signal, 1)
	// Notify the channel, when syscall.SIGINT || syscall.SIGTERM is called
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	// Listen to channel and cancel the context when channel is notified
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
