// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

// mockStateDB - мок для интерфейса StateDB для тестирования
type mockStateDB struct {
	state.StateDB
	accounts map[common.Address]*mockAccount
	logs     []*types.Log
}

type mockAccount struct {
	balance *uint256.Int
	nonce   uint64
	code    []byte
	storage map[common.Hash]common.Hash
}

func newMockAccount() *mockAccount {
	return &mockAccount{
		balance: uint256.NewInt(0),
		nonce:   0,
		code:    nil,
		storage: make(map[common.Hash]common.Hash),
	}
}

func newMockStateDB() *mockStateDB {
	return &mockStateDB{
		accounts: make(map[common.Address]*mockAccount),
		logs:     []*types.Log{},
	}
}

// Реализация методов интерфейса StateDB для тестирования
func (s *mockStateDB) getAccount(addr common.Address) *mockAccount {
	if account, exists := s.accounts[addr]; exists {
		return account
	}
	s.accounts[addr] = newMockAccount()
	return s.accounts[addr]
}

func (s *mockStateDB) CreateAccount(addr common.Address) {
	s.getAccount(addr)
}

func (s *mockStateDB) CreateContract(addr common.Address) {
	s.getAccount(addr)
}

func (s *mockStateDB) SubBalance(addr common.Address, amount *uint256.Int, _ tracing.BalanceChangeReason) uint256.Int {
	account := s.getAccount(addr)
	prevBalance := *account.balance
	account.balance.Sub(account.balance, amount)
	return prevBalance
}

func (s *mockStateDB) AddBalance(addr common.Address, amount *uint256.Int, _ tracing.BalanceChangeReason) uint256.Int {
	account := s.getAccount(addr)
	prevBalance := *account.balance
	account.balance.Add(account.balance, amount)
	return prevBalance
}

func (s *mockStateDB) GetBalance(addr common.Address) *uint256.Int {
	return s.getAccount(addr).balance
}

func (s *mockStateDB) GetNonce(addr common.Address) uint64 {
	return s.getAccount(addr).nonce
}

func (s *mockStateDB) SetNonce(addr common.Address, nonce uint64, _ tracing.NonceChangeReason) {
	s.getAccount(addr).nonce = nonce
}

func (s *mockStateDB) GetCodeHash(addr common.Address) common.Hash {
	code := s.getAccount(addr).code
	if code == nil {
		return common.Hash{}
	}
	return crypto.Keccak256Hash(code)
}

func (s *mockStateDB) GetCode(addr common.Address) []byte {
	return s.getAccount(addr).code
}

func (s *mockStateDB) SetCode(addr common.Address, code []byte) []byte {
	account := s.getAccount(addr)
	prevCode := account.code
	account.code = code
	return prevCode
}

func (s *mockStateDB) GetCodeSize(addr common.Address) int {
	code := s.getAccount(addr).code
	if code == nil {
		return 0
	}
	return len(code)
}

func (s *mockStateDB) AddRefund(gas uint64) {}
func (s *mockStateDB) SubRefund(gas uint64) {}
func (s *mockStateDB) GetRefund() uint64    { return 0 }

func (s *mockStateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	return s.getAccount(addr).storage[hash]
}

func (s *mockStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) common.Hash {
	account := s.getAccount(addr)
	prev := account.storage[key]
	account.storage[key] = value
	return prev
}

func (s *mockStateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	return s.getAccount(addr).storage[hash]
}

func (s *mockStateDB) GetStorageRoot(addr common.Address) common.Hash {
	return common.Hash{}
}

func (s *mockStateDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash{}
}

func (s *mockStateDB) SetTransientState(addr common.Address, key, value common.Hash) {}

func (s *mockStateDB) SelfDestruct(addr common.Address) uint256.Int {
	account := s.getAccount(addr)
	balance := *account.balance
	account.balance = uint256.NewInt(0)
	return balance
}

func (s *mockStateDB) HasSelfDestructed(addr common.Address) bool {
	return false
}

func (s *mockStateDB) SelfDestruct6780(addr common.Address) (uint256.Int, bool) {
	account := s.getAccount(addr)
	balance := *account.balance
	account.balance = uint256.NewInt(0)
	return balance, true
}

func (s *mockStateDB) Exist(addr common.Address) bool {
	_, exists := s.accounts[addr]
	return exists
}

func (s *mockStateDB) Empty(addr common.Address) bool {
	if !s.Exist(addr) {
		return true
	}
	account := s.getAccount(addr)
	return account.nonce == 0 && account.balance.IsZero() && len(account.code) == 0
}

func (s *mockStateDB) AddressInAccessList(addr common.Address) bool {
	return false
}

