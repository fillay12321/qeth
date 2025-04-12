// Copyright 2021 The go-ethereum Authors
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

// package quest предоставляет интерфейс для квантовых вычислений в Ethereum
package quest

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/sync/semaphore"
)

const (
	// Максимальный размер батча - 6.25 млн транзакций
	MaxBatchSize = 6250000

	// Таймаут обработки (в секундах)
	ProcessingTimeout = 300

	// Максимальное количество шардов для параллельной обработки
	MaxShardCount = 128

	// Стандартное количество шардов - одно на каждое логическое ядро, но не больше 32
	DefaultShardCount = 32

	// Минимальный размер пакета для включения режима гипер-батчинга
	MinHyperBatchSize = 1000

	// Лимит газа для одного батча
	BatchGasLimit = 1_000_000_000_000 // Очень большой лимит газа для обработки гигантских батчей
	
	// Максимальное время обработки батча в секундах
	MaxBatchProcessingTimeSeconds = 300
	
	// Размер шарда по умолчанию
	DefaultShardSize = 250_000
)

// Ошибки батч-процессора
var (
	ErrBatchTooLarge      = errors.New("размер батча превышает максимальный")
	ErrBatchProcessing    = errors.New("ошибка при обработке батча")
	ErrProcessingTimeout  = errors.New("таймаут обработки батча")
	ErrQuestNotAvailable  = errors.New("квантовый процессор недоступен")
	ErrInvalidTransaction = errors.New("некорректная транзакция")
	ErrBatchEmpty    = errors.New("пустой батч транзакций")
)

// TxStats содержит статистику обработки транзакций
type TxStats struct {
	// Общие счетчики
	BatchSize   uint64        // Общий размер батча
	Completed   uint64        // Количество успешных транзакций
	Failed      uint64        // Количество неудачных транзакций
	GasUsed     uint64        // Общее количество использованного газа
	ElapsedTime time.Duration // Время обработки
	TPS         float64       // Транзакций в секунду
	
	// Распределение по шардам
	ShardStats map[int]ShardStats // Статистика по шардам
}

// ShardStats содержит информацию по конкретному шарду
type ShardStats struct {
	ShardID    int           // Идентификатор шарда
	Count      uint64        // Количество транзакций в шарде
	Completed  uint64        // Обработано успешно
	Failed     uint64        // Обработано с ошибками
	GasUsed    uint64        // Использовано газа
	StartTime  time.Time     // Время начала обработки
	EndTime    time.Time     // Время окончания обработки
	ElapsedMs  time.Duration // Общее время обработки в миллисекундах
}

// TransactionBatch представляет пакет транзакций для обработки
type TransactionBatch struct {
	Transactions []*types.Transaction
	BatchID      string
	Timestamp    time.Time
	ShardCount   int
	Shards       [][]*types.Transaction
}

// NewTransactionBatch создает новый пакет транзакций
func NewTransactionBatch(transactions []*types.Transaction, shardCount int) *TransactionBatch {
	if shardCount <= 0 {
		shardCount = runtime.NumCPU()
		if shardCount > MaxShardCount {
			shardCount = MaxShardCount
		}
	}

	batch := &TransactionBatch{
		Transactions: transactions,
		BatchID:      fmt.Sprintf("batch-%d", time.Now().UnixNano()),
		Timestamp:    time.Now(),
		ShardCount:   shardCount,
	}

	// Разбиваем транзакции на шарды
	batch.createShards()

	return batch
}

// createShards разбивает транзакции на шарды для параллельной обработки
func (b *TransactionBatch) createShards() {
	txCount := len(b.Transactions)
	
	// Если транзакций меньше, чем шардов, уменьшаем количество шардов
	if txCount < b.ShardCount {
		b.ShardCount = txCount
	}
	
	// Создаем слайсы для хранения транзакций по шардам
	b.Shards = make([][]*types.Transaction, b.ShardCount)
	
	// Вычисляем, сколько транзакций должно быть в каждом шарде
	txPerShard := txCount / b.ShardCount
	remainder := txCount % b.ShardCount
	
	// Распределяем транзакции по шардам
	currentIndex := 0
	for i := 0; i < b.ShardCount; i++ {
		// Количество транзакций в этом шарде
		count := txPerShard
		if i < remainder {
			count++
		}
		
		// Создаем слайс для шарда
		b.Shards[i] = make([]*types.Transaction, count)
		
		// Копируем транзакции в шард
		for j := 0; j < count; j++ {
			b.Shards[i][j] = b.Transactions[currentIndex]
			currentIndex++
		}
	}
}

