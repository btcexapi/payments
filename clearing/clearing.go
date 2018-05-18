package clearing

import (
	"github.com/MysteriumNetwork/payments/clearing/generated"
	"math/big"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/common"
	"github.com/MysteriumNetwork/payments/registry"
)

//go:generate ../scripts/abigen.sh --sol ../contracts/clearingContract.sol --exc contract/registry.sol:IdentityRegistry --pkg generated --out generated/clearing.go

type PromiseClearer struct {
	generated.ClearingContractSession
}

type Promise struct {
	seqNo int
	amount int
	receiverSign []byte
	payerSign []byte
}

func DeployPromiseClearer(owner * bind.TransactOpts , erc20Token common.Address , fee int64, backend bind.ContractBackend) (*PromiseClearer , error) {
	_ , _ , clearingContract , err := generated.DeployClearingContract(owner , backend, erc20Token, big.NewInt(fee))
	if err != nil {
		return nil , err
	}

	return NewPromiseClearer(owner , clearingContract), nil
}


func NewPromiseClearer(transactOpts * bind.TransactOpts, contract * generated.ClearingContract) *PromiseClearer {
	return &PromiseClearer{
		generated.ClearingContractSession{
			Contract: contract,
			CallOpts: bind.CallOpts{},
			TransactOpts: *transactOpts,
		},
	}
}

func (pc * PromiseClearer) RegisterIdentities(identities ...registry.MystIdentity) error {
	for _ , identity := range identities {
		proof, err := registry.CreateProofOfIdentity(&identity)
		if err != nil {
			return err
		}
		_ , err = pc.RegisterIdentity(proof.RandomNumber, proof.Signature)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pc * PromiseClearer) ClearMyPromise(promise Promise) error {
	seqNo :=big.NewInt(int64(promise.seqNo))
	amount :=big.NewInt(int64(promise.amount))
	_ , err := pc.ClearPromise( seqNo, amount, promise.receiverSign, promise.payerSign)
	return err
}

func (pc * PromiseClearer) BindForEvents(eventChan chan<- *generated.ClearingContractPromiseCleared) (event.Subscription , error) {
	start := uint64(0)
	return pc.Contract.WatchPromiseCleared(&bind.WatchOpts{Start:&start} , eventChan, []common.Address{}, []common.Address{})
}