/**
 * quest_kit.h - Интерфейс для взаимодействия с квантовым процессором Quest
 * 
 * Этот заголовочный файл определяет API для использования квантового процессора
 * Quest для выполнения транзакций и симуляции квантовых схем из Go-кода.
 */

#ifndef QUEST_KIT_H
#define QUEST_KIT_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stddef.h>
#include <stdint.h>

/**
 * Тип для обработчика состояния Quest
 */
typedef void* quest_handle_t;

/**
 * Устанавливает количество потоков для параллельного выполнения.
 * 
 * @param num_threads Количество потоков
 */
void quest_set_num_threads(int num_threads);

/**
 * Инициализирует процессор Quest.
 * 
 * @param handle Указатель для сохранения обработчика Quest
 * @return 0 в случае успеха, ненулевое значение в случае ошибки
 */
int quest_initialize(quest_handle_t* handle);

/**
 * Освобождает ресурсы процессора Quest.
 * 
 * @param handle Обработчик Quest
 * @return 0 в случае успеха, ненулевое значение в случае ошибки
 */
int quest_finalize(quest_handle_t handle);

/**
 * Выполняет транзакцию с использованием квантового процессора.
 * 
 * @param handle Обработчик Quest
 * @param data Данные транзакции
 * @param data_size Размер данных транзакции
 * @param sender Адрес отправителя
 * @param sender_size Размер адреса отправителя
 * @param result_size Указатель для сохранения размера результата
 * @return Указатель на результат выполнения (должен быть освобожден с помощью quest_free_result)
 */
unsigned char* quest_execute_transaction(
    quest_handle_t handle,
    const unsigned char* data,
    size_t data_size,
    const unsigned char* sender,
    size_t sender_size,
    size_t* result_size
);

/**
 * Симулирует квантовую схему.
 * 
 * @param handle Обработчик Quest
 * @param circuit Данные квантовой схемы
 * @param circuit_size Размер данных квантовой схемы
 * @param result_size Указатель для сохранения размера результата
 * @return Указатель на результат симуляции (должен быть освобожден с помощью quest_free_result)
 */
unsigned char* quest_simulate_circuit(
    quest_handle_t handle,
    const unsigned char* circuit,
    size_t circuit_size,
    size_t* result_size
);

/**
 * Освобождает память, выделенную для результата.
 * 
 * @param result Указатель на результат
 */
void quest_free_result(unsigned char* result);

/**
 * Вычисляет хэш состояния квантовой системы.
 * 
 * @param handle Обработчик Quest
 * @param hash Указатель на буфер для хэша (размер должен быть не менее 32 байт)
 * @return 0 в случае успеха, ненулевое значение в случае ошибки
 */
int quest_calc_state_hash(quest_handle_t handle, unsigned char* hash);

/**
 * Возвращает версию библиотеки quest-kit.
 * 
 * @return Строка с версией библиотеки
 */
const char* quest_version();

#ifdef __cplusplus
}
#endif

#endif /* QUEST_KIT_H */ 