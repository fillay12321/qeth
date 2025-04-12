package test

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

// Тестовые приватные ключи (НЕ ИСПОЛЬЗОВАТЬ В ПРОДАКШЕНЕ)
var (
	key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr   = crypto.PubkeyToAddress(key.PublicKey)
)

// Контракт квантового теста (заглушка для использования с симулятором)
type QuestTest struct {
	address common.Address
	backend *backends.SimulatedBackend
	auth    *bind.TransactOpts
}

// Создание нового экземпляра контракта QuestTest
func deployQuestTest(t *testing.T, enableQuest bool) (*QuestTest, error) {
	// Генерируем начальное состояние блокчейна
	genesis := core.GenesisAlloc{
		addr: {Balance: big.NewInt(1000000000000000000)}, // 1 ETH
	}
	
	// Создаем симулятор с настройками квантового процессора
	config := vm.Config{
		EnableQuest: enableQuest,
	}
	
	backend := backends.NewSimulatedBackendWithConfig(genesis, 10000000, config)
	
	// Создаем транзакцию для авторизации
	auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	if err != nil {
		t.Fatalf("Не удалось создать транзактор: %v", err)
		return nil, err
	}
	
	// Так как у нас нет реального ABI и bytecode, мы имитируем их для теста
	// В реальном сценарии здесь был бы код компиляции из Solidity
	
	// Здесь мы просто создаем пустой контракт для демонстрации
	tx := types.NewContractCreation(0, big.NewInt(0), 3000000, big.NewInt(875000000), nil)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1337)), key)
	if err != nil {
		t.Fatalf("Не удалось подписать транзакцию: %v", err)
		return nil, err
	}
	
	// Отправляем транзакцию
	if err := backend.SendTransaction(context.Background(), signedTx); err != nil {
		t.Fatalf("Не удалось отправить транзакцию: %v", err)
		return nil, err
	}
	
	backend.Commit()
	
	// Получаем адрес контракта
	receipt, err := backend.TransactionReceipt(context.Background(), signedTx.Hash())
	if err != nil {
		t.Fatalf("Не удалось получить чек транзакции: %v", err)
		return nil, err
	}
	
	return &QuestTest{
		address: receipt.ContractAddress,
		backend: backend,
		auth:    auth,
	}, nil
}

// Простой тест для проверки включения/выключения квантового режима
func TestQuestEnablement(t *testing.T) {
	// Сначала запускаем с выключенным Quest
	_, err := deployQuestTest(t, false)
	if err != nil {
		t.Fatal("Не удалось развернуть тестовый контракт без Quest:", err)
	}
	
	// Затем с включенным Quest
	_, err = deployQuestTest(t, true)
	if err != nil {
		t.Fatal("Не удалось развернуть тестовый контракт с Quest:", err)
	}
	
	t.Log("Тест успешно пройден: Quest может быть включен и выключен без ошибок")
}

// Имитация вызова квантовой функции Гровера
func simulateGroversAlgorithm(t *testing.T, enableQuest bool) {
	contract, err := deployQuestTest(t, enableQuest)
	if err != nil {
		t.Fatal("Не удалось развернуть тестовый контракт:", err)
	}
	
	// Создаем данные для вызова квантовой функции
	// В реальном коде здесь был бы вызов метода из привязанного ABI
	questMarker := []byte{0xF0, 0xF1}
	groversSearchOp := []byte{0x01}
	searchSpace := big.NewInt(1000000).Bytes()
	target := big.NewInt(42).Bytes()
	
	data := append(questMarker, groversSearchOp...)
	data = append(data, searchSpace...)
	data = append(data, target...)
	
	// Создаем и отправляем транзакцию
	tx := types.NewTransaction(
		contract.auth.Nonce.Uint64(),
		contract.address,
		big.NewInt(0),
		5000000,
		contract.auth.GasPrice,
		data,
	)
	
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1337)), key)
	if err != nil {
		t.Fatalf("Не удалось подписать транзакцию: %v", err)
	}
	
	start := time.Now()
	
	err = contract.backend.SendTransaction(context.Background(), signedTx)
	if err != nil {
		if enableQuest {
			t.Fatalf("Не удалось отправить транзакцию с включенным Quest: %v", err)
		} else {
			t.Logf("Ожидаемая ошибка с выключенным Quest: %v", err)
		}
	} else {
		contract.backend.Commit()
		
		duration := time.Since(start)
		t.Logf("Время выполнения Гровера: %v (Quest включен: %v)", duration, enableQuest)
		
		receipt, err := contract.backend.TransactionReceipt(context.Background(), signedTx.Hash())
		if err != nil {
			t.Fatalf("Не удалось получить чек транзакции: %v", err)
		}
		
		t.Logf("Статус транзакции: %d, использовано газа: %d", receipt.Status, receipt.GasUsed)
	}
}