func (s *mockStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (bool, bool) {
	return false, false
}

func (s *mockStateDB) AddAddressToAccessList(addr common.Address) {}

func (s *mockStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {}

func (s *mockStateDB) RevertToSnapshot(id int) {}
func (s *mockStateDB) Snapshot() int           { return 0 }

func (s *mockStateDB) AddLog(log *types.Log) {
	s.logs = append(s.logs, log)
}

func (s *mockStateDB) AddPreimage(hash common.Hash, preimage []byte) {}

func (s *mockStateDB) Prepare(rules params.Rules, sender, coinbase common.Address, 
                           dest *common.Address, precompiles []common.Address, 
                           txAccesses types.AccessList) {}

func (s *mockStateDB) Finalise(bool) {}

func (s *mockStateDB) PointCache() *utils.PointCache { return nil }

func (s *mockStateDB) Witness() *stateless.Witness { return nil }

func (s *mockStateDB) AccessEvents() *state.AccessEvents { return nil }

// Copy возвращает копию mockStateDB
func (s *mockStateDB) Copy() StateDB {
	newState := newMockStateDB()
	for addr, account := range s.accounts {
		newAccount := newMockAccount()
		newAccount.balance = new(uint256.Int).Set(account.balance)
		newAccount.nonce = account.nonce
		if len(account.code) > 0 {
			newAccount.code = make([]byte, len(account.code))
			copy(newAccount.code, account.code)
		}
		for key, value := range account.storage {
			newAccount.storage[key] = value
		}
		newState.accounts[addr] = newAccount
	}
	return newState
}

// Тесты для параллельного EVM

func TestParallelExecution(t *testing.T) {
	// Создаем тестовую среду
	chainConfig := params.TestChainConfig
	blockNum := big.NewInt(1)
	blockTime := uint64(time.Now().Unix())
	
	// Создаем приватные ключи для тестовых аккаунтов
	key1, _ := crypto.GenerateKey()
	key2, _ := crypto.GenerateKey()
	key3, _ := crypto.GenerateKey()
	
	addr1 := crypto.PubkeyToAddress(key1.PublicKey)
	addr2 := crypto.PubkeyToAddress(key2.PublicKey)
	addr3 := crypto.PubkeyToAddress(key3.PublicKey)
	
	signer := types.LatestSigner(chainConfig)
	
	// Создаем тестовые транзакции
	tx1 := types.NewTransaction(0, addr2, big.NewInt(1000), 21000, big.NewInt(1), nil)
	signedTx1, _ := types.SignTx(tx1, signer, key1)
	
	tx2 := types.NewTransaction(0, addr3, big.NewInt(2000), 21000, big.NewInt(1), nil)
	signedTx2, _ := types.SignTx(tx2, signer, key2)
	
	// Транзакции, которые должны выполняться параллельно (нет зависимостей)
	tx3 := types.NewTransaction(0, addr2, big.NewInt(500), 21000, big.NewInt(1), nil)
	signedTx3, _ := types.SignTx(tx3, signer, key3)
	
	// Создаем состояние
	stateDB := newMockStateDB()
	stateDB.AddBalance(addr1, uint256.NewInt(10000), tracing.BalanceChangeGenesis)
	stateDB.AddBalance(addr2, uint256.NewInt(10000), tracing.BalanceChangeGenesis)
	stateDB.AddBalance(addr3, uint256.NewInt(10000), tracing.BalanceChangeGenesis)
	
	// Создаем BlockContext
	blockCtx := BlockContext{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     func(uint64) common.Hash { return common.Hash{} },
		Coinbase:    common.Address{},
		BlockNumber: blockNum,
		Time:        blockTime,
		GasLimit:    10000000,
		BaseFee:     big.NewInt(1),
	}
	
	// Создаем ParallelRunner
	runner := NewParallelRunner(chainConfig, Config{}, 4)
	
	// Тест 1: Выполнение одной транзакции
	receipts, gasUsed, err := runner.RunParallelTxs(blockCtx, stateDB, []*types.Transaction{signedTx1})
	assert.NoError(t, err)
	assert.Len(t, receipts, 1)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipts[0].Status)
	assert.Equal(t, uint64(21000), gasUsed)
	
	// Проверяем изменение балансов
	assert.Equal(t, uint256.NewInt(10000-1000-21000), stateDB.GetBalance(addr1))
	assert.Equal(t, uint256.NewInt(10000+1000), stateDB.GetBalance(addr2))
	
	// Тест 2: Выполнение нескольких транзакций с зависимостями
	stateDB = newMockStateDB() // Сбрасываем состояние
	stateDB.AddBalance(addr1, uint256.NewInt(10000), tracing.BalanceChangeGenesis)
	stateDB.AddBalance(addr2, uint256.NewInt(10000), tracing.BalanceChangeGenesis)
	stateDB.AddBalance(addr3, uint256.NewInt(10000), tracing.BalanceChangeGenesis)
	
	receipts, gasUsed, err = runner.RunParallelTxs(blockCtx, stateDB, []*types.Transaction{signedTx1, signedTx2})
	assert.NoError(t, err)
	assert.Len(t, receipts, 2)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipts[0].Status)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipts[1].Status)
	assert.Equal(t, uint64(21000*2), gasUsed)
	
	// Проверяем изменение балансов
	assert.Equal(t, uint256.NewInt(10000-1000-21000), stateDB.GetBalance(addr1))
	assert.Equal(t, uint256.NewInt(10000+1000-2000-21000), stateDB.GetBalance(addr2))
	assert.Equal(t, uint256.NewInt(10000+2000), stateDB.GetBalance(addr3))
	
	// Тест 3: Выполнение нескольких независимых транзакций (проверка параллельного выполнения)
	stateDB = newMockStateDB() // Сбрасываем состояние
	stateDB.AddBalance(addr1, uint256.NewInt(10000), tracing.BalanceChangeGenesis)
	stateDB.AddBalance(addr2, uint256.NewInt(10000), tracing.BalanceChangeGenesis)
	stateDB.AddBalance(addr3, uint256.NewInt(10000), tracing.BalanceChangeGenesis)
	
	// signedTx1 и signedTx3 независимы и могут выполняться параллельно
	receipts, gasUsed, err = runner.RunParallelTxs(blockCtx, stateDB, []*types.Transaction{signedTx1, signedTx3})
	assert.NoError(t, err)
	assert.Len(t, receipts, 2)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipts[0].Status)
	assert.Equal(t, types.ReceiptStatusSuccessful, receipts[1].Status)
	assert.Equal(t, uint64(21000*2), gasUsed)
	
	// Проверяем изменение балансов
	assert.Equal(t, uint256.NewInt(10000-1000-21000), stateDB.GetBalance(addr1))
	assert.Equal(t, uint256.NewInt(10000+1000+500), stateDB.GetBalance(addr2))
	assert.Equal(t, uint256.NewInt(10000-500-21000), stateDB.GetBalance(addr3))
}