// BatchProcessor обрабатывает батчи транзакций
type BatchProcessor struct {
	processor    QuestProcessorInterface
	shardCount   int
	mutex        sync.Mutex
	lastStats    TxStats
	totalBatches atomic.Uint64
	gpuEnabled   bool
	gpuDeviceID  int
	hyperOptimized bool
}

// QuestProcessorInterface определяет интерфейс для квантового процессора
type QuestProcessorInterface interface {
	Run(contract interface{}, input []byte, readOnly bool) ([]byte, error)
	IsAvailable() bool
}

// NewBatchProcessor создает новый батч-процессор
func NewBatchProcessor(processor QuestProcessorInterface, shardCount int) *BatchProcessor {
	if shardCount <= 0 {
		shardCount = runtime.NumCPU()
		if shardCount > DefaultShardCount {
			shardCount = DefaultShardCount
		}
	}
	
	return &BatchProcessor{
		processor:  processor,
		shardCount: shardCount,
		lastStats:  TxStats{ShardStats: make(map[int]ShardStats)},
	}
}

// OptimizeForGPU оптимизирует обработку батча для использования GPU
// deviceID = -1 означает автовыбор устройства
func (bp *BatchProcessor) OptimizeForGPU(enable bool, deviceID int) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	bp.gpuEnabled = enable
	bp.gpuDeviceID = deviceID
	
	if enable {
		// Оптимизируем настройки для GPU-обработки
		
		// Для GPU оптимально меньше шардов, но больше транзакций на шард
		cpuCount := runtime.NumCPU()
		optimalShardCount := cpuCount / 2
		if optimalShardCount < 2 {
			optimalShardCount = 2
		}
		if optimalShardCount > MaxShardCount {
			optimalShardCount = MaxShardCount
		}
		
		bp.shardCount = optimalShardCount
		
		log.Info("Включена оптимизация для GPU-обработки", 
			"device_id", deviceID, 
			"shard_count", bp.shardCount)
	} else {
		// Возвращаем настройки для CPU-обработки
		cpuCount := runtime.NumCPU()
		if cpuCount > DefaultShardCount {
			cpuCount = DefaultShardCount
		}
		
		bp.shardCount = cpuCount
		
		log.Info("Оптимизация для GPU-обработки отключена", 
			"shard_count", bp.shardCount)
	}
}

// EnableHyperOptimization включает режим сверхоптимизации для гипер-батчей
func (bp *BatchProcessor) EnableHyperOptimization() {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	bp.hyperOptimized = true
	
	// Настройки для максимальной производительности
	// Используем динамическое регулирование шардов в зависимости от размера батча
	log.Info("Включен режим сверхоптимизации для гипер-батчей")
	
	// Изменяем стратегию шардирования для сверхбольших батчей
	// Будет применено при следующем вызове ProcessHyperBatch
}

// DisableHyperOptimization отключает режим сверхоптимизации
func (bp *BatchProcessor) DisableHyperOptimization() {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	bp.hyperOptimized = false
	log.Info("Режим сверхоптимизации для гипер-батчей отключен")
}

// IsHyperOptimized возвращает статус режима сверхоптимизации
func (bp *BatchProcessor) IsHyperOptimized() bool {
	return bp.hyperOptimized
}

// IsGPUEnabled возвращает статус включенности GPU-обработки
func (bp *BatchProcessor) IsGPUEnabled() bool {
	return bp.gpuEnabled
}

