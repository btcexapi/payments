/* Mysterium network payment library.
 *
 * Copyright (C) 2020 BlockDev AG
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 * You should have received a copy of the GNU Lesser General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package client

import (
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/pkg/errors"
)

type Estimatable interface {
	getGasLimit() uint64
	toEstimator(ethClientGetter) (*bindings.ContractEstimator, error)
	toEstimateOps() *bindings.EstimateOpts
}

// ErrorTransactionReverted when contract execution is interrupted with an error./
type ErrorTransactionReverted struct {
	Err    rpc.Error
	Reason string
}

func (e ErrorTransactionReverted) Error() string {
	return e.Err.Error()
}

// WithDryRuns forces a dry run before running a write transaction on blockchain.
// Ethereum client will perform a dry run on a transaction with no gas limit set.
// This component will perform a dry run if and only if the gas limit is set to a non zero value.
// In this way, the dry run is always performed before sending the transaction to the network.
// For convenience, this component proxies read only calls to the underlying blockchain.
type WithDryRuns struct {
	bc        BC
	ethClient ethClientGetter
}

// NewWithDryRuns creates a new instance of client with dry runs.
func NewWithDryRuns(bc BC, ethClient ethClientGetter) *WithDryRuns {
	return &WithDryRuns{
		bc:        bc,
		ethClient: ethClient,
	}
}

type gasLimitProvider interface {
	GetGasLimit() uint64
}

// GetEthBalance gets the current ethereum balance for the address.
func (cwdr *WithDryRuns) GetEthBalance(address common.Address) (*big.Int, error) {
	return cwdr.bc.GetEthBalance(address)
}

func (cwdr *WithDryRuns) GetHermessAvailableBalance(hermesAddress common.Address) (*big.Int, error) {
	return cwdr.bc.GetHermessAvailableBalance(hermesAddress)
}

func (cwdr *WithDryRuns) TransferEth(etr EthTransferRequest) (*types.Transaction, error) {
	// TODO: implement this dry run
	return cwdr.bc.TransferEth(etr)
}

func (cwdr *WithDryRuns) Estimate(req Estimatable) (uint64, error) {
	// If the gas limit is set to 0, ethereum client will do the estimation for us.
	// We only force the estimation if the gas limit is set to a non zero value.
	if req.getGasLimit() == 0 {
		return 0, nil
	}

	estimator, err := req.toEstimator(cwdr.ethClient)
	if err != nil {
		return 0, err
	}

	gas, err := estimator.Estimate(req.toEstimateOps())
	return gas, errors.Wrap(err, "could not estimate gas")
}

// DryRun simulates the (paid) contract method with params as input values.
func (cwdr *WithDryRuns) DryRun(req Estimatable) error {
	_, err := cwdr.Estimate(req)
	if err == nil {
		return nil
	}

	// Extract error thrown in contract
	if rpcCauseErr, hasCause := errors.Cause(err).(rpc.Error); hasCause {
		if rpcCauseErr.ErrorCode() == -32000 && strings.Contains(rpcCauseErr.Error(), "VM Exception while processing transaction: revert") {
			err = &ErrorTransactionReverted{
				Err:    rpcCauseErr,
				Reason: strings.TrimPrefix(rpcCauseErr.Error(), "VM Exception while processing transaction: revert "),
			}
		}
	}

	return err
}

// TransferMyst transfers myst
func (cwdr *WithDryRuns) TransferMyst(req TransferRequest) (tx *types.Transaction, err error) {
	if _, err := cwdr.Estimate(req); err != nil {
		return nil, err
	}

	return cwdr.bc.TransferMyst(req)
}

// SettlePromise is settling the given consumer issued promise
func (cwdr *WithDryRuns) SettlePromise(req SettleRequest) (*types.Transaction, error) {
	if _, err := cwdr.Estimate(req); err != nil {
		return nil, err
	}

	return cwdr.bc.SettlePromise(req)
}

// RegisterIdentity registers the given identity on blockchain
func (cwdr *WithDryRuns) RegisterIdentity(req RegistrationRequest) (*types.Transaction, error) {
	if _, err := cwdr.Estimate(req); err != nil {
		return nil, err
	}

	return cwdr.bc.RegisterIdentity(req)
}

// SettleAndRebalance is settling given hermes issued promise
func (cwdr *WithDryRuns) SettleAndRebalance(req SettleAndRebalanceRequest) (*types.Transaction, error) {
	if _, err := cwdr.Estimate(req); err != nil {
		return nil, err
	}

	return cwdr.bc.SettleAndRebalance(req)
}

// GetHermesFee fetches the hermes fee from blockchain
func (cwdr *WithDryRuns) GetHermesFee(hermesAddress common.Address) (uint16, error) {
	return cwdr.bc.GetHermesFee(hermesAddress)
}

// GetHermesURL returns the thermes URL.
func (cwdr *WithDryRuns) GetHermesURL(registryID, hermesID common.Address) (string, error) {
	return cwdr.bc.GetHermesURL(registryID, hermesID)
}

// CalculateHermesFee fetches the hermes fee from blockchain
func (cwdr *WithDryRuns) CalculateHermesFee(hermesAddress common.Address, value *big.Int) (*big.Int, error) {
	return cwdr.bc.CalculateHermesFee(hermesAddress, value)
}

// IsRegisteredAsProvider checks if the provider is registered with the hermes properly
func (cwdr *WithDryRuns) IsRegisteredAsProvider(hermesAddress, registryAddress, addressToCheck common.Address) (bool, error) {
	return cwdr.bc.IsRegisteredAsProvider(hermesAddress, registryAddress, addressToCheck)
}

// GetProviderChannel returns the provider channel
func (cwdr *WithDryRuns) GetProviderChannel(hermesAddress common.Address, addressToCheck common.Address, pending bool) (ProviderChannel, error) {
	return cwdr.bc.GetProviderChannel(hermesAddress, addressToCheck, pending)
}

// IsRegistered checks wether the given identity is registered or not
func (cwdr *WithDryRuns) IsRegistered(registryAddress, addressToCheck common.Address) (bool, error) {
	return cwdr.bc.IsRegistered(registryAddress, addressToCheck)
}

// SubscribeToPromiseSettledEvent subscribes to promise settled events
func (cwdr *WithDryRuns) SubscribeToPromiseSettledEvent(providerID, hermesID common.Address) (sink chan *bindings.HermesImplementationPromiseSettled, cancel func(), err error) {
	return cwdr.bc.SubscribeToPromiseSettledEvent(providerID, hermesID)
}

// GetMystBalance returns the balance in myst
func (cwdr *WithDryRuns) GetMystBalance(mystSCAddress, address common.Address) (*big.Int, error) {
	return cwdr.bc.GetMystBalance(mystSCAddress, address)
}

// SubscribeToConsumerBalanceEvent subscribes to the consumer balance change events
func (cwdr *WithDryRuns) SubscribeToConsumerBalanceEvent(channel, mystSCAddress common.Address, timeout time.Duration) (chan *bindings.MystTokenTransfer, func(), error) {
	return cwdr.bc.SubscribeToConsumerBalanceEvent(channel, mystSCAddress, timeout)
}

// IsHermesRegistered checks if given hermes is registered and returns true or false.
func (cwdr *WithDryRuns) IsHermesRegistered(registryAddress, acccountantID common.Address) (bool, error) {
	return cwdr.bc.IsHermesRegistered(registryAddress, acccountantID)
}

// GetHermesOperator returns operator address of given hermes
func (cwdr *WithDryRuns) GetHermesOperator(hermesID common.Address) (common.Address, error) {
	return cwdr.bc.GetHermesOperator(hermesID)
}

// GetConsumerChannelsHermes returns the consumer channels hermes
func (cwdr *WithDryRuns) GetConsumerChannelsHermes(channelAddress common.Address) (ConsumersHermes, error) {
	return cwdr.bc.GetConsumerChannelsHermes(channelAddress)
}

// SubscribeToMystTokenTransfers subscribes to myst token transfers
func (cwdr *WithDryRuns) SubscribeToMystTokenTransfers(mystSCAddress common.Address) (chan *bindings.MystTokenTransfer, func(), error) {
	return cwdr.bc.SubscribeToMystTokenTransfers(mystSCAddress)
}

// GetConsumerChannelOperator returns the consumer channel operator/identity
func (cwdr *WithDryRuns) GetConsumerChannelOperator(channelAddress common.Address) (common.Address, error) {
	return cwdr.bc.GetConsumerChannelOperator(channelAddress)
}

// GetConsumerChannel returns the consumer channel
func (cwdr *WithDryRuns) GetConsumerChannel(addr common.Address, mystSCAddress common.Address) (ConsumerChannel, error) {
	return cwdr.bc.GetConsumerChannel(addr, mystSCAddress)
}

// GetProviderChannelByID returns the given channel information
func (cwdr *WithDryRuns) GetProviderChannelByID(acc common.Address, chID []byte) (ProviderChannel, error) {
	return cwdr.bc.GetProviderChannelByID(acc, chID)
}

// SubscribeToIdentityRegistrationEvents subscribes to identity registration events
func (cwdr *WithDryRuns) SubscribeToIdentityRegistrationEvents(registryAddress common.Address) (sink chan *bindings.RegistryRegisteredIdentity, cancel func(), err error) {
	return cwdr.bc.SubscribeToIdentityRegistrationEvents(registryAddress)
}

// SubscribeToConsumerChannelBalanceUpdate subscribes to consumer channel balance update events
func (cwdr *WithDryRuns) SubscribeToConsumerChannelBalanceUpdate(mystSCAddress common.Address, channelAddresses []common.Address) (sink chan *bindings.MystTokenTransfer, cancel func(), err error) {
	return cwdr.bc.SubscribeToConsumerChannelBalanceUpdate(mystSCAddress, channelAddresses)
}

// SubscribeToPromiseSettledEventByChannelID subscribes to promise settled events
func (cwdr *WithDryRuns) SubscribeToPromiseSettledEventByChannelID(hermesID common.Address, providerAddresses [][32]byte) (sink chan *bindings.HermesImplementationPromiseSettled, cancel func(), err error) {
	return cwdr.bc.SubscribeToPromiseSettledEventByChannelID(hermesID, providerAddresses)
}

// NetworkID returns the network id
func (cwdr *WithDryRuns) NetworkID() (*big.Int, error) {
	return cwdr.bc.NetworkID()
}

// GetStakeThresholds returns the stake tresholds for the given hermes.
func (cwdr *WithDryRuns) GetStakeThresholds(hermesID common.Address) (min, max *big.Int, err error) {
	return cwdr.bc.GetStakeThresholds(hermesID)
}

// SettleWithBeneficiary sets new beneficiary and settling given hermes issued promise into it.
func (cwdr *WithDryRuns) SettleWithBeneficiary(req SettleWithBeneficiaryRequest) (*types.Transaction, error) {
	if _, err := cwdr.Estimate(req); err != nil {
		return nil, err
	}

	return cwdr.bc.SettleWithBeneficiary(req)
}

// DecreaseProviderStake decreases provider stake.
func (cwdr *WithDryRuns) DecreaseProviderStake(req DecreaseProviderStakeRequest) (*types.Transaction, error) {
	if _, err := cwdr.Estimate(req); err != nil {
		return nil, err
	}

	return cwdr.bc.DecreaseProviderStake(req)
}

func (cwdr *WithDryRuns) GetBeneficiary(registryAddress, identity common.Address) (common.Address, error) {
	return cwdr.bc.GetBeneficiary(registryAddress, identity)
}

func (cwdr *WithDryRuns) SuggestGasPrice() (*big.Int, error) {
	return cwdr.bc.SuggestGasPrice()
}

// SettleIntoStake settles the hermes promise into stake increase.
func (cwdr *WithDryRuns) SettleIntoStake(req SettleIntoStakeRequest) (*types.Transaction, error) {
	if _, err := cwdr.Estimate(req); err != nil {
		return nil, err
	}

	return cwdr.bc.SettleIntoStake(req)
}

// IncreaseProviderStake increases the provider stake.
func (cwdr *WithDryRuns) IncreaseProviderStake(req ProviderStakeIncreaseRequest) (*types.Transaction, error) {
	if _, err := cwdr.Estimate(req); err != nil {
		return nil, err
	}

	return cwdr.bc.IncreaseProviderStake(req)
}

func (cwdr *WithDryRuns) TransactionReceipt(hash common.Hash) (*types.Receipt, error) {
	return cwdr.bc.TransactionReceipt(hash)
}