func TestTransactionDependencyAnalysis(t *testing.T) {
	// Создаем тестовую среду
	chainConfig := params.TestChainConfig
	blockNum := big.NewInt(1)
	
	// Создаем приватные ключи для тестовых аккаунтов
	key1, _ := crypto.GenerateKey()
	key2, _ := crypto.GenerateKey()
	
	addr1 := crypto.PubkeyToAddress(key1.PublicKey)
	addr2 := crypto.PubkeyToAddress(key2.PublicKey)
	
	signer := types.LatestSigner(chainConfig)
	
	// Создаем транзакции с явной зависимостью
	tx1 := types.NewTransaction(0, addr2, big.NewInt(1000), 21000, big.NewInt(1), nil)
	signedTx1, _ := types.SignTx(tx1, signer, key1)
	
	tx2 := types.NewTransaction(0, addr1, big.NewInt(500), 21000, big.NewInt(1), nil)
	signedTx2, _ := types.SignTx(tx2, signer, key2)
	
	// Транзакции от одного отправителя (неявная зависимость по nonce)
	tx3 := types.NewTransaction(1, addr2, big.NewInt(300), 21000, big.NewInt(1), nil)
	signedTx3, _ := types.SignTx(tx3, signer, key1)
	
	// Анализируем зависимости
	blockNumUint64 := blockNum.Uint64()
	dependencies, err := AnalyzeTransactions([]*types.Transaction{signedTx1, signedTx2, signedTx3}, chainConfig, &blockNumUint64)
	assert.NoError(t, err)
	
	// Проверяем результаты анализа
	// TX2 зависит от TX1, так как TX1 изменяет баланс addr2, а TX2 отправляется от addr2
	assert.Contains(t, dependencies[1], 0)
	
	// TX3 зависит от TX1, так как это транзакция от того же отправителя с последовательным nonce
	assert.Contains(t, dependencies[2], 0)
}

func TestGroupTransactions(t *testing.T) {
	// Создаем граф зависимостей для тестирования
	dependencies := map[int][]int{
		0: {},
		1: {},
		2: {0},
		3: {1},
		4: {2, 3},
	}
	
	// Группируем транзакции
	groups := GroupTransactions(make([]*types.Transaction, 5), dependencies)
	
	// Проверяем результаты группировки
	assert.Len(t, groups, 3) // Должно быть 3 группы
	
	// Группа 1: транзакции без зависимостей (0, 1)
	assert.Len(t, groups[0], 2)
	assert.Contains(t, groups[0], 0)
	assert.Contains(t, groups[0], 1)
	
	// Группа 2: транзакции, зависящие от группы 1 (2, 3)
	assert.Len(t, groups[1], 2)
	assert.Contains(t, groups[1], 2)
	assert.Contains(t, groups[1], 3)
	
	// Группа 3: транзакции, зависящие от группы 2 (4)
	assert.Len(t, groups[2], 1)
	assert.Equal(t, 4, groups[2][0])
} 