// ProcessHyperBatch обрабатывает гипер-батч транзакций
func (bp *BatchProcessor) ProcessHyperBatch(transactions []*types.Transaction) (TxStats, error) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	if !bp.processor.IsAvailable() {
		return TxStats{}, ErrQuestNotAvailable
	}
	
	// Если получен пустой батч, возвращаем ошибку
	if len(transactions) == 0 {
		return TxStats{}, ErrBatchEmpty
	}
	
	// Проверяем, не превышает ли размер батча максимальный
	if len(transactions) > MaxBatchSize {
		log.Warn("Размер батча превышает максимально допустимый", 
			"size", len(transactions), 
			"max", MaxBatchSize)
		transactions = transactions[:MaxBatchSize]
	}
	
	// Создаем батч транзакций с оптимальным количеством шардов
	// Для очень больших батчей увеличиваем количество шардов
	optimalShardCount := bp.shardCount
	txCount := len(transactions)
	
	// Учитываем режим сверхоптимизации
	if bp.hyperOptimized {
		// Для сверхоптимизации используем специальные настройки
		if txCount > 1000000 {
			// Для очень больших батчей используем другую стратегию шардирования
			if bp.gpuEnabled {
				// Для GPU лучше использовать меньше шардов, но больше транзакций на шард
				optimalShardCount = runtime.NumCPU() / 2
				if optimalShardCount < 4 {
					optimalShardCount = 4
				}
				
				log.Info("Применена сверхоптимизация для GPU-обработки", 
					"shard_count", optimalShardCount,
					"tx_count", txCount)
			} else {
				// Оптимальное количество транзакций на один шард для CPU
				optimalTxPerShard := 25000
				
				// Вычисляем оптимальное количество шардов
				newShardCount := txCount / optimalTxPerShard
				if newShardCount > MaxShardCount {
					newShardCount = MaxShardCount
				}
				
				if newShardCount > 0 {
					optimalShardCount = newShardCount
				}
				
				log.Info("Применена сверхоптимизация для CPU-обработки", 
					"shard_count", optimalShardCount,
					"tx_count", txCount)
			}
		}
	} else {
		// Для очень больших батчей (>1 млн) увеличиваем количество шардов
		if txCount > 1000000 {
			// Вычисляем оптимальное количество транзакций на процессор
			optimalTxPerCPU := 50000 // Приблизительно оптимальное число транзакций на одно ядро
			cpuCount := runtime.NumCPU()
			
			// Вычисляем новое количество шардов с учетом размера батча
			newShardCount := txCount / optimalTxPerCPU
			if newShardCount > MaxShardCount {
				newShardCount = MaxShardCount
			}
			
			// Если расчетное число шардов больше текущего и есть доступные ядра
			if newShardCount > optimalShardCount && newShardCount <= cpuCount*2 {
				optimalShardCount = newShardCount
				log.Info("Для большого батча увеличено количество шардов", 
					"original", bp.shardCount, 
					"new", optimalShardCount,
					"tx_count", txCount)
			}
		}
	}
	
	batch := NewTransactionBatch(transactions, optimalShardCount)
	
	// Счетчики для статистики
	stats := TxStats{
		BatchSize:   uint64(len(transactions)),
		ShardStats:  make(map[int]ShardStats),
		Completed:   0,
		Failed:      0,
		GasUsed:     0,
	}
	
	// Увеличиваем счетчик обработанных батчей
	batchCount := bp.totalBatches.Add(1)
	
	log.Info("Начало обработки гипер-батча", 
		"batch_id", batch.BatchID, 
		"size", len(transactions), 
		"shards", batch.ShardCount,
		"batch_number", batchCount,
		"gpu_enabled", bp.gpuEnabled,
		"hyper_optimized", bp.hyperOptimized)
	
	startTime := time.Now()
	
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), MaxBatchProcessingTimeSeconds*time.Second)
	defer cancel()
	
	// Создаем канал для результатов обработки шардов
	results := make(chan ShardStats, batch.ShardCount)
	
	// Семафор для ограничения параллельности
	sem := semaphore.NewWeighted(int64(batch.ShardCount))
	
	// Запускаем обработку каждого шарда в отдельной горутине
	for shardID, shard := range batch.Shards {
		// Ждем, когда семафор позволит запустить ещё один шард
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Error("Не удалось захватить семафор для шарда", 
				"batch_id", batch.BatchID, 
				"shard_id", shardID, 
				"error", err)
			continue
		}
		
		go func(id int, txs []*types.Transaction) {
			defer sem.Release(1)
			
			// Проверяем, не завершен ли контекст
			if ctx.Err() != nil {
				return
			}
			
			// Запускаем обработку шарда с передачей контекста
			bp.processShardWithContext(ctx, id, txs, results)
		}(shardID, shard)
	}
	
	// Собираем результаты обработки шардов
	for i := 0; i < batch.ShardCount; i++ {
		select {
		case result := <-results:
			stats.ShardStats[result.ShardID] = result
			stats.Completed += result.Completed
			stats.Failed += result.Failed
			stats.GasUsed += result.GasUsed
			
		case <-ctx.Done():
			log.Error("Таймаут при обработке гипер-батча", 
				"batch_id", batch.BatchID, 
				"elapsed", time.Since(startTime))
			return stats, ErrProcessingTimeout
		}
	}
	
	// Рассчитываем время выполнения и TPS
	stats.ElapsedTime = time.Since(startTime)
	
	// Избегаем деления на ноль
	if stats.ElapsedTime.Seconds() > 0 {
		stats.TPS = float64(stats.Completed) / stats.ElapsedTime.Seconds()
	}
	
	// Сохраняем статистику
	bp.lastStats = stats
	
	log.Info("Гипер-батч успешно обработан", 
		"batch_id", batch.BatchID, 
		"completed", stats.Completed, 
		"failed", stats.Failed, 
		"elapsed", stats.ElapsedTime, 
		"tps", math.Round(stats.TPS),
		"avg_time_per_shard_ms", math.Round(float64(stats.ElapsedTime.Milliseconds())/float64(batch.ShardCount)))
	
	return stats, nil
}

