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
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// ParallelRunner предоставляет интерфейс для запуска EVM в параллельном режиме
type ParallelRunner struct {
	chainConfig *params.ChainConfig
	vmConfig    Config
	maxWorkers  int
}

// NewParallelRunner создает новый экземпляр ParallelRunner
func NewParallelRunner(chainConfig *params.ChainConfig, vmConfig Config, maxWorkers int) *ParallelRunner {
	if maxWorkers <= 0 {
		// По умолчанию используем GOMAXPROCS
		maxWorkers = 4 // Можно получить из runtime.GOMAXPROCS(0)
	}
	return &ParallelRunner{
		chainConfig: chainConfig,
		vmConfig:    vmConfig,
		maxWorkers:  maxWorkers,
	}
}

// RunParallelTxs выполняет транзакции параллельно и возвращает результаты
func (r *ParallelRunner) RunParallelTxs(blockCtx BlockContext, stateDB StateDB, txs []*types.Transaction) ([]*types.Receipt, uint64, error) {
	if len(txs) == 0 {
		return nil, 0, nil
	}
	
	// Улучшенный анализ зависимостей между транзакциями
	dependencyGraph, err := AnalyzeTransactions(txs, r.chainConfig, blockCtx.BlockNumber)
	if err != nil {
		log.Warn("Failed to analyze transaction dependencies, falling back to sequential execution", "err", err)
		// Упрощенный подход - каждая транзакция зависит от предыдущей
		dependencyGraph = make(map[int][]int)
		for i := 1; i < len(txs); i++ {
			dependencyGraph[i] = []int{i - 1}
		}
	}
	
	// Группировка транзакций для оптимального параллельного выполнения
	groups := GroupTransactions(txs, dependencyGraph)
	log.Debug("Grouped transactions for parallel execution", "txCount", len(txs), "groupCount", len(groups))
	
	// Создаем ParallelEVM
	pevm := NewParallelEVM(blockCtx, stateDB, r.chainConfig, r.vmConfig, r.maxWorkers)
	
	// Результаты выполнения для всех транзакций
	results := make([]*ExecutionResult, len(txs))
	var totalGasUsed uint64
	
	// Выполняем группы транзакций последовательно, а внутри группы - параллельно
	for groupIdx, group := range groups {
		log.Debug("Processing transaction group", "groupIndex", groupIdx, "txCount", len(group))
		
		// Получаем список транзакций в этой группе
		groupTxs := make([]*types.Transaction, len(group))
		for i, txIdx := range group {
			groupTxs[i] = txs[txIdx]
		}
		
		// Выполняем группу транзакций параллельно
		groupResults, err := pevm.ProcessTransactions(groupTxs)
		if err != nil {
			return nil, 0, err
		}
		
		// Сохраняем результаты в общем массиве
		for i, txIdx := range group {
			results[txIdx] = groupResults[i]
		}
	}
	
	// Обрабатываем результаты
	receipts := make([]*types.Receipt, len(txs))
	
	// Создаем чеки для всех транзакций
	for i, result := range results {
		if result == nil {
			// Это не должно происходить при правильной реализации
			log.Error("Missing transaction result", "index", i)
			continue
		}
		
		if result.Err != nil {
			log.Debug("Transaction failed", "index", i, "hash", txs[i].Hash(), "error", result.Err)
		}
		
		// Создаем чек для транзакции
		receipt := &types.Receipt{
			Type:              txs[i].Type(),
			CumulativeGasUsed: totalGasUsed + result.UsedGas,
			TxHash:            txs[i].Hash(),
			GasUsed:           result.UsedGas,
		}
		
		if result.Err != nil {
			receipt.Status = types.ReceiptStatusFailed
		} else {
			receipt.Status = types.ReceiptStatusSuccessful
		}
		
		receipts[i] = receipt
		totalGasUsed += result.UsedGas
	}
	
	return receipts, totalGasUsed, nil
}

// Вспомогательная функция для получения транзакций, зависящих от данной
func getDependents(graph map[int][]int, txIndex int) []int {
	dependents := []int{}
	for depIdx, deps := range graph {
		for _, dep := range deps {
			if dep == txIndex {
				dependents = append(dependents, depIdx)
				break
			}
		}
	}
	return dependents
}

// EnableParallelExecution включает параллельное выполнение в конфигурации
func EnableParallelExecution(config *params.ChainConfig) {
	// Условная имплементация - в реальном коде здесь будет 
	// обновление соответствующего флага в конфигурации
}

// IsParallelExecutionEnabled проверяет, включено ли параллельное выполнение
func IsParallelExecutionEnabled(config *params.ChainConfig) bool {
	// Упрощенная реализация
	return true
} 