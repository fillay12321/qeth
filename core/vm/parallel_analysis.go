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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// AccessRecord хранит информацию о доступе к слоту хранилища
type AccessRecord struct {
	Address    common.Address
	SlotKey    common.Hash
	IsRead     bool
	IsWrite    bool
	IsCreate   bool
	IsCodeRead bool
}

// AccessSet хранит все доступы к хранилищу
type AccessSet struct {
	Records map[common.Address]map[common.Hash]*AccessRecord
}

// NewAccessSet создает пустой AccessSet
func NewAccessSet() *AccessSet {
	return &AccessSet{
		Records: make(map[common.Address]map[common.Hash]*AccessRecord),
	}
}

// AddAccess добавляет запись о доступе к слоту
func (as *AccessSet) AddAccess(addr common.Address, slot common.Hash, isRead, isWrite, isCreate, isCodeRead bool) {
	if _, ok := as.Records[addr]; !ok {
		as.Records[addr] = make(map[common.Hash]*AccessRecord)
	}
	record, exists := as.Records[addr][slot]
	if !exists {
		record = &AccessRecord{
			Address:    addr,
			SlotKey:    slot,
		}
		as.Records[addr][slot] = record
	}
	
	record.IsRead = record.IsRead || isRead
	record.IsWrite = record.IsWrite || isWrite
	record.IsCreate = record.IsCreate || isCreate
	record.IsCodeRead = record.IsCodeRead || isCodeRead
}

// AnalyzeTransactions выполняет статический анализ транзакций для определения зависимостей
func AnalyzeTransactions(txs []*types.Transaction, chainConfig *params.ChainConfig, blockNumber *uint64) (map[int][]int, error) {
	accessSets := make([]*AccessSet, len(txs))
	
	// Для каждой транзакции пытаемся предсказать, к каким данным она обратится
	for i, tx := range txs {
		accessSets[i] = AnalyzeTransaction(tx, chainConfig, blockNumber)
	}
	
	// Строим граф зависимостей
	dependencies := make(map[int][]int)
	for i := 0; i < len(txs); i++ {
		dependencies[i] = []int{}
		
		// Проверяем все предыдущие транзакции
		for j := 0; j < i; j++ {
			if HasDependency(accessSets[i], accessSets[j]) {
				dependencies[i] = append(dependencies[i], j)
			}
		}
	}
	
	return dependencies, nil
}

// AnalyzeTransaction анализирует отдельную транзакцию и пытается определить, к каким данным
// она обратится при выполнении
func AnalyzeTransaction(tx *types.Transaction, chainConfig *params.ChainConfig, blockNumber *uint64) *AccessSet {
	accessSet := NewAccessSet()

	// Получаем отправителя транзакции
	signer := types.MakeSigner(chainConfig, blockNumber)
	sender, err := types.Sender(signer, tx)
	if err != nil {
		log.Warn("Failed to get sender", "tx", tx.Hash(), "err", err)
		return accessSet
	}
	
	// Добавляем доступ к балансу и nonce отправителя
	balanceSlot := crypto.Keccak256Hash([]byte("balance"), sender.Bytes())
	nonceSlot := crypto.Keccak256Hash([]byte("nonce"), sender.Bytes())
	accessSet.AddAccess(sender, balanceSlot, true, true, false, false)  // Баланс: чтение и запись
	accessSet.AddAccess(sender, nonceSlot, true, true, false, false)    // Nonce: чтение и запись
	
	// Если есть получатель, добавляем доступ к его балансу
	if tx.To() != nil {
		recipientBalanceSlot := crypto.Keccak256Hash([]byte("balance"), tx.To().Bytes())
		accessSet.AddAccess(*tx.To(), recipientBalanceSlot, true, true, false, false)
		
		// Если это вызов контракта, добавляем доступ к коду
		codeSlot := crypto.Keccak256Hash([]byte("code"), tx.To().Bytes())
		accessSet.AddAccess(*tx.To(), codeSlot, true, false, false, true)
		
		// Для контрактов также можно проводить дополнительный статический анализ
		// данных транзакции, чтобы определить, какие слоты хранилища могут быть затронуты
		AnalyzeCallData(tx.Data(), accessSet, *tx.To())
	} else {
		// Создание контракта
		contractAddr := crypto.CreateAddress(sender, tx.Nonce())
		codeSlot := crypto.Keccak256Hash([]byte("code"), contractAddr.Bytes())
		accessSet.AddAccess(contractAddr, codeSlot, false, true, true, false)
	}
	
	return accessSet
}