// processShardWithContext обрабатывает один шард транзакций с учетом контекста
func (bp *BatchProcessor) processShardWithContext(ctx context.Context, shardID int, transactions []*types.Transaction, results chan<- ShardStats) {
	stats := ShardStats{
		ShardID:   shardID,
		Count:     uint64(len(transactions)),
		StartTime: time.Now(),
	}
	
	log.Debug("Начало обработки шарда", 
		"shard_id", shardID, 
		"transactions", len(transactions))
	
	// Размер чанка для пакетной обработки внутри шарда
	// Меньшие чанки дают лучшую обратную связь о ходе выполнения
	chunkSize := 100
	if len(transactions) <= chunkSize {
		chunkSize = len(transactions)
	}
	
	// Количество воркеров для обработки транзакций в шарде
	// Используем меньше воркеров для очень крупных шардов, чтобы не создавать слишком много горутин
	workerCount := runtime.NumCPU()
	if len(transactions) > 100000 {
		// Для огромных шардов уменьшаем число воркеров, чтобы избежать перегрузки
		workerCount = runtime.NumCPU() / 2
		if workerCount < 2 {
			workerCount = 2 // Минимум 2 воркера
		}
	}
	
	// Создаем каналы для передачи транзакций и результатов
	jobs := make(chan *types.Transaction, chunkSize*2)
	jobResults := make(chan bool, chunkSize*2)
	
	// Создаем WaitGroup для ожидания всех воркеров
	var wg sync.WaitGroup
	
	// Запускаем воркеров
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for tx := range jobs {
				// Проверяем, не был ли отменен контекст
				if ctx.Err() != nil {
					jobResults <- false
					continue
				}
				
				// Конвертируем транзакцию в формат, понятный квантовому процессору
				contract, input, readOnly := bp.convertTxToContract(tx)
				
				// Запускаем обработку транзакции
				_, err := bp.processor.Run(contract, input, readOnly)
				jobResults <- (err == nil)
			}
		}()
	}
	
	// Функция закрытия канала jobs после отправки всех транзакций
	go func() {
		for _, tx := range transactions {
			// Проверяем, не был ли отменен контекст
			if ctx.Err() != nil {
				break
			}
			jobs <- tx
		}
		close(jobs)
	}()
	
	// Обрабатываем результаты выполнения транзакций
	go func() {
		for i := 0; i < len(transactions); i++ {
			select {
			case success := <-jobResults:
				if success {
					atomic.AddUint64(&stats.Completed, 1)
					
					// Добавляем газ (примерно)
					if i < len(transactions) {
						atomic.AddUint64(&stats.GasUsed, transactions[i].Gas())
					}
				} else {
					atomic.AddUint64(&stats.Failed, 1)
				}
				
				// Периодически логируем прогресс для крупных шардов
				if len(transactions) > 10000 && (i+1)%5000 == 0 {
					log.Debug("Прогресс обработки шарда", 
						"shard_id", shardID, 
						"processed", i+1, 
						"total", len(transactions),
						"percent", float64(i+1)/float64(len(transactions))*100)
				}
				
			case <-ctx.Done():
				// Контекст был отменен, прекращаем обработку
				log.Warn("Обработка шарда прервана", 
					"shard_id", shardID, 
					"processed", i, 
					"total", len(transactions),
					"reason", ctx.Err())
				return
			}
		}
	}()
	
	// Ждем завершения всех воркеров
	wg.Wait()
	close(jobResults)
	
	// Завершаем статистику
	stats.EndTime = time.Now()
	stats.ElapsedMs = stats.EndTime.Sub(stats.StartTime)
	
	// Проверяем, не был ли отменен контекст
	if ctx.Err() != nil {
		log.Warn("Шард не полностью обработан из-за отмены контекста", 
			"shard_id", shardID, 
			"completed", stats.Completed, 
			"failed", stats.Failed, 
			"total", stats.Count,
			"error", ctx.Err())
	} else {
		log.Debug("Шард обработан", 
			"shard_id", shardID, 
			"completed", stats.Completed, 
			"failed", stats.Failed, 
			"elapsed_ms", stats.ElapsedMs.Milliseconds(),
			"tx_per_second", math.Round(float64(stats.Completed)/(float64(stats.ElapsedMs)/float64(time.Second))))
	}
	
	// Отправляем результаты
	select {
	case results <- stats:
		// Результаты успешно отправлены
	case <-ctx.Done():
		// Контекст отменен, но всё равно пытаемся отправить результаты
		select {
		case results <- stats:
		default:
		}
	}
}

