// Package quantum обеспечивает интеграцию квантовых вычислений с EVM
package quantum

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
)

// Константы для регистрации квантовых опкодов
const (
	// Диапазон квантовых опкодов
	QUEST_OPCODE_BEGIN = 0xf0
	QUEST_OPCODE_END   = 0xff
	QUEST_ALG_BEGIN    = 0xe0
	QUEST_ALG_END      = 0xef
)

// QuestOpcodeRegistry представляет регистрацию квантовых опкодов в EVM
type QuestOpcodeRegistry struct {
	// Контекст выполнения квантовых операций
	context *QEVMContext

	// Флаг, указывающий, активированы ли квантовые опкоды
	enabled bool
}

// NewQuestOpcodeRegistry создает новый регистр квантовых опкодов
func NewQuestOpcodeRegistry(maxQubits int, useGPU bool, gpuDeviceID int) *QuestOpcodeRegistry {
	return &QuestOpcodeRegistry{
		context: nil,
		enabled: false,
	}
}

// Enable активирует квантовые опкоды в EVM
func (qor *QuestOpcodeRegistry) Enable(evm *vm.EVM, maxQubits int, useGPU bool, gpuDeviceID int) error {
	if qor.enabled {
		return errors.New("квантовые опкоды уже активированы")
	}

	if evm == nil {
		return errors.New("EVM не может быть nil")
	}

	// Создаем контекст для выполнения квантовых операций
	var err error
	qor.context, err = NewQEVMContext(evm, maxQubits, useGPU, gpuDeviceID)
	if err != nil {
		return err
	}

	qor.enabled = true
	return nil
}

// Disable деактивирует квантовые опкоды в EVM
func (qor *QuestOpcodeRegistry) Disable() error {
	if !qor.enabled {
		return nil
	}

	if qor.context != nil {
		err := qor.context.Destroy()
		if err != nil {
			return err
		}
		qor.context = nil
	}

	qor.enabled = false
	return nil
}

// IsEnabled проверяет, активированы ли квантовые опкоды
func (qor *QuestOpcodeRegistry) IsEnabled() bool {
	return qor.enabled
}

// GetContext возвращает контекст для выполнения квантовых операций
func (qor *QuestOpcodeRegistry) GetContext() *QEVMContext {
	return qor.context
}