// AnalyzeCallData анализирует данные вызова контракта, чтобы предсказать
// доступы к хранилищу на основе сигнатуры функции
func AnalyzeCallData(data []byte, accessSet *AccessSet, contractAddr common.Address) {
	if len(data) < 4 {
		return
	}
	
	// Получаем сигнатуру функции (первые 4 байта)
	signature := data[:4]
	
	// В реальной реализации здесь будет более сложный анализ на основе
	// известных сигнатур функций, ABI контрактов и т.д.
	
	// Пример: для ERC20 токенов
	transferSig := crypto.Keccak256([]byte("transfer(address,uint256)"))[:4]
	if common.BytesToHash(signature).Big().Cmp(common.BytesToHash(transferSig).Big()) == 0 {
		// Токен transfer затрагивает балансы отправителя и получателя
		if len(data) >= 4+32+32 {
			// Извлекаем адрес получателя (пропускаем сигнатуру и отступ)
			recipientAddr := common.BytesToAddress(data[4+12 : 4+32])
			
			// Добавляем доступы к балансам
			senderBalanceSlot := GetERC20BalanceSlot(contractAddr, contractAddr)
			recipientBalanceSlot := GetERC20BalanceSlot(contractAddr, recipientAddr)
			
			accessSet.AddAccess(contractAddr, senderBalanceSlot, true, true, false, false)
			accessSet.AddAccess(contractAddr, recipientBalanceSlot, true, true, false, false)
		}
	}
}

// GetERC20BalanceSlot возвращает слот хранилища для баланса ERC20 токена
func GetERC20BalanceSlot(token common.Address, owner common.Address) common.Hash {
	// Эта формула зависит от конкретной реализации контракта
	// В большинстве ERC20: mapping(address => uint256) balances
	return crypto.Keccak256Hash(owner.Bytes(), common.BigToHash(common.Big0).Bytes())
}

// HasDependency проверяет, есть ли зависимость между двумя транзакциями
// на основе их доступов к хранилищу
func HasDependency(accessSet1, accessSet2 *AccessSet) bool {
	// Проверяем конфликты Write-Write, Read-Write, Write-Read
	for addr1, slots1 := range accessSet1.Records {
		for slot1, record1 := range slots1 {
			if slots2, ok := accessSet2.Records[addr1]; ok {
				if record2, ok := slots2[slot1]; ok {
					// Write-Write конфликт
					if record1.IsWrite && record2.IsWrite {
						return true
					}
					
					// Read-Write конфликт
					if record1.IsRead && record2.IsWrite {
						return true
					}
					
					// Write-Read конфликт
					if record1.IsWrite && record2.IsRead {
						return true
					}
				}
			}
		}
	}
	
	return false
}

// GroupTransactions группирует транзакции для параллельного выполнения
func GroupTransactions(txs []*types.Transaction, dependencies map[int][]int) [][]int {
	// Упрощенная реализация: группируем транзакции по уровням зависимостей
	groups := [][]int{}
	processed := make(map[int]bool)
	
	for len(processed) < len(txs) {
		group := []int{}
		
		for i := 0; i < len(txs); i++ {
			if processed[i] {
				continue
			}
			
			canAdd := true
			for _, dep := range dependencies[i] {
				if !processed[dep] {
					canAdd = false
					break
				}
			}
			
			if canAdd {
				group = append(group, i)
			}
		}
		
		if len(group) == 0 {
			// Защита от цикличных зависимостей
			break
		}
		
		for _, idx := range group {
			processed[idx] = true
		}
		
		groups = append(groups, group)
	}
	
	return groups
} 