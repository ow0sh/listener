package crypto

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"golang.org/x/crypto/sha3"
)

func IsTransfer(data []byte) bool {
	methodId := []byte("transfer(address,uint256)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(methodId)
	if hexutil.Encode(hash.Sum(nil)[:4]) == hexutil.Encode(data[:4]) {
		return true
	}
	return false
}

func IsTransferFrom(data []byte) bool {
	methodId := []byte("transferFrom(address,address,uint256)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(methodId)
	if hexutil.Encode(hash.Sum(nil)[:4]) == hexutil.Encode(data[:4]) {
		return true
	}
	return false
}
