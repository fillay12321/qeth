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

// package utils предоставляет вспомогательные функции для квантового процессора
package utils

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// OperationStats содержит статистику по операции
type OperationStats struct {
	Count        int64
	TotalTime    time.Duration
	MinTime      time.Duration
	MaxTime      time.Duration
	AvgTime      time.Duration
	LastCallTime time.Time
}

// Profiler предоставляет средства для профилирования производительности
type Profiler struct {
	// Статистика по операциям
	operationStats map[string]*OperationStats
	
	// Текущие операции (имя -> время начала)
	currentOperations map[string]time.Time
	
	// Мьютекс для защиты параллельного доступа
	mutex sync.Mutex
	
	// Включен ли профайлер
	enabled bool
}

// NewProfiler создает новый профайлер
func NewProfiler() *Profiler {
	return &Profiler{
		operationStats:    make(map[string]*OperationStats),
		currentOperations: make(map[string]time.Time),
		enabled:           true,
	}
}

// StartOperation отмечает начало операции
func (p *Profiler) StartOperation(operationName string) {
	if !p.enabled {
		return
	}
	
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// Отмечаем время начала операции
	p.currentOperations[operationName] = time.Now()
}

// EndOperation отмечает конец операции и обновляет статистику
func (p *Profiler) EndOperation(operationName string) time.Duration {
	if !p.enabled {
		return 0
	}
	
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// Проверяем, была ли начата операция
	startTime, ok := p.currentOperations[operationName]
	if !ok {
		return 0
	}
	
	// Вычисляем длительность операции
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	
	// Удаляем операцию из списка текущих
	delete(p.currentOperations, operationName)
	
	// Обновляем статистику
	stats, ok := p.operationStats[operationName]
	if !ok {
		// Создаем новую статистику для операции
		stats = &OperationStats{
			Count:        0,
			TotalTime:    0,
			MinTime:      duration,
			MaxTime:      duration,
			AvgTime:      duration,
			LastCallTime: endTime,
		}
		p.operationStats[operationName] = stats
	}
	
	// Обновляем статистику
	stats.Count++
	stats.TotalTime += duration
	
	// Обновляем минимальное время
	if duration < stats.MinTime {
		stats.MinTime = duration
	}
	
	// Обновляем максимальное время
	if duration > stats.MaxTime {
		stats.MaxTime = duration
	}
	
	// Вычисляем среднее время
	stats.AvgTime = time.Duration(stats.TotalTime.Nanoseconds() / stats.Count)
	
	// Обновляем время последнего вызова
	stats.LastCallTime = endTime
	
	return duration
}

// GetOperationStats возвращает статистику по конкретной операции
func (p *Profiler) GetOperationStats(operationName string) *OperationStats {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	stats, ok := p.operationStats[operationName]
	if !ok {
		return nil
	}
	
	return stats
}

// GetAllOperationStats возвращает статистику по всем операциям
func (p *Profiler) GetAllOperationStats() map[string]*OperationStats {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// Создаем копию, чтобы избежать гонок данных
	result := make(map[string]*OperationStats)
	
	for name, stats := range p.operationStats {
		statsCopy := *stats
		result[name] = &statsCopy
	}
	
	return result
}

// Reset сбрасывает всю статистику
func (p *Profiler) Reset() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.operationStats = make(map[string]*OperationStats)
	p.currentOperations = make(map[string]time.Time)
}

// Enable включает профайлер
func (p *Profiler) Enable() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.enabled = true
}

// Disable выключает профайлер
func (p *Profiler) Disable() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.enabled = false
}

// IsEnabled возвращает текущий статус профайлера
func (p *Profiler) IsEnabled() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	return p.enabled
}

// LogStatistics выводит статистику в лог
func (p *Profiler) LogStatistics() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	log.Info("=== Статистика профайлера ===")
	
	if len(p.operationStats) == 0 {
		log.Info("Нет данных для вывода")
		return
	}
	
	for name, stats := range p.operationStats {
		log.Info(fmt.Sprintf("Операция: %s", name),
			"вызовов", stats.Count,
			"общее_время_мс", stats.TotalTime.Milliseconds(),
			"мин_мс", stats.MinTime.Milliseconds(),
			"макс_мс", stats.MaxTime.Milliseconds(),
			"сред_мс", stats.AvgTime.Milliseconds())
	}
	
	log.Info("============================")
}

// GetAverageTime возвращает среднее время выполнения операции
func (p *Profiler) GetAverageTime(operationName string) time.Duration {
	stats := p.GetOperationStats(operationName)
	if stats == nil {
		return 0
	}
	
	return stats.AvgTime
}

// GetTotalTime возвращает общее время, затраченное на операцию
func (p *Profiler) GetTotalTime(operationName string) time.Duration {
	stats := p.GetOperationStats(operationName)
	if stats == nil {
		return 0
	}
	
	return stats.TotalTime
}

// GetCallCount возвращает количество вызовов операции
func (p *Profiler) GetCallCount(operationName string) int64 {
	stats := p.GetOperationStats(operationName)
	if stats == nil {
		return 0
	}
	
	return stats.Count
} 