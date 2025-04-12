// Copyright 2014 The go-ethereum Authors
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
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
)

// Config are the configuration options for the Interpreter
type Config struct {
	Tracer                  *tracing.Hooks
	NoBaseFee               bool  // Forces the EIP-1559 baseFee to 0 (needed for 0 price calls)
	EnablePreimageRecording bool  // Enables recording of SHA3/keccak preimages
	ExtraEips               []int // Additional EIPS that are to be enabled

	StatelessSelfValidation bool // Generate execution witnesses and self-check against them (testing purpose)
	
	EnableParallelExecution bool // Включает параллельное выполнение операций
	ParallelThreads        int  // Количество потоков для параллельного выполнения (0 = автоматически)
}

// ScopeContext contains the things that are per-call, such as stack and memory,
// but not transients like pc and gas
type ScopeContext struct {
	Memory   *Memory
	Stack    *Stack
	Contract *Contract
	
	// Для защиты контекста в параллельном режиме
	mutex sync.RWMutex
}

// MemoryData returns the underlying memory slice. Callers must not modify the contents
// of the returned data.
func (ctx *ScopeContext) MemoryData() []byte {
	if ctx.Memory == nil {
		return nil
	}
	return ctx.Memory.Data()
}

// StackData returns the stack data. Callers must not modify the contents
// of the returned data.
func (ctx *ScopeContext) StackData() []uint256.Int {
	if ctx.Stack == nil {
		return nil
	}
	return ctx.Stack.Data()
}

// Caller returns the current caller.
func (ctx *ScopeContext) Caller() common.Address {
	return ctx.Contract.Caller()
}

// Address returns the address where this scope of execution is taking place.
func (ctx *ScopeContext) Address() common.Address {
	return ctx.Contract.Address()
}

// CallValue returns the value supplied with this call.
func (ctx *ScopeContext) CallValue() *uint256.Int {
	return ctx.Contract.Value()
}

// CallInput returns the input/calldata with this call. Callers must not modify
// the contents of the returned data.
func (ctx *ScopeContext) CallInput() []byte {
	return ctx.Contract.Input
}

// ContractCode returns the code of the contract being executed.
func (ctx *ScopeContext) ContractCode() []byte {
	return ctx.Contract.Code
}

// EVMInterpreter represents an EVM interpreter
type EVMInterpreter struct {
	evm   *EVM
	table *JumpTable

	hasher    crypto.KeccakState // Keccak256 hasher instance shared across opcodes
	hasherBuf common.Hash        // Keccak256 hasher result array shared across opcodes

	readOnly   bool   // Whether to throw on stateful modifications
	returnData []byte // Last CALL's return data for subsequent reuse
	
	// Для параллельного выполнения
	mutex       sync.Mutex    // Защита критических секций
	isParallel  bool          // Режим параллельного выполнения 
	numThreads  int           // Количество потоков для параллельного выполнения
	abortFlag   atomic.Bool   // Флаг прерывания выполнения
}

// EnableParallel активирует режим параллельного выполнения инструкций
func (in *EVMInterpreter) EnableParallel(threads int) {
	in.mutex.Lock()
	defer in.mutex.Unlock()
	
	in.isParallel = true
	in.numThreads = threads
	
	if threads <= 0 {
		// Автоматическое определение количества потоков
		numCPU := runtime.NumCPU()
		in.numThreads = numCPU
		if numCPU > 4 {
			// Оставляем некоторые ресурсы для системы
			in.numThreads = numCPU - 2
		}
	}
	
	log.Info("Parallel execution enabled", "threads", in.numThreads)
}

// DisableParallel отключает режим параллельного выполнения инструкций
func (in *EVMInterpreter) DisableParallel() {
	in.mutex.Lock()
	defer in.mutex.Unlock()
	
	in.isParallel = false
	log.Info("Parallel execution disabled")
}