// Тест для сравнения производительности с включенным и выключенным Quest
func TestGroversPerformance(t *testing.T) {
	// Тестирование с выключенным Quest
	t.Run("Гровер без Quest", func(t *testing.T) {
		simulateGroversAlgorithm(t, false)
	})
	
	// Тестирование с включенным Quest
	t.Run("Гровер с Quest", func(t *testing.T) {
		simulateGroversAlgorithm(t, true)
	})
}

// Бенчмаркинг для измерения производительности
func BenchmarkGroversAlgorithm(b *testing.B) {
	// Подготовка
	genesis := core.GenesisAlloc{
		addr: {Balance: big.NewInt(1000000000000000000)},
	}
	
	// Бенчмарк с включенным Quest
	b.Run("Quest включен", func(b *testing.B) {
		config := vm.Config{
			EnableQuest: true,
		}
		backend := backends.NewSimulatedBackendWithConfig(genesis, 10000000, config)
		auth, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
		
		// Деплоим контракт один раз для всех запусков
		tx := types.NewContractCreation(0, big.NewInt(0), 3000000, big.NewInt(875000000), nil)
		signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1337)), key)
		backend.SendTransaction(context.Background(), signedTx)
		backend.Commit()
		receipt, _ := backend.TransactionReceipt(context.Background(), signedTx.Hash())
		contractAddress := receipt.ContractAddress
		
		// Подготавливаем данные для квантовой операции
		questMarker := []byte{0xF0, 0xF1}
		groversSearchOp := []byte{0x01}
		searchSpace := big.NewInt(1000000).Bytes()
		target := big.NewInt(42).Bytes()
		
		data := append(questMarker, groversSearchOp...)
		data = append(data, searchSpace...)
		data = append(data, target...)
		
		b.ResetTimer()
		
		// Запускаем бенчмарк
		for i := 0; i < b.N; i++ {
			tx := types.NewTransaction(
				uint64(i),
				contractAddress,
				big.NewInt(0),
				5000000,
				auth.GasPrice,
				data,
			)
			
			signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1337)), key)
			backend.SendTransaction(context.Background(), signedTx)
			backend.Commit()
		}
	})
	
	// Бенчмарк с выключенным Quest (ожидаем ошибки, но измеряем скорость проверки)
	b.Run("Quest выключен", func(b *testing.B) {
		config := vm.Config{
			EnableQuest: false,
		}
		backend := backends.NewSimulatedBackendWithConfig(genesis, 10000000, config)
		auth, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
		
		// Деплоим контракт один раз для всех запусков
		tx := types.NewContractCreation(0, big.NewInt(0), 3000000, big.NewInt(875000000), nil)
		signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1337)), key)
		backend.SendTransaction(context.Background(), signedTx)
		backend.Commit()
		receipt, _ := backend.TransactionReceipt(context.Background(), signedTx.Hash())
		contractAddress := receipt.ContractAddress
		
		// Подготавливаем данные для квантовой операции
		questMarker := []byte{0xF0, 0xF1}
		groversSearchOp := []byte{0x01}
		searchSpace := big.NewInt(1000000).Bytes()
		target := big.NewInt(42).Bytes()
		
		data := append(questMarker, groversSearchOp...)
		data = append(data, searchSpace...)
		data = append(data, target...)
		
		b.ResetTimer()
		
		// Запускаем бенчмарк (ожидаем ошибки)
		for i := 0; i < b.N; i++ {
			tx := types.NewTransaction(
				uint64(i),
				contractAddress,
				big.NewInt(0),
				5000000,
				auth.GasPrice,
				data,
			)
			
			signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1337)), key)
			backend.SendTransaction(context.Background(), signedTx)
			backend.Commit()
		}
	})
}

func TestMain(m *testing.M) {
	fmt.Println("Запуск тестов Quest интеграции с Ethereum")
	m.Run()
} 