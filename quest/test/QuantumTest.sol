// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract QuantumTest {
    // Маркер для квантового процессора (в байт-коде будет 0xF0F1)
    bytes2 constant QUANTUM_MARKER = hex"F0F1";
    
    uint public result;
    
    // Функция для квантового вычисления (используется QUANTUM_MARKER)
    function quantumComputation(uint[] calldata data) public returns (uint) {
        // Вставляем квантовый маркер для выполнения на Quest
        bytes memory quantum = abi.encodePacked(QUANTUM_MARKER);
        
        assembly {
            // Размещаем маркер в памяти для его обнаружения
            mstore(0, quantum)
        }
        
        // Простое вычисление (сумма элементов)
        uint sum = 0;
        for (uint i = 0; i < data.length; i++) {
            sum += data[i];
        }
        
        result = sum;
        return sum;
    }
    
    // Стандартное EVM-вычисление (без QUANTUM_MARKER)
    function standardComputation(uint[] calldata data) public returns (uint) {
        // То же самое вычисление, но без квантового маркера
        uint sum = 0;
        for (uint i = 0; i < data.length; i++) {
            sum += data[i];
        }
        
        result = sum;
        return sum;
    }
    
    // Функция для бенчмаркинга, которая выполняет сложные вычисления
    function benchmark(uint iterations) public returns (uint) {
        // Вставляем квантовый маркер для выполнения на Quest
        bytes memory quantum = abi.encodePacked(QUANTUM_MARKER);
        
        assembly {
            // Размещаем маркер в памяти для его обнаружения
            mstore(0, quantum)
        }
        
        // Выполняем сложные вычисления
        uint val = 1;
        for (uint i = 0; i < iterations; i++) {
            val = (val * 7 + 11) % 1000000007;
        }
        
        result = val;
        return val;
    }
    
    // Та же функция для бенчмаркинга, но без квантового маркера
    function standardBenchmark(uint iterations) public returns (uint) {
        // Выполняем те же вычисления без квантового маркера
        uint val = 1;
        for (uint i = 0; i < iterations; i++) {
            val = (val * 7 + 11) % 1000000007;
        }
        
        result = val;
        return val;
    }
} 