// NewEVMInterpreter returns a new instance of the Interpreter.
func NewEVMInterpreter(evm *EVM) *EVMInterpreter {
	// If jump table was not initialised we set the default one.
	var table *JumpTable
	switch {
	case evm.chainRules.IsVerkle:
		// TODO replace with proper instruction set when fork is specified
		table = &verkleInstructionSet
	case evm.chainRules.IsPrague:
		table = &pragueInstructionSet
	case evm.chainRules.IsCancun:
		table = &cancunInstructionSet
	case evm.chainRules.IsShanghai:
		table = &shanghaiInstructionSet
	case evm.chainRules.IsMerge:
		table = &mergeInstructionSet
	case evm.chainRules.IsLondon:
		table = &londonInstructionSet
	case evm.chainRules.IsBerlin:
		table = &berlinInstructionSet
	case evm.chainRules.IsIstanbul:
		table = &istanbulInstructionSet
	case evm.chainRules.IsConstantinople:
		table = &constantinopleInstructionSet
	case evm.chainRules.IsByzantium:
		table = &byzantiumInstructionSet
	case evm.chainRules.IsEIP158:
		table = &spuriousDragonInstructionSet
	case evm.chainRules.IsEIP150:
		table = &tangerineWhistleInstructionSet
	case evm.chainRules.IsHomestead:
		table = &homesteadInstructionSet
	default:
		table = &frontierInstructionSet
	}
	var extraEips []int
	if len(evm.Config.ExtraEips) > 0 {
		// Deep-copy jumptable to prevent modification of opcodes in other tables
		table = copyJumpTable(table)
	}
	for _, eip := range evm.Config.ExtraEips {
		if err := EnableEIP(eip, table); err != nil {
			// Disable it, so caller can check if it's activated or not
			log.Error("EIP activation failed", "eip", eip, "error", err)
		} else {
			extraEips = append(extraEips, eip)
		}
	}
	evm.Config.ExtraEips = extraEips
	
	interpreter := &EVMInterpreter{evm: evm, table: table}
	
	// Активация параллельного режима, если указано в конфигурации
	if evm.Config.EnableParallelExecution {
		interpreter.EnableParallel(evm.Config.ParallelThreads)
	}
	
	return interpreter
}

// Определяет, можно ли безопасно выполнить операцию параллельно
func (in *EVMInterpreter) canRunParallel(op OpCode) bool {
	// Операции, которые не должны выполняться параллельно
	switch op {
	case JUMP, JUMPI, JUMPDEST, CALL, CALLCODE, 
		 DELEGATECALL, STATICCALL, CREATE, CREATE2,
		 RETURN, REVERT, SELFDESTRUCT, STOP:
		return false
	}
	
	// Проверка на операции с зависимостями от состояния
	if op >= SLOAD && op <= SELFDESTRUCT {
		return false // Состояние хранилища не может изменяться параллельно
	}
	
	// По умолчанию считаем, что математические операции можно выполнять параллельно
	return true
}

// findIndependentBlocks ищет блоки независимых инструкций для параллельного выполнения
func (in *EVMInterpreter) findIndependentBlocks(contract *Contract, startPC uint64, maxPC uint64) []struct{start, end uint64} {
	var blocks []struct{start, end uint64}
	currentStart := startPC
	
	for pc := startPC; pc < maxPC; {
		op := contract.GetOp(pc)
		
		// Если операция нарушает последовательность, завершаем текущий блок
		if !in.canRunParallel(op) {
			if pc > currentStart {
				blocks = append(blocks, struct{start, end uint64}{currentStart, pc})
			}
			currentStart = pc + 1
		}
		
		// Увеличиваем счетчик
		pc++
		
		// Учитываем PUSH операции с аргументами
		if op >= PUSH1 && op <= PUSH32 {
			n := int(op - PUSH1 + 1)
			pc += uint64(n)
		}
	}
	
	// Добавляем последний блок, если он существует
	if currentStart < maxPC {
		blocks = append(blocks, struct{start, end uint64}{currentStart, maxPC})
	}
	
	return blocks
}

