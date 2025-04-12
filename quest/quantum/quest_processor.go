// Package quantum обеспечивает интеграцию квантовых вычислений с EVM
package quantum

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

// QuestProcessor представляет квантовый процессор для EVM
type QuestProcessor struct {
	// Реестр квантовых опкодов
	registry *QuestOpcodeRegistry

	// Квантовое окружение
	env *QuestEnv

	// Максимальное количество кубитов
	maxQubits int

	// Флаг использования GPU
	useGPU bool

	// ID устройства GPU
	gpuDeviceID int

	// Флаг, указывающий, включен ли процессор
	enabled bool

	// Мьютекс для синхронизации доступа
	mutex sync.Mutex
}

// NewQuestProcessor создает новый квантовый процессор
func NewQuestProcessor(maxQubits int, useGPU bool, gpuDeviceID int) (*QuestProcessor, error) {
	if maxQubits <= 0 {
		maxQubits = 5 // Значение по умолчанию
	}

	// Создаем реестр опкодов
	registry := NewQuestOpcodeRegistry(maxQubits, useGPU, gpuDeviceID)

	return &QuestProcessor{
		registry:    registry,
		maxQubits:   maxQubits,
		useGPU:      useGPU,
		gpuDeviceID: gpuDeviceID,
		enabled:     false,
	}, nil
}

// Initialize инициализирует квантовый процессор
func (p *QuestProcessor) Initialize(evm *vm.EVM) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if evm == nil {
		return errors.New("EVM не может быть nil")
	}

	// Активируем квантовые опкоды
	err := p.registry.Enable(evm, p.maxQubits, p.useGPU, p.gpuDeviceID)
	if err != nil {
		return err
	}

	p.enabled = true
	log.Info("Квантовый процессор инициализирован", "qubits", p.maxQubits, "gpu", p.useGPU)
	return nil
}

// IsEnabled возвращает состояние активации процессора
func (p *QuestProcessor) IsEnabled() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.enabled
}

// GetMaxQubits возвращает максимальное количество кубитов
func (p *QuestProcessor) GetMaxQubits() int {
	return p.maxQubits
}

// GetContext возвращает контекст QEVM
func (p *QuestProcessor) GetContext() *QEVMContext {
	if p.registry == nil {
		return nil
	}
	return p.registry.GetContext()
}

// Shutdown освобождает ресурсы квантового процессора
func (p *QuestProcessor) Shutdown() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.registry != nil {
		err := p.registry.Disable()
		if err != nil {
			return err
		}
	}

	p.enabled = false
	return nil
} 