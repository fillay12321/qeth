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
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// Config are the configuration options for the Interpreter
type Config struct {
	Debug                   bool      // Включен ли режим отладки
	Tracer                  EVMLogger // Логгер для трассировки выполнения
	NoRecursion             bool      // Запрещать вложенные вызовы?
	NoBaseFee               bool      // Делать ли базовую плату равной нулю
	EnablePreimageRecording bool      // Включить запись preimage
	ExtraEips               []int     // Дополнительные EIP для активации
	HasEIP1153              bool      // Активация расширенного хранилища MCOPY
	
	// Настройки квантового процессора Quest
	EnableQuest               bool                  // Включить квантовый процессор Quest
	QuestHardwareAcceleration bool                  // Использовать аппаратное ускорение для квантовых вычислений
	QuestOptions              map[string]string     // Дополнительные настройки для квантового процессора
	QuestAutodetectHardware   bool                  // Автоматически определять и оптимизировать под оборудование
	QuestNumQubits            int                   // Количество кубитов (0 = автоопределение)
	QuestDeltaCompression     bool                  // Использовать дельта-сжатие для оптимизации памяти
	QuestPreferredDevice      int                   // Предпочтительное CUDA устройство (-1 = автоопределение)
	QuestDeltaThreshold       float64               // Порог для дельта-сжатия (минимально значимое изменение)
	QuestForceGPU             bool                  // Принудительно использовать GPU даже если автоопределение не рекомендует
	QuestForceCPU             bool                  // Принудительно использовать CPU независимо от наличия GPU

	StatelessSelfValidation bool // Generate execution witnesses and self-check against them (testing purpose)

	// Параллельное выполнение транзакций
	EnableParallelExecution bool // Включает параллельное выполнение операций
	ParallelThreads         int  // Количество потоков для параллельного выполнения (0 = автоматически)
	
	// Новые настройки для масштабирования TPS
	HyperParallelMode      bool  // Режим гипер-параллелизма для достижения 1M TPS
	ShardedExecution       bool  // Выполнение в шардах для лучшего параллелизма
	MaxBatchSize           int   // Максимальный размер пакета транзакций для обработки
	ConflictResolution     string // Метод разрешения конфликтов (optimistic, pessimistic, speculative)
	QuestTPSBooster        bool  // Активировать ускоритель TPS на базе Quest
	TPSTargetMultiplier    int   // Целевой множитель TPS (1000 = 1M TPS)

	// Расширения конфигурации для поддержки Quest
	QuestDebug              bool                   // Включает отладочный режим для квантового процессора
	QuestLevelParallelism   int                    // Уровень параллелизма для квантовых вычислений (0=авто)
	QuestProfiling          bool                   // Включить профилирование производительности квантового процессора
	QuestBenchmarkOnStart   bool                   // Выполнить тест производительности при запуске

	// Настройки квантового процессора
	QuestHardwareAccelerationOptions map[string]interface{} // Дополнительные настройки для квантового процессора Quest

	// Ценовые модели
	BaseFeePerByteForNonZeroes common.Hash
	BaseFeePerByteForZeroes    common.Hash
	BlobBaseFee                *big.Int // Стоимость blob-данных (EIP-4844)
}

// NewConfig создает новую конфигурацию для EVM с квантовой поддержкой
func NewConfig() Config {
	return Config{
		Debug:                   false,
		Tracer:                  nil,
		NoRecursion:             false,
		NoBaseFee:               false,
		EnablePreimageRecording: false,
		ExtraEips:               nil,
		
		// Квантовые вычисления включены по умолчанию
		EnableQuest:               true,
		QuestHardwareAcceleration: true,
		QuestAutodetectHardware:   true,
		QuestNumQubits:            0,  // Автоопределение
		QuestDeltaCompression:     true,
		QuestPreferredDevice:      -1, // Автоопределение
		QuestDeltaThreshold:       1e-6, // Порог для дельта-сжатия
		QuestForceGPU:             true, // Всегда использовать GPU для максимальной производительности
		QuestForceCPU:             false,
		QuestOptions:              make(map[string]string),
		
		// Параллельное выполнение
		EnableParallelExecution:  true,
		ParallelThreads:          0, // Автоматически использовать все доступные ядра
		
		// Настройки для 1M TPS
		HyperParallelMode:        true,
		ShardedExecution:         true,
		MaxBatchSize:             10000,
		ConflictResolution:       "speculative",
		QuestTPSBooster:          true,
		TPSTargetMultiplier:      1000,
		
		// Отладка и профилирование
		QuestDebug:                false,
		QuestLevelParallelism:     0, // Автоматический выбор
		QuestProfiling:            true,
		QuestBenchmarkOnStart:     true, // Проводить тест производительности при старте
		
		QuestHardwareAccelerationOptions: make(map[string]interface{}),
	}
}

// DefaultConfig предоставляет конфигурацию по умолчанию с поддержкой Quest
var DefaultConfig = NewConfig() 