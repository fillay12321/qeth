// Package quantum обеспечивает интеграцию квантовых вычислений с EVM
package quantum

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

// QuestContractInterface обеспечивает интерфейс для взаимодействия с квантовым процессором
// из смарт-контрактов Ethereum
type QuestContractInterface struct {
	// Процессор квантовых вычислений
	processor *QuestProcessor

	// Текущий EVM
	evm *vm.EVM

	// Текущий контракт
	contract *vm.Contract
}

// NewQuestContractInterface создает новый интерфейс для взаимодействия с квантовым процессором
func NewQuestContractInterface(processor *QuestProcessor, evm *vm.EVM, contract *vm.Contract) *QuestContractInterface {
	return &QuestContractInterface{
		processor: processor,
		evm:       evm,
		contract:  contract,
	}
}

// InitializeQuantumState инициализирует квантовое состояние с указанным количеством кубитов
func (qci *QuestContractInterface) InitializeQuantumState(numQubits uint64) error {
	if qci.processor == nil {
		return errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return errors.New("квантовый процессор не активирован")
	}

	// Проверяем, что количество кубитов не превышает максимально допустимое
	if int(numQubits) > qci.processor.GetMaxQubits() {
		return errors.New("превышено максимальное количество кубитов")
	}

	// Получаем реестр опкодов
	registry := qci.processor.registry
	if registry == nil {
		return errors.New("реестр опкодов не инициализирован")
	}

	// Получаем контекст QEVM
	ctx := registry.GetContext()
	if ctx == nil {
		return errors.New("контекст QEVM не активирован")
	}

	// Инициализируем квантовое состояние
	err := ctx.InitializeState(numQubits)
	if err != nil {
		return err
	}

	// Привязываем состояние к адресу контракта
	ctx.SetContractAddress(qci.contract.Address())

	log.Info("Инициализировано квантовое состояние", 
		"contract", qci.contract.Address(),
		"numQubits", numQubits)

	return nil
}

// ApplyGate применяет квантовый вентиль к указанному кубиту
func (qci *QuestContractInterface) ApplyGate(gateType uint8, qubitIndex uint64) error {
	if qci.processor == nil {
		return errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return errors.New("квантовый процессор не активирован")
	}

	// Получаем реестр опкодов
	registry := qci.processor.registry
	if registry == nil {
		return errors.New("реестр опкодов не инициализирован")
	}

	// Получаем контекст QEVM
	ctx := registry.GetContext()
	if ctx == nil {
		return errors.New("контекст QEVM не активирован")
	}

	// Проверяем, что состояние инициализировано
	if !ctx.IsStateInitialized() {
		return errors.New("квантовое состояние не инициализировано")
	}

	// Проверяем, что индекс кубита не превышает количество инициализированных кубитов
	if qubitIndex >= ctx.GetNumQubits() {
		return errors.New("индекс кубита превышает количество инициализированных кубитов")
	}

	// Проверяем, что тип вентиля допустим
	if !isValidGateType(gateType) {
		return errors.New("недопустимый тип квантового вентиля")
	}

	// Применяем вентиль к кубиту
	err := ctx.ApplyGate(gateType, qubitIndex)
	if err != nil {
		return err
	}

	return nil
}

// ApplyControlledGate применяет управляемый квантовый вентиль к указанным кубитам
func (qci *QuestContractInterface) ApplyControlledGate(gateType uint8, controlQubit, targetQubit uint64) error {
	if qci.processor == nil {
		return errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return errors.New("квантовый процессор не активирован")
	}

	// Получаем реестр опкодов
	registry := qci.processor.registry
	if registry == nil {
		return errors.New("реестр опкодов не инициализирован")
	}

	// Получаем контекст QEVM
	ctx := registry.GetContext()
	if ctx == nil {
		return errors.New("контекст QEVM не активирован")
	}

	// Проверяем, что состояние инициализировано
	if !ctx.IsStateInitialized() {
		return errors.New("квантовое состояние не инициализировано")
	}

	// Проверяем, что индексы кубитов не превышают количество инициализированных кубитов
	if controlQubit >= ctx.GetNumQubits() || targetQubit >= ctx.GetNumQubits() {
		return errors.New("индекс кубита превышает количество инициализированных кубитов")
	}

	// Проверяем, что кубиты различны
	if controlQubit == targetQubit {
		return errors.New("управляющий и целевой кубиты должны быть различны")
	}

	// Проверяем, что тип вентиля допустим
	if !isValidGateType(gateType) {
		return errors.New("недопустимый тип квантового вентиля")
	}

	// Применяем управляемый вентиль
	err := ctx.ApplyControlledGate(gateType, controlQubit, targetQubit)
	if err != nil {
		return err
	}

	return nil
}