// Run loops and evaluates the contract's code with the given input data and returns
// the return byte-slice and an error if one occurred.
//
// It's important to note that any errors returned by the interpreter should be
// considered a revert-and-consume-all-gas operation except for
// ErrExecutionReverted which means revert-and-keep-gas-left.
func (in *EVMInterpreter) Run(contract *Contract, input []byte, readOnly bool) (ret []byte, err error) {
	// Increments the call depth which is restricted to 1024
	in.evm.depth++
	defer func() { in.evm.depth-- }()

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This makes also sure that the readOnly flag isn't removed for child calls.
	if readOnly && !in.readOnly {
		in.readOnly = true
		defer func() { in.readOnly = false }()
	}

	// Reset the previous call's return data. It's unimportant to preserve the old buffer
	// as every returning call will return new data anyway.
	in.returnData = nil

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	var (
		op          OpCode        // current opcode
		mem         = NewMemory() // bound memory
		stack       = newstack()  // local stack
		callContext = &ScopeContext{
			Memory:   mem,
			Stack:    stack,
			Contract: contract,
		}
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC
		// to be uint256. Practically much less so feasible.
		pc   = uint64(0) // program counter
		cost uint64
		// copies used by tracer
		pcCopy  uint64 // needed for the deferred EVMLogger
		gasCopy uint64 // for EVMLogger to log gas remaining before execution
		logged  bool   // deferred EVMLogger should ignore already logged steps
		res     []byte // result of the opcode execution function
	)
	// Don't move this deferrred function, it's placed before the capturestate-deferred method,
	// so that it gets executed _after_: the capturestate needs the stacks before
	// they are returned to the pools
	defer func() {
		returnStack(stack)
	}()
	contract.Input = input

	if in.evm.Config.Debug {
		defer func() {
			if err != nil {
				if !logged {
					in.evm.Config.Tracer.CaptureState(pcCopy, op, gasCopy, cost, callContext, in.returnData, in.evm.depth, err)
				} else {
					in.evm.Config.Tracer.CaptureFault(pcCopy, op, gasCopy, cost, callContext, in.evm.depth, err)
				}
			}
		}()
	}

	// Check для параллельного выполнения
	if in.evm.Config.EnableParallelExecution && in.evm.Config.HyperParallelMode && canRunParallel(contract.Code) {
		// Параллельное выполнение
		return in.runParallel(contract, callContext, mem, stack)
	}

	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	for {
		if in.evm.Config.Debug {
			// Capture pre-execution values for tracing.
			logged, pcCopy, gasCopy = false, pc, contract.Gas
		}

		// Get the operation from the jump table and validate the stack to ensure there are
		// enough stack items available to perform the operation.
		op = contract.GetOp(pc)
		operation := in.table[op]
		if operation == nil {
			return nil, &ErrInvalidOpCode{opcode: op}
		}
		// Validate stack
		if sLen := stack.len(); sLen < operation.minStack {
			return nil, &ErrStackUnderflow{stackLen: sLen, required: operation.minStack}
		} else if sLen > operation.maxStack {
			return nil, &ErrStackOverflow{stackLen: sLen, limit: operation.maxStack}
		}
		// If the operation is valid, enforce write restrictions
		if in.readOnly && in.evm.chainRules.IsByzantium {
			// If the interpreter is operating in readonly mode, make sure no
			// state-modifying operation is performed. The 3rd stack item
			// for a call operation is the value. Transferring value from one
			// account to the others means the state is modified and should also
			// return with an error.
			if operation.writes || (op == CALL && stack.Back(2).Sign() != 0) {
				return nil, ErrWriteProtection
			}
		}
		// Static portion of gas
		cost = operation.constantGas // For tracing
		if !contract.UseGas(operation.constantGas) {
			return nil, ErrOutOfGas
		}

		var memorySize uint64
		// calculate the new memory size and expand the memory to fit
		// the operation
		// Memory check needs to be done prior to evaluating the dynamic gas portion,
		// to detect calculation overflows
		if operation.memorySize != nil {
			memSize, overflow := operation.memorySize(stack)
			if overflow {
				return nil, ErrGasUintOverflow
			}
			// memory is expanded in words of 32 bytes. Gas
			// is also calculated in words.
			if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
				return nil, ErrGasUintOverflow
			}
		}
		// Dynamic portion of gas
		// consume the gas and return an error if not enough gas is available.
		// cost is explicitly set so that the capture state defer method can get the proper cost
		if operation.dynamicGas != nil {
			var dynamicCost uint64
			dynamicCost, err = operation.dynamicGas(in.evm, contract, stack, mem, memorySize)
			cost += dynamicCost // total cost, for debug tracing
			if err != nil || !contract.UseGas(dynamicCost) {
				return nil, ErrOutOfGas
			}
		}
		if memorySize > 0 {
			mem.Resize(memorySize)
		}

		if in.evm.Config.Debug {
			in.evm.Config.Tracer.CaptureState(pc, op, gasCopy, cost, callContext, in.returnData, in.evm.depth, err)
			logged = true
		}

		// execute the operation
		res, err = operation.execute(&pc, in, callContext)
		
		// Проверка на возможность использования квантового процессора
		if in.evm.Config.EnableQuest && in.evm.Config.QuestTPSBooster && isQuestCompatible(op) {
			// Попытка ускорить выполнение через квантовый процессор
			qRes, qErr := in.executeWithQuest(op, callContext, &pc)
			if qErr == nil {
				// Успешное выполнение квантовой операции
				res, err = qRes, nil
			}
		}

		// если операция возвращает что-то, мы сохраняем это в виде "результата выполнения EVM"
		// только возвращенные данные из внутренних вызовов сохраняются в поле returnData
		if operation.returns {
			in.returnData = res
		}

		switch {
		case err != nil:
			return nil, err
		case operation.reverts:
			return res, ErrExecutionReverted
		case operation.halts:
			return res, nil
		case !operation.jumps:
			pc++
		}
	}
}