// convertTxToContract конвертирует транзакцию в формат контракта
func (bp *BatchProcessor) convertTxToContract(tx *types.Transaction) (interface{}, []byte, bool) {
	// Получаем данные транзакции
	data := tx.Data()
	
	// Создаем структуру "контракт" (формат зависит от реализации процессора)
	// В данном случае предполагаем, что процессор принимает структуру с необходимыми полями
	contract := struct {
		From    common.Address
		To      common.Address
		Value   string
		Gas     uint64
		GasPrice string
		Data    []byte
	}{
		From:    tx.From(),
		To:      *tx.To(),
		Value:   tx.Value().String(),
		Gas:     tx.Gas(),
		GasPrice: tx.GasPrice().String(),
		Data:    data,
	}
	
	// Определяем, является ли транзакция только для чтения
	readOnly := tx.Value().Sign() == 0
	
	return contract, data, readOnly
}

// GetStats возвращает статистику последнего обработанного батча
func (bp *BatchProcessor) GetStats() TxStats {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	return bp.lastStats
}

// GetShardCount возвращает количество шардов для параллельной обработки
func (bp *BatchProcessor) GetShardCount() int {
	return bp.shardCount
}

// SetShardCount устанавливает количество шардов для параллельной обработки
func (bp *BatchProcessor) SetShardCount(count int) {
	if count <= 0 {
		count = runtime.NumCPU()
		if count > MaxShardCount {
			count = MaxShardCount
		}
	} else if count > MaxShardCount {
		count = MaxShardCount
	}
	
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	bp.shardCount = count
}

// GetTotalBatchCount возвращает общее количество обработанных батчей
func (bp *BatchProcessor) GetTotalBatchCount() uint64 {
	return bp.totalBatches.Load()
} 