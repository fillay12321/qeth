package test

import (
	"context"
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

// Тест для проверки алгоритма Шора
func TestShorFactorization(t *testing.T) {
	// Создаем два теста: с включенным и выключенным Quest
	t.Run("Шор с включенным Quest", func(t *testing.T) {
		testShorAlgorithm(t, true)
	})

	t.Run("Шор с выключенным Quest", func(t *testing.T) {
		testShorAlgorithm(t, false)
	})
}

// Функция для тестирования алгоритма Шора
func testShorAlgorithm(t *testing.T, enableQuest bool) {
	// Создаем симулятор
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr := crypto.PubkeyToAddress(key.PublicKey)

	genesis := core.GenesisAlloc{
		addr: {Balance: big.NewInt(1000000000000000000)}, // 1 ETH
	}

	config := vm.Config{
		EnableQuest: enableQuest,
	}

	backend := backends.NewSimulatedBackendWithConfig(genesis, 10000000, config)

	// Создаем транзакцию для авторизации
	auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	if err != nil {
		t.Fatalf("Не удалось создать транзактор: %v", err)
	}

	// Деплоим пустой контракт
	tx := types.NewContractCreation(0, big.NewInt(0), 3000000, big.NewInt(875000000), nil)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1337)), key)
	if err != nil {
		t.Fatalf("Не удалось подписать транзакцию: %v", err)
	}

	err = backend.SendTransaction(context.Background(), signedTx)
	if err != nil {
		t.Fatalf("Не удалось отправить транзакцию: %v", err)
	}

	backend.Commit()

	receipt, err := backend.TransactionReceipt(context.Background(), signedTx.Hash())
	if err != nil {
		t.Fatalf("Не удалось получить чек транзакции: %v", err)
	}

	contractAddress := receipt.ContractAddress

	// Составляем данные для вызова алгоритма Шора
	// В данных будет квантовый маркер и число для факторизации
	questMarker := []byte{0xF0, 0xF1}
	shorOp := []byte{0x02} // Опкод для алгоритма Шора
	
	// Знаменитое число RSA: N = 15 = 3 * 5 (простой пример)
	numberToFactorize := big.NewInt(15).Bytes()
	
	// Компонуем данные
	data := append(questMarker, shorOp...)
	data = append(data, numberToFactorize...)

	// Создаем транзакцию
	tx = types.NewTransaction(
		auth.Nonce.Uint64(),
		contractAddress,
		big.NewInt(0),
		5000000,
		auth.GasPrice,
		data,
	)

	signedTx, err = types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1337)), key)
	if err != nil {
		t.Fatalf("Не удалось подписать транзакцию: %v", err)
	}

	// Измеряем время выполнения
	start := time.Now()

	// Отправляем транзакцию
	err = backend.SendTransaction(context.Background(), signedTx)
	
	var factorizeResult string
	
	if err != nil {
		if enableQuest {
			t.Fatalf("Не удалось отправить транзакцию с включенным Quest: %v", err)
		} else {
			t.Logf("Ожидаемая ошибка с выключенным Quest: %v", err)
			factorizeResult = "Ошибка"
		}
	} else {
		backend.Commit()
		
		// Рассчитываем время выполнения
		duration := time.Since(start)
		
		// Получаем результаты
		receipt, err := backend.TransactionReceipt(context.Background(), signedTx.Hash())
		if err != nil {
			t.Fatalf("Не удалось получить чек транзакции: %v", err)
		}
		
		t.Logf("Время выполнения алгоритма Шора: %v (Quest включен: %v)", duration, enableQuest)
		t.Logf("Статус транзакции: %d, использовано газа: %d", receipt.Status, receipt.GasUsed)
		
		// Проверяем данные из лога
		if len(receipt.Logs) > 0 {
			// В реальности здесь был бы настоящий анализ данных
			if receipt.Logs[0].Data != nil && len(receipt.Logs[0].Data) >= 2 {
				factor1 := receipt.Logs[0].Data[0]
				factor2 := receipt.Logs[0].Data[1]
				factorizeResult = fmt.Sprintf("%d * %d = 15", factor1, factor2)
				
				// Проверяем, что факторы действительно верны
				if int(factor1) * int(factor2) == 15 {
					t.Logf("Факторизация успешна: %s", factorizeResult)
				} else {
					t.Errorf("Ошибка факторизации: %s != 15", factorizeResult)
				}
			} else {
				factorizeResult = "Нет данных в логах"
			}
		} else {
			factorizeResult = "Нет логов"
		}
	}
	
	t.Logf("Результат факторизации: %s", factorizeResult)
}

// Бенчмаркинг для алгоритма Шора
func BenchmarkShorAlgorithm(b *testing.B) {
	// Подготовка
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr := crypto.PubkeyToAddress(key.PublicKey)

	genesis := core.GenesisAlloc{
		addr: {Balance: big.NewInt(1000000000000000000)},
	}
	
	// Тестирование с различными числами
	numTests := []int64{15, 21, 35, 91, 143} // Произведения простых чисел

	for _, num := range numTests {
		numToFactor := big.NewInt(num)
		
		// Бенчмарк с включенным Quest
		b.Run(fmt.Sprintf("Шор-Quest-%d", num), func(b *testing.B) {
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
			
			// Формируем данные
			questMarker := []byte{0xF0, 0xF1}
			shorOp := []byte{0x02}
			numberToFactorize := numToFactor.Bytes()
			
			data := append(questMarker, shorOp...)
			data = append(data, numberToFactorize...)
			
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
		
		// Классическая факторизация (пробным делением)
		b.Run(fmt.Sprintf("Классический-%d", num), func(b *testing.B) {
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				// Простой алгоритм пробного деления
				n := numToFactor.Int64()
				var factors []int64
				
				// Проверяем делители от 2 до sqrt(n)
				for d := int64(2); d*d <= n; d++ {
					for n%d == 0 {
						factors = append(factors, d)
						n /= d
					}
				}
				
				// Если n > 1, то n - простое число
				if n > 1 {
					factors = append(factors, n)
				}
				
				// Проверка (можно убрать в продакшн-коде)
				product := int64(1)
				for _, f := range factors {
					product *= f
				}
				
				if product != numToFactor.Int64() {
					b.Fatalf("Ошибка факторизации: %d != %s", product, numToFactor.String())
				}
			}
		})
	}
} 