// executeWithQuest выполняет операцию с использованием квантового процессора Quest
func (in *EVMInterpreter) executeWithQuest(op OpCode, callContext *ScopeContext, pc *uint64) ([]byte, error) {
	// Используем квантовый процессор для ускорения выполнения операции
	if in.evm.questProcessor == nil {
		return nil, fmt.Errorf("квантовый процессор не инициализирован")
	}
	
	// Подготовка входных данных для квантового процессора
	input := make([]byte, 0)
	input = append(input, byte(op))
	
	// Добавляем данные стека если они есть
	if callContext.Stack.len() > 0 {
		stackData := callContext.StackData()
		for i := 0; i < len(stackData) && i < 10; i++ {
			bytes := stackData[i].Bytes32()
			input = append(input, bytes[:]...)
		}
	}
	
	// Выполняем операцию через квантовый процессор
	result, err := in.evm.questProcessor.Run(callContext.Contract, input, in.readOnly)
	if err != nil {
		return nil, err
	}
	
	// Увеличиваем счетчик инструкций
	*pc++
	
	return result, nil
}

// runParallel выполняет контракт в параллельном режиме
func (in *EVMInterpreter) runParallel(contract *Contract, callContext *ScopeContext, mem *Memory, stack *Stack) ([]byte, error) {
	// Получаем код контракта
	code := contract.Code
	
	// Анализируем код и разбиваем его на независимые блоки
	blocks := analyzeCodeBlocks(code)
	
	// Если блоков слишком мало, используем обычное последовательное выполнение
	if len(blocks) < 2 {
		return in.Run(contract, contract.Input, in.readOnly)
	}
	
	// Определяем количество потоков
	numThreads := in.numThreads
	if numThreads <= 0 || numThreads > len(blocks) {
		numThreads = min(runtime.NumCPU(), len(blocks))
	}
	
	// Создаем WaitGroup для синхронизации горутин
	var wg sync.WaitGroup
	
	// Результаты выполнения блоков
	results := make([][]byte, len(blocks))
	errors := make([]error, len(blocks))
	
	// Канал для распределения задач
	blockCh := make(chan int, len(blocks))
	for i := range blocks {
		blockCh <- i
	}
	close(blockCh)
	
	// Запускаем воркеры
	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for blockIndex := range blockCh {
				block := blocks[blockIndex]
				
				// Создаем локальный контекст для каждого блока
				localMem := NewMemory()
				localMem.Set(0, uint64(len(mem.Data())), mem.Data())
				
				localStack := newstack()
				for j := 0; j < stack.len(); j++ {
					localStack.Push(stack.data[j])
				}
				
				localContract := &Contract{
					CallerAddress:   contract.CallerAddress,
					caller:          contract.caller,
					self:            contract.self,
					Gas:             contract.Gas / uint64(len(blocks)),
					Code:            block.code,
					Input:           contract.Input,
					value:           contract.value,
				}
				
				localContext := &ScopeContext{
					Memory:   localMem,
					Stack:    localStack,
					Contract: localContract,
				}
				
				// Выполняем блок кода
				localPC := block.startPC
				for localPC < block.endPC {
					op := localContract.GetOp(localPC)
					operation := in.table[op]
					if operation == nil {
						errors[blockIndex] = &ErrInvalidOpCode{opcode: op}
						break
					}
					
					// Проверяем стек
					if sLen := localStack.len(); sLen < operation.minStack {
						errors[blockIndex] = &ErrStackUnderflow{stackLen: sLen, required: operation.minStack}
						break
					} else if sLen > operation.maxStack {
						errors[blockIndex] = &ErrStackOverflow{stackLen: sLen, limit: operation.maxStack}
						break
					}
					
					// Выполняем операцию
					res, err := operation.execute(&localPC, in, localContext)
					if err != nil {
						errors[blockIndex] = err
						break
					}
					
					// Сохраняем результат
					if operation.returns {
						results[blockIndex] = res
					}
					
					// Переходим к следующей инструкции
					if !operation.jumps {
						localPC++
					}
				}
			}
		}()
	}
	
	// Ждем завершения всех горутин
	wg.Wait()
	
	// Обрабатываем результаты и ошибки
	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}
	
	// Объединяем результаты
	var result []byte
	for _, res := range results {
		if res != nil {
			result = res
			break
		}
	}
	
	return result, nil
}

// CodeBlock представляет независимый блок кода
type CodeBlock struct {
	startPC uint64
	endPC   uint64
	code    []byte
}

// analyzeCodeBlocks анализирует код и разбивает его на независимые блоки
func analyzeCodeBlocks(code []byte) []CodeBlock {
	blocks := make([]CodeBlock, 0)
	
	// Простая эвристика: разбиваем код по JUMPDEST
	currentStart := uint64(0)
	
	for pc := uint64(0); pc < uint64(len(code)); pc++ {
		op := OpCode(code[pc])
		
		// JUMPDEST указывает на начало нового блока
		if op == JUMPDEST && pc > currentStart {
			blocks = append(blocks, CodeBlock{
				startPC: currentStart,
				endPC:   pc,
				code:    code[currentStart:pc],
			})
			currentStart = pc
		}
		
		// JUMP или RETURN заканчивает текущий блок
		if (op == JUMP || op == JUMPI || op == RETURN || op == STOP || op == REVERT) && pc > currentStart {
			blocks = append(blocks, CodeBlock{
				startPC: currentStart,
				endPC:   pc + 1,
				code:    code[currentStart:pc+1],
			})
			currentStart = pc + 1
		}
	}
	
	// Добавляем последний блок
	if currentStart < uint64(len(code)) {
		blocks = append(blocks, CodeBlock{
			startPC: currentStart,
			endPC:   uint64(len(code)),
			code:    code[currentStart:],
		})
	}
	
	return blocks
}

// isQuestCompatible проверяет, может ли операция быть ускорена через Quest
func isQuestCompatible(op OpCode) bool {
	compatibleOps := map[OpCode]bool{
		ADD:      true,
		MUL:      true,
		SUB:      true,
		DIV:      true,
		SDIV:     true,
		MOD:      true,
		SMOD:     true,
		EXP:      true,
		NOT:      true,
		LT:       true,
		GT:       true,
		SLT:      true,
		SGT:      true,
		EQ:       true,
		AND:      true,
		OR:       true,
		XOR:      true,
		BYTE:     true,
		SHL:      true,
		SHR:      true,
		SAR:      true,
		SHA3:     true,
		ADDRESS:  true,
		BALANCE:  true,
		ORIGIN:   true,
		CALLER:   true,
		CALLVALUE: true,
		CALLDATALOAD: true,
		CALLDATASIZE: true,
		CALLDATACOPY: true,
		GASPRICE: true,
		EXTCODESIZE: true,
		BLOCKHASH: true,
		TIMESTAMP: true,
		NUMBER:    true,
		DIFFICULTY: true,
		GASLIMIT:  true,
		CHAINID:   true,
		SELFBALANCE: true,
	}
	
	return compatibleOps[op]
}

// canRunParallel проверяет, может ли код быть выполнен параллельно
func canRunParallel(code []byte) bool {
	// Аналитическая функция для определения, можно ли выполнить код параллельно
	// Проверяем на наличие инструкций, которые не могут выполняться параллельно
	
	// Инструкции, которые нельзя выполнять параллельно
	nonParallelOps := map[OpCode]bool{
		JUMP:     true,
		JUMPI:    true,
		JUMPDEST: true,
		// Инструкции, изменяющие состояние
		SSTORE:   true,
		CREATE:   true,
		CALL:     true,
		CALLCODE: true,
		RETURN:   true,
		DELEGATECALL: true,
		CREATE2:  true,
		STATICCALL: true,
		REVERT:   true,
		SELFDESTRUCT: true,
	}
	
	for pc := 0; pc < len(code); pc++ {
		op := OpCode(code[pc])
		if nonParallelOps[op] {
			// Нашли инструкцию, которая не может выполняться параллельно
			return false
		}
	}
	
	// Дополнительный анализ зависимостей между блоками
	// Здесь может быть более сложная логика
	
	return true
}

// min возвращает минимальное из двух значений
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
