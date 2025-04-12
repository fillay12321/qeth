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
	"math"
	"sync"
)

// DeltaCompression обеспечивает сжатие квантовых состояний путем хранения только дельт
type DeltaCompression struct {
	// Базовое состояние, относительно которого вычисляются дельты
	baseState []complex128
	
	// Последнее обработанное состояние
	lastState []complex128
	
	// Порог дельты, ниже которого значения считаются незначительными
	threshold float64
	
	// Сжатые состояния
	compressedStates []CompressedState
	
	// Мьютекс для защиты параллельного доступа
	mutex sync.RWMutex
}

// CompressedState представляет сжатое квантовое состояние
type CompressedState struct {
	// Индексы измененных амплитуд
	Indices []int
	
	// Соответствующие дельты амплитуд
	Deltas []complex128
	
	// Временная метка
	Timestamp int64
}

// NewDeltaCompression создает новый экземпляр для дельта-компрессии
func NewDeltaCompression(initialState []complex128, threshold float64) *DeltaCompression {
	if threshold <= 0 {
		threshold = 1e-6 // Значение по умолчанию
	}
	
	dc := &DeltaCompression{
		threshold:        threshold,
		compressedStates: make([]CompressedState, 0),
	}
	
	if initialState != nil {
		dc.baseState = make([]complex128, len(initialState))
		copy(dc.baseState, initialState)
		
		dc.lastState = make([]complex128, len(initialState))
		copy(dc.lastState, initialState)
	}
	
	return dc
}

// UpdateState обновляет состояние и вычисляет дельту
func (dc *DeltaCompression) UpdateState(newState []complex128) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	
	// Если это первое состояние, инициализируем базовое состояние
	if dc.baseState == nil || len(dc.baseState) == 0 {
		dc.baseState = make([]complex128, len(newState))
		copy(dc.baseState, newState)
		
		dc.lastState = make([]complex128, len(newState))
		copy(dc.lastState, newState)
		
		return
	}
	
	// Проверяем соответствие размеров
	if len(newState) != len(dc.lastState) {
		// Размеры не соответствуют - сбрасываем базовое состояние
		dc.baseState = make([]complex128, len(newState))
		copy(dc.baseState, newState)
		
		dc.lastState = make([]complex128, len(newState))
		copy(dc.lastState, newState)
		
		// Очищаем сжатые состояния
		dc.compressedStates = make([]CompressedState, 0)
		
		return
	}
	
	// Вычисляем дельту и сохраняем только значимые изменения
	indices := make([]int, 0)
	deltas := make([]complex128, 0)
	
	for i := 0; i < len(newState); i++ {
		// Вычисляем дельту
		delta := newState[i] - dc.lastState[i]
		
		// Проверяем, является ли дельта значимой
		magnitude := cmplx128Abs(delta)
		if magnitude > dc.threshold {
			indices = append(indices, i)
			deltas = append(deltas, delta)
		}
	}
	
	// Создаем сжатое состояние
	compressedState := CompressedState{
		Indices:   indices,
		Deltas:    deltas,
		Timestamp: currentTimeNano(),
	}
	
	// Добавляем к списку сжатых состояний
	dc.compressedStates = append(dc.compressedStates, compressedState)
	
	// Обновляем последнее состояние
	copy(dc.lastState, newState)
	
	// Если список сжатых состояний стал слишком большим, удаляем старые
	if len(dc.compressedStates) > 100 {
		// Оставляем только последние 50 состояний
		dc.compressedStates = dc.compressedStates[len(dc.compressedStates)-50:]
	}
}

// ReconstructState восстанавливает полное состояние на основе базового и дельт
func (dc *DeltaCompression) ReconstructState(index int) []complex128 {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	
	// Проверяем, что индекс в пределах диапазона
	if index < 0 || index >= len(dc.compressedStates) {
		return nil
	}
	
	// Начинаем с базового состояния
	result := make([]complex128, len(dc.baseState))
	copy(result, dc.baseState)
	
	// Применяем все дельты до указанного индекса
	for i := 0; i <= index; i++ {
		compressed := dc.compressedStates[i]
		
		// Применяем дельты
		for j, idx := range compressed.Indices {
			result[idx] += compressed.Deltas[j]
		}
	}
	
	return result
}

// GetLastState возвращает последнее сохраненное состояние
func (dc *DeltaCompression) GetLastState() []complex128 {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	
	result := make([]complex128, len(dc.lastState))
	copy(result, dc.lastState)
	
	return result
}

// GetStateCount возвращает количество сохраненных состояний
func (dc *DeltaCompression) GetStateCount() int {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	
	return len(dc.compressedStates) + 1 // +1, так как учитываем базовое состояние
}

// Reset сбрасывает все состояния
func (dc *DeltaCompression) Reset() {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	
	dc.baseState = nil
	dc.lastState = nil
	dc.compressedStates = make([]CompressedState, 0)
}

// SetThreshold устанавливает новый порог для дельт
func (dc *DeltaCompression) SetThreshold(threshold float64) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	
	if threshold <= 0 {
		threshold = 1e-6
	}
	
	dc.threshold = threshold
}

// GetCompressionRatio возвращает коэффициент сжатия (отношение размера сжатых данных к полному размеру)
func (dc *DeltaCompression) GetCompressionRatio() float64 {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	
	if len(dc.compressedStates) == 0 || len(dc.baseState) == 0 {
		return 1.0
	}
	
	totalElements := len(dc.baseState) * len(dc.compressedStates)
	compressedElements := 0
	
	for _, cs := range dc.compressedStates {
		compressedElements += len(cs.Indices)
	}
	
	return float64(compressedElements) / float64(totalElements)
}

// Вспомогательные функции

// cmplx128Abs возвращает модуль комплексного числа
func cmplx128Abs(c complex128) float64 {
	return math.Sqrt(real(c)*real(c) + imag(c)*imag(c))
}

// currentTimeNano возвращает текущее время в наносекундах
func currentTimeNano() int64 {
	return 0 // В реальной реализации здесь был бы вызов time.Now().UnixNano()
} 