// RegisterOpcodes регистрирует квантовые опкоды в JumpTable EVM
func (qor *QuestOpcodeRegistry) RegisterOpcodes(jumpTable *vm.JumpTable) error {
	if !qor.enabled || qor.context == nil {
		return errors.New("квантовые опкоды не активированы")
	}

	// Регистрируем базовые квантовые операции
	err := qor.registerQInit(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQDestroy(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQReset(jumpTable)
	if err != nil {
		return err
	}

	// Регистрируем квантовые вентили
	err = qor.registerQHadamard(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQPauliX(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQPauliY(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQPauliZ(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQPhase(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQRotX(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQRotY(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQRotZ(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQCNOT(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQSwap(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQToffoli(jumpTable)
	if err != nil {
		return err
	}

	// Регистрируем квантовые измерения
	err = qor.registerQMeasure(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQMeasureAll(jumpTable)
	if err != nil {
		return err
	}

	// Регистрируем квантовые алгоритмы
	err = qor.registerQShor(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQGrover(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQQFT(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQQPE(jumpTable)
	if err != nil {
		return err
	}

	err = qor.registerQRandom(jumpTable)
	if err != nil {
		return err
	}

	return nil
}

// Методы для регистрации конкретных опкодов

// registerQInit регистрирует опкод QINIT
func (qor *QuestOpcodeRegistry) registerQInit(jumpTable *vm.JumpTable) error {
	opcode := byte(QINIT)
	gas, err := qor.context.GasForOp(QINIT)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QINIT),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    1,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QINIT",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQDestroy регистрирует опкод QDESTROY
func (qor *QuestOpcodeRegistry) registerQDestroy(jumpTable *vm.JumpTable) error {
	opcode := byte(QDESTROY)
	gas, err := qor.context.GasForOp(QDESTROY)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QDESTROY),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    0,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QDESTROY",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQReset регистрирует опкод QRESET
func (qor *QuestOpcodeRegistry) registerQReset(jumpTable *vm.JumpTable) error {
	opcode := byte(QRESET)
	gas, err := qor.context.GasForOp(QRESET)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QRESET),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    0,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QRESET",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQHadamard регистрирует опкод QHADAMARD
func (qor *QuestOpcodeRegistry) registerQHadamard(jumpTable *vm.JumpTable) error {
	opcode := byte(QHADAMARD)
	gas, err := qor.context.GasForOp(QHADAMARD)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QHADAMARD),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    1,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QHADAMARD",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQPauliX регистрирует опкод QPAULIX
func (qor *QuestOpcodeRegistry) registerQPauliX(jumpTable *vm.JumpTable) error {
	opcode := byte(QPAULIX)
	gas, err := qor.context.GasForOp(QPAULIX)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QPAULIX),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    1,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QPAULIX",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQPauliY регистрирует опкод QPAULIY
func (qor *QuestOpcodeRegistry) registerQPauliY(jumpTable *vm.JumpTable) error {
	opcode := byte(QPAULIY)
	gas, err := qor.context.GasForOp(QPAULIY)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QPAULIY),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    1,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QPAULIY",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQPauliZ регистрирует опкод QPAULIZ
func (qor *QuestOpcodeRegistry) registerQPauliZ(jumpTable *vm.JumpTable) error {
	opcode := byte(QPAULIZ)
	gas, err := qor.context.GasForOp(QPAULIZ)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QPAULIZ),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    1,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QPAULIZ",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQPhase регистрирует опкод QPHASE
func (qor *QuestOpcodeRegistry) registerQPhase(jumpTable *vm.JumpTable) error {
	opcode := byte(QPHASE)
	gas, err := qor.context.GasForOp(QPHASE)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QPHASE),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    2,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QPHASE",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQRotX регистрирует опкод QROTX
func (qor *QuestOpcodeRegistry) registerQRotX(jumpTable *vm.JumpTable) error {
	opcode := byte(QROTX)
	gas, err := qor.context.GasForOp(QROTX)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QROTX),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    2,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QROTX",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQRotY регистрирует опкод QROTY
func (qor *QuestOpcodeRegistry) registerQRotY(jumpTable *vm.JumpTable) error {
	opcode := byte(QROTY)
	gas, err := qor.context.GasForOp(QROTY)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QROTY),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    2,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QROTY",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQRotZ регистрирует опкод QROTZ
func (qor *QuestOpcodeRegistry) registerQRotZ(jumpTable *vm.JumpTable) error {
	opcode := byte(QROTZ)
	gas, err := qor.context.GasForOp(QROTZ)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QROTZ),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    2,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QROTZ",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQCNOT регистрирует опкод QCNOT
func (qor *QuestOpcodeRegistry) registerQCNOT(jumpTable *vm.JumpTable) error {
	opcode := byte(QCNOT)
	gas, err := qor.context.GasForOp(QCNOT)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QCNOT),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    2,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QCNOT",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQSwap регистрирует опкод QSWAP
func (qor *QuestOpcodeRegistry) registerQSwap(jumpTable *vm.JumpTable) error {
	opcode := byte(QSWAP)
	gas, err := qor.context.GasForOp(QSWAP)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QSWAP),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    2,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QSWAP",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQToffoli регистрирует опкод QTOFFOLI
func (qor *QuestOpcodeRegistry) registerQToffoli(jumpTable *vm.JumpTable) error {
	opcode := byte(QTOFFOLI)
	gas, err := qor.context.GasForOp(QTOFFOLI)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QTOFFOLI),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    3,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QTOFFOLI",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQMeasure регистрирует опкод QMEASURE
func (qor *QuestOpcodeRegistry) registerQMeasure(jumpTable *vm.JumpTable) error {
	opcode := byte(QMEASURE)
	gas, err := qor.context.GasForOp(QMEASURE)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QMEASURE),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    1,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QMEASURE",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQMeasureAll регистрирует опкод QMEASUREALL
func (qor *QuestOpcodeRegistry) registerQMeasureAll(jumpTable *vm.JumpTable) error {
	opcode := byte(QMEASUREALL)
	gas, err := qor.context.GasForOp(QMEASUREALL)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QMEASUREALL),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    0,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QMEASUREALL",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQShor регистрирует опкод QSHOR
func (qor *QuestOpcodeRegistry) registerQShor(jumpTable *vm.JumpTable) error {
	opcode := byte(QSHOR)
	gas, err := qor.context.GasForOp(QSHOR)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QSHOR),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    1,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QSHOR",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQGrover регистрирует опкод QGROVER
func (qor *QuestOpcodeRegistry) registerQGrover(jumpTable *vm.JumpTable) error {
	opcode := byte(QGROVER)
	gas, err := qor.context.GasForOp(QGROVER)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QGROVER),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    3,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QGROVER",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  memoryQGrover, // Особая функция для расчета использования памяти
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// memoryQGrover рассчитывает использование памяти для алгоритма Гровера
func memoryQGrover(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 {
	if stack.Len() < 3 {
		return 0
	}
	
	searchSpace := stack.Back(0).Uint64()
	targetLen := stack.Back(1).Uint64()
	targetOffset := stack.Back(2).Uint64()
	
	// Рассчитываем объем памяти, который будет использоваться
	memorySize := targetOffset + targetLen
	
	// Округляем до ближайшего кратного 32
	return (memorySize + 31) / 32 * 32
}

// registerQQFT регистрирует опкод QQFT
func (qor *QuestOpcodeRegistry) registerQQFT(jumpTable *vm.JumpTable) error {
	opcode := byte(QQFT)
	gas, err := qor.context.GasForOp(QQFT)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QQFT),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    2,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QQFT",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  memoryQQFT, // Особая функция для расчета использования памяти
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// memoryQQFT рассчитывает использование памяти для квантового преобразования Фурье
func memoryQQFT(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 {
	if stack.Len() < 2 {
		return 0
	}
	
	dataLen := stack.Back(0).Uint64()
	dataOffset := stack.Back(1).Uint64()
	
	// Рассчитываем объем памяти, который будет использоваться (16 байт на одно комплексное число)
	memorySize := dataOffset + dataLen*16
	
	// Округляем до ближайшего кратного 32
	return (memorySize + 31) / 32 * 32
}

// registerQQPE регистрирует опкод QQPE
func (qor *QuestOpcodeRegistry) registerQQPE(jumpTable *vm.JumpTable) error {
	opcode := byte(QQPE)
	gas, err := qor.context.GasForOp(QQPE)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QQPE),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    3,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QQPE",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  func(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 { return 0 },
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// registerQRandom регистрирует опкод QRANDOM
func (qor *QuestOpcodeRegistry) registerQRandom(jumpTable *vm.JumpTable) error {
	opcode := byte(QRANDOM)
	gas, err := qor.context.GasForOp(QRANDOM)
	if err != nil {
		return err
	}

	// Создаем операцию для опкода
	operation := &vm.Operation{
		Execute:     qor.createQuestOperationFunc(QRANDOM),
		Gas:         gas,
		Const:       true, // Константная стоимость газа
		MinStack:    2,    // Минимальный размер стека для операции
		MaxStack:    1024, // Максимальный размер стека
		Name:        "QRANDOM",
		IsPush:      false,
		IsJump:      false,
		OpCode:      opcode,
		MemorySize:  memoryQRandom, // Особая функция для расчета использования памяти
	}

	// Добавляем операцию в JumpTable
	jumpTable[opcode] = operation

	return nil
}

// memoryQRandom рассчитывает использование памяти для генерации квантовых случайных чисел
func memoryQRandom(pc *uint64, stack *vm.Stack, mem *vm.Memory, contract *vm.Contract) uint64 {
	if stack.Len() < 2 {
		return 0
	}
	
	length := stack.Back(0).Uint64()
	offset := stack.Back(1).Uint64()
	
	// Рассчитываем объем памяти, который будет использоваться
	memorySize := offset + length
	
	// Округляем до ближайшего кратного 32
	return (memorySize + 31) / 32 * 32
}

// createQuestOperationFunc создает функцию для выполнения квантовой операции
func (qor *QuestOpcodeRegistry) createQuestOperationFunc(opcode OpCode) vm.ExecutionFunc {
	return func(pc *uint64, interpreter *vm.EVMInterpreter, scope *vm.ScopeContext) ([]byte, error) {
		if !qor.enabled || qor.context == nil {
			return nil, errors.New("квантовые опкоды не активированы")
		}
		
		// Выполняем операцию через контекст QEVM
		err := qor.context.ExecuteOp(opcode, scope.Stack, scope.Memory, interpreter)
		if err != nil {
			return nil, fmt.Errorf("ошибка выполнения квантовой операции %s: %w", OpCodeToString(opcode), err)
		}
		
		*pc++
		return nil, nil
	}
}

// IsQuestOpCode проверяет, является ли опкод квантовым
func IsQuestOpCode(opcode byte) bool {
	return (opcode >= QUEST_OPCODE_BEGIN && opcode <= QUEST_OPCODE_END) ||
		(opcode >= QUEST_ALG_BEGIN && opcode <= QUEST_ALG_END)
}

// OpCodeToString преобразует квантовый опкод в строку
func OpCodeToString(opcode OpCode) string {
	switch opcode {
	case QINIT:
		return "QINIT"
	case QDESTROY:
		return "QDESTROY"
	case QRESET:
		return "QRESET"
	case QHADAMARD:
		return "QHADAMARD"
	case QPAULIX:
		return "QPAULIX"
	case QPAULIY:
		return "QPAULIY"
	case QPAULIZ:
		return "QPAULIZ"
	case QPHASE:
		return "QPHASE"
	case QROTX:
		return "QROTX"
	case QROTY:
		return "QROTY"
	case QROTZ:
		return "QROTZ"
	case QCNOT:
		return "QCNOT"
	case QSWAP:
		return "QSWAP"
	case QTOFFOLI:
		return "QTOFFOLI"
	case QMEASURE:
		return "QMEASURE"
	case QMEASUREALL:
		return "QMEASUREALL"
	case QSHOR:
		return "QSHOR"
	case QGROVER:
		return "QGROVER"
	case QQFT:
		return "QQFT"
	case QQPE:
		return "QQPE"
	case QRANDOM:
		return "QRANDOM"
	default:
		return fmt.Sprintf("UNKNOWN_QOPCODE(%d)", opcode)
	}
} 