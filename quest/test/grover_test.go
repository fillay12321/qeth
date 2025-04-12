package test

import (
	"context"
	"crypto/sha256"
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

// Тест для поиска прообраза SHA-256 с помощью алгоритма Гровера
func TestGroverHashPreimage(t *testing.T) {
	// Генерируем тестовое значение хэша
	targetPreimage := []byte("квантовый_ethereum")
	targetHash := sha256.Sum256(targetPreimage)
	
	t.Logf("Целевой прообраз: %s", string(targetPreimage))
	t.Logf("SHA-256 хэш: %x", targetHash)
	
	// Проверяем с включенным и выключенным Quest
	t.Run("Гровер с включенным Quest", func(t *testing.T) {
		testGroverAlgorithm(t, true, targetHash[:])
	})
	
	t.Run("Классический поиск", func(t *testing.T) {
		testClassicalSearch(t, targetHash[:])
	})
}

// Функция для тестирования алгоритма Гровера
func testGroverAlgorithm(t *testing.T, enableQuest bool, targetHash []byte) {
	// Настраиваем среду тестирования
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr := crypto.PubkeyToAddress(key.PublicKey)

	genesis := core.GenesisAlloc{
		addr: {Balance: big.NewInt(1000000000000000000)}, // 1 ETH
	}

	config := vm.Config{
		EnableQuest: enableQuest,
	}

	backend := backends.NewSimulatedBackendWithConfig(genesis, 10000000, config)

	// Деплоим специальный контракт для квантовых вычислений
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
	
	// Компонуем данные для вызова алгоритма Гровера
	questMarker := []byte{0xF0, 0xF1}
	groverOp := []byte{0x01} // Опкод для алгоритма Гровера
	
	// Добавляем целевой хэш
	data := append(questMarker, groverOp...)
	data = append(data, targetHash...)
	
	// Для простоты теста, также добавим информацию о битовом размере поиска (8 бит = 256 вариантов)
	// В реальности искали бы в гораздо большем пространстве
	searchSpaceBits := []byte{0x08}
	data = append(data, searchSpaceBits...)
	
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
	if err != nil {
		if enableQuest {
			t.Fatalf("Не удалось отправить транзакцию с включенным Quest: %v", err)
		} else {
			t.Logf("Ожидаемая ошибка с выключенным Quest: %v", err)
		}
		return
	}
	
	backend.Commit()
	
	// Рассчитываем время выполнения
	duration := time.Since(start)
	
	// Получаем результаты
	receipt, err = backend.TransactionReceipt(context.Background(), signedTx.Hash())
	if err != nil {
		t.Fatalf("Не удалось получить чек транзакции: %v", err)
	}
	
	t.Logf("Время выполнения алгоритма Гровера: %v", duration)
	t.Logf("Статус транзакции: %d, использовано газа: %d", receipt.Status, receipt.GasUsed)
	
	// Проверяем данные из лога
	if len(receipt.Logs) > 0 {
		// Здесь бы декодировали найденный прообраз
		if receipt.Logs[0].Data != nil {
			foundPreimage := receipt.Logs[0].Data
			t.Logf("Найденный прообраз: %x", foundPreimage)
			
			// Проверяем, что хэш действительно совпадает
			computedHash := sha256.Sum256(foundPreimage)
			if common.BytesToHash(computedHash[:]) == common.BytesToHash(targetHash) {
				t.Logf("Прообраз ВЕРНЫЙ!")
			} else {
				t.Errorf("Прообраз НЕВЕРНЫЙ! Хэш: %x", computedHash)
			}
		} else {
			t.Logf("Нет данных в логах")
		}
	} else {
		t.Logf("Нет логов")
	}
}

// Функция для тестирования классического поиска прообраза
func testClassicalSearch(t *testing.T, targetHash []byte) {
	// Для простоты демонстрации ограничим поиск небольшим пространством
	maxAttempts := 1000
	
	start := time.Now()
	
	// Простой перебор случайных значений
	var foundPreimage []byte
	found := false
	
	for i := 0; i < maxAttempts; i++ {
		// Генерируем случайное значение
		candidate := []byte(fmt.Sprintf("test_value_%d", i))
		
		// Считаем хэш
		hash := sha256.Sum256(candidate)
		
		// Проверяем совпадение
		if common.BytesToHash(hash[:]) == common.BytesToHash(targetHash) {
			foundPreimage = candidate
			found = true
			break
		}
	}
	
	duration := time.Since(start)
	
	if found {
		t.Logf("Классический поиск нашел прообраз за %v: %s", duration, string(foundPreimage))
	} else {
		t.Logf("Классический поиск не нашел прообраз за %v после %d попыток", duration, maxAttempts)
	}
}

// Бенчмарк для сравнения производительности
func BenchmarkGroverVsClassical(b *testing.B) {
	// Подготовка тестовых данных для разных размеров
	testSizes := []int{8, 10, 12, 14, 16} // размеры в битах
	
	for _, size := range testSizes {
		// Создаем фиксированный хэш для тестирования
		targetString := fmt.Sprintf("test_hash_%d", size)
		targetHashFull := sha256.Sum256([]byte(targetString))
		targetHash := targetHashFull[:]
		
		// Квантовый алгоритм Гровера
		b.Run(fmt.Sprintf("Гровер-%d-бит", size), func(b *testing.B) {
			key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
			addr := crypto.PubkeyToAddress(key.PublicKey)

			genesis := core.GenesisAlloc{
				addr: {Balance: big.NewInt(1000000000000000000)},
			}

			config := vm.Config{
				EnableQuest: true, // Включаем квантовый режим
			}

			backend := backends.NewSimulatedBackendWithConfig(genesis, 10000000, config)
			auth, _ := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
			
			// Деплоим контракт
			tx := types.NewContractCreation(0, big.NewInt(0), 3000000, big.NewInt(875000000), nil)
			signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1337)), key)
			backend.SendTransaction(context.Background(), signedTx)
			backend.Commit()
			receipt, _ := backend.TransactionReceipt(context.Background(), signedTx.Hash())
			contractAddress := receipt.ContractAddress
			
			// Формируем данные
			questMarker := []byte{0xF0, 0xF1}
			groverOp := []byte{0x01}
			searchSpaceBits := []byte{byte(size)}
			
			data := append(questMarker, groverOp...)
			data = append(data, targetHash...)
			data = append(data, searchSpaceBits...)
			
			b.ResetTimer()
			
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
		
		// Классический перебор
		b.Run(fmt.Sprintf("Классический-%d-бит", size), func(b *testing.B) {
			maxSpace := 1 << size // 2^size
			
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				// Перебираем все возможные значения в пространстве поиска
				for j := 0; j < maxSpace; j++ {
					candidate := []byte(fmt.Sprintf("value_%d", j))
					hash := sha256.Sum256(candidate)
					
					// Простая проверка первых байтов для ускорения
					if hash[0] == targetHash[0] && hash[1] == targetHash[1] {
						// Полная проверка
						if common.BytesToHash(hash[:]) == common.BytesToHash(targetHash) {
							break
						}
					}
				}
			}
		})
	}
} 