// MeasureQubit измеряет состояние указанного кубита
func (qci *QuestContractInterface) MeasureQubit(qubitIndex uint64) (uint8, error) {
	if qci.processor == nil {
		return 0, errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return 0, errors.New("квантовый процессор не активирован")
	}

	// Получаем реестр опкодов
	registry := qci.processor.registry
	if registry == nil {
		return 0, errors.New("реестр опкодов не инициализирован")
	}

	// Получаем контекст QEVM
	ctx := registry.GetContext()
	if ctx == nil {
		return 0, errors.New("контекст QEVM не активирован")
	}

	// Проверяем, что состояние инициализировано
	if !ctx.IsStateInitialized() {
		return 0, errors.New("квантовое состояние не инициализировано")
	}

	// Проверяем, что индекс кубита не превышает количество инициализированных кубитов
	if qubitIndex >= ctx.GetNumQubits() {
		return 0, errors.New("индекс кубита превышает количество инициализированных кубитов")
	}

	// Измеряем кубит
	result, err := ctx.MeasureQubit(qubitIndex)
	if err != nil {
		return 0, err
	}

	return result, nil
}

// MeasureAllQubits измеряет состояние всех кубитов
func (qci *QuestContractInterface) MeasureAllQubits() ([]uint8, error) {
	if qci.processor == nil {
		return nil, errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return nil, errors.New("квантовый процессор не активирован")
	}

	// Получаем реестр опкодов
	registry := qci.processor.registry
	if registry == nil {
		return nil, errors.New("реестр опкодов не инициализирован")
	}

	// Получаем контекст QEVM
	ctx := registry.GetContext()
	if ctx == nil {
		return nil, errors.New("контекст QEVM не активирован")
	}

	// Проверяем, что состояние инициализировано
	if !ctx.IsStateInitialized() {
		return nil, errors.New("квантовое состояние не инициализировано")
	}

	// Измеряем все кубиты
	results, err := ctx.MeasureAllQubits()
	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetStateProbabilities возвращает вероятности состояний
func (qci *QuestContractInterface) GetStateProbabilities() (map[uint64]float64, error) {
	if qci.processor == nil {
		return nil, errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return nil, errors.New("квантовый процессор не активирован")
	}

	// Получаем реестр опкодов
	registry := qci.processor.registry
	if registry == nil {
		return nil, errors.New("реестр опкодов не инициализирован")
	}

	// Получаем контекст QEVM
	ctx := registry.GetContext()
	if ctx == nil {
		return nil, errors.New("контекст QEVM не активирован")
	}

	// Проверяем, что состояние инициализировано
	if !ctx.IsStateInitialized() {
		return nil, errors.New("квантовое состояние не инициализировано")
	}

	// Получаем вероятности состояний
	probabilities, err := ctx.GetStateProbabilities()
	if err != nil {
		return nil, err
	}

	return probabilities, nil
}

// ResetQuantumState сбрасывает квантовое состояние
func (qci *QuestContractInterface) ResetQuantumState() error {
	if qci.processor == nil {
		return errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return errors.New("квантовый процессор не активирован")
	}

	// Получаем реестр опкодов
	registry := qci.processor.registry
	if registry == nil {
		return errors.New("реестр опкодов не инициализирован")
	}

	// Получаем контекст QEVM
	ctx := registry.GetContext()
	if ctx == nil {
		return errors.New("контекст QEVM не активирован")
	}

	// Проверяем, что состояние инициализировано
	if !ctx.IsStateInitialized() {
		return errors.New("квантовое состояние не инициализировано")
	}

	// Сбрасываем состояние
	err := ctx.ResetState()
	if err != nil {
		return err
	}

	log.Info("Сброшено квантовое состояние", "contract", qci.contract.Address())

	return nil
}

// DestroyQuantumState уничтожает квантовое состояние
func (qci *QuestContractInterface) DestroyQuantumState() error {
	if qci.processor == nil {
		return errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return errors.New("квантовый процессор не активирован")
	}

	// Получаем реестр опкодов
	registry := qci.processor.registry
	if registry == nil {
		return errors.New("реестр опкодов не инициализирован")
	}

	// Получаем контекст QEVM
	ctx := registry.GetContext()
	if ctx == nil {
		return errors.New("контекст QEVM не активирован")
	}

	// Проверяем, что состояние инициализировано
	if !ctx.IsStateInitialized() {
		return errors.New("квантовое состояние не инициализировано")
	}

	// Уничтожаем состояние
	err := ctx.DestroyState()
	if err != nil {
		return err
	}

	log.Info("Уничтожено квантовое состояние", "contract", qci.contract.Address())

	return nil
}

// ExecuteShor выполняет алгоритм Шора для факторизации числа
func (qci *QuestContractInterface) ExecuteShor(n *big.Int) (*big.Int, *big.Int, error) {
	if qci.processor == nil {
		return nil, nil, errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return nil, nil, errors.New("квантовый процессор не активирован")
	}

	// Выполняем алгоритм Шора
	p, q, err := qci.processor.ExecuteShor(n)
	if err != nil {
		return nil, nil, err
	}

	// Логируем результат
	log.Info("Выполнен алгоритм Шора", 
		"contract", qci.contract.Address(),
		"n", n,
		"p", p,
		"q", q)

	return p, q, nil
}

// ExecuteGrover выполняет алгоритм Гровера для поиска элемента в неупорядоченном списке
func (qci *QuestContractInterface) ExecuteGrover(searchSpace uint64, targetValue []byte) (uint64, error) {
	if qci.processor == nil {
		return 0, errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return 0, errors.New("квантовый процессор не активирован")
	}

	// Выполняем алгоритм Гровера
	result, err := qci.processor.ExecuteGrover(searchSpace, targetValue)
	if err != nil {
		return 0, err
	}

	// Логируем результат
	log.Info("Выполнен алгоритм Гровера", 
		"contract", qci.contract.Address(),
		"searchSpace", searchSpace,
		"result", result)

	return result, nil
}

// GenerateQuantumRandomNumber генерирует квантовое случайное число
func (qci *QuestContractInterface) GenerateQuantumRandomNumber(min, max uint64) (uint64, error) {
	if qci.processor == nil {
		return 0, errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return 0, errors.New("квантовый процессор не активирован")
	}

	// Генерируем квантовое случайное число
	result, err := qci.processor.GenerateQuantumRandomNumber(min, max)
	if err != nil {
		return 0, err
	}

	// Логируем результат
	log.Debug("Сгенерировано квантовое случайное число", 
		"contract", qci.contract.Address(),
		"min", min,
		"max", max,
		"result", result)

	return result, nil
}

// ExecuteQuantumFourierTransform выполняет квантовое преобразование Фурье
func (qci *QuestContractInterface) ExecuteQuantumFourierTransform(data []complex128) ([]complex128, error) {
	if qci.processor == nil {
		return nil, errors.New("квантовый процессор не инициализирован")
	}

	if !qci.processor.IsEnabled() {
		return nil, errors.New("квантовый процессор не активирован")
	}

	// Выполняем квантовое преобразование Фурье
	result, err := qci.processor.ExecuteQuantumFourierTransform(data)
	if err != nil {
		return nil, err
	}

	// Логируем результат
	log.Debug("Выполнено квантовое преобразование Фурье", 
		"contract", qci.contract.Address(),
		"dataLength", len(data))

	return result, nil
}

// GetQuestStats возвращает статистику использования квантового процессора
func (qci *QuestContractInterface) GetQuestStats() QuestStats {
	if qci.processor == nil {
		return QuestStats{}
	}

	return qci.processor.GetStats()
}

// isValidGateType проверяет, что тип вентиля допустим
func isValidGateType(gateType uint8) bool {
	validGates := map[uint8]bool{
		0: true, // Identity
		1: true, // Hadamard
		2: true, // PauliX (NOT)
		3: true, // PauliY
		4: true, // PauliZ
		5: true, // S
		6: true, // T
		7: true, // CNOT
		8: true, // SWAP
		9: true, // Toffoli
	}

	return validGates[gateType]
} 