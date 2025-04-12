// Copyright 2023 The go-ethereum Authors
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

// package utils предоставляет вспомогательные функции для квантового процессора
package utils

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

// HardwareInfo предоставляет информацию об аппаратном обеспечении системы
type HardwareInfo struct {
	CPUCores     int
	CPUThreads   int
	MemoryGB     float64
	GPUInfo      []GPUInfo
	hasGPU       bool
	gpuDetected  bool
	mutex        sync.Mutex
}

// GPUInfo содержит информацию о графическом ускорителе
type GPUInfo struct {
	Name      string
	Memory    int // в МБ
	Available bool
}

// NewHardwareDetector создает новый детектор аппаратного обеспечения
func NewHardwareDetector() *HardwareInfo {
	info := &HardwareInfo{
		CPUCores:   runtime.NumCPU(),
		CPUThreads: runtime.NumCPU(), // В Go это обычно одно и то же
		MemoryGB:   16,               // Значение по умолчанию
		hasGPU:     false,
		gpuDetected: false,
	}
	
	// Инициализация выполняется при первом запросе HasGPU или при явном вызове DetectHardware
	
	return info
}

// DetectHardware определяет доступные аппаратные ресурсы
func (h *HardwareInfo) DetectHardware() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	// Избегаем повторного определения
	if h.gpuDetected {
		return
	}
	
	// Определяем количество ядер CPU (уже сделано при инициализации)
	
	// Проверяем наличие GPU
	h.detectGPU()
	
	h.gpuDetected = true
	
	log.Info("Обнаружено аппаратное обеспечение", 
		"cpu_cores", h.CPUCores, 
		"cpu_threads", h.CPUThreads,
		"memory_gb", h.MemoryGB,
		"has_gpu", h.hasGPU,
		"gpu_count", len(h.GPUInfo))
}

// detectGPU проверяет наличие GPU в системе
func (h *HardwareInfo) detectGPU() {
	// Сначала проверяем NVIDIA GPU с помощью nvidia-smi
	if h.detectNvidiaGPU() {
		h.hasGPU = true
		return
	}
	
	// Затем проверяем AMD GPU
	if h.detectAMDGPU() {
		h.hasGPU = true
		return
	}
	
	// Проверяем Intel GPU
	if h.detectIntelGPU() {
		h.hasGPU = true
		return
	}
	
	// Проверка для macOS (Metal API)
	if runtime.GOOS == "darwin" {
		if h.detectMacGPU() {
			h.hasGPU = true
			return
		}
	}
	
	log.Debug("GPU не обнаружены")
	h.hasGPU = false
}

// detectNvidiaGPU проверяет наличие NVIDIA GPU
func (h *HardwareInfo) detectNvidiaGPU() bool {
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader")
	output, err := cmd.Output()
	if err != nil {
		// nvidia-smi не доступен или ошибка выполнения
		return false
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[0])
			memoryStr := strings.TrimSpace(parts[1])
			
			// Примерное извлечение объема памяти
			var memory int
			fmt.Sscanf(memoryStr, "%d MiB", &memory)
			
			h.GPUInfo = append(h.GPUInfo, GPUInfo{
				Name:      name,
				Memory:    memory,
				Available: true,
			})
			
			log.Info("Обнаружен NVIDIA GPU", "name", name, "memory", memory)
		}
	}
	
	return len(h.GPUInfo) > 0
}

// detectAMDGPU проверяет наличие AMD GPU
func (h *HardwareInfo) detectAMDGPU() bool {
	// В реальной реализации здесь был бы код для обнаружения AMD GPU
	// Например, с использованием rocm-smi для AMD GPUs
	return false
}

// detectIntelGPU проверяет наличие Intel GPU
func (h *HardwareInfo) detectIntelGPU() bool {
	// В реальной реализации здесь был бы код для обнаружения Intel GPU
	return false
}

// detectMacGPU проверяет наличие GPU на macOS (Metal API)
func (h *HardwareInfo) detectMacGPU() bool {
	// На macOS просто предполагаем наличие GPU через Metal API
	if runtime.GOOS == "darwin" {
		h.GPUInfo = append(h.GPUInfo, GPUInfo{
			Name:      "Apple Metal GPU",
			Memory:    4096, // Предполагаем минимум 4GB
			Available: true,
		})
		log.Info("Обнаружен Apple Metal GPU")
		return true
	}
	return false
}

// HasGPU возвращает true, если в системе есть GPU
func (h *HardwareInfo) HasGPU() bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	// Если еще не определяли GPU, делаем это сейчас
	if !h.gpuDetected {
		h.mutex.Unlock()
		h.DetectHardware()
		h.mutex.Lock()
	}
	
	return h.hasGPU
}

// DetermineOptimalQubits определяет оптимальное количество кубитов на основе аппаратных ресурсов
func (h *HardwareInfo) DetermineOptimalQubits() int {
	// Убеждаемся, что аппаратное обеспечение определено
	if !h.gpuDetected {
		h.DetectHardware()
	}
	
	// Базовое количество кубитов
	baseQubits := 24
	
	// Корректируем в зависимости от доступного железа
	if h.HasGPU() {
		// У нас есть GPU
		if len(h.GPUInfo) > 0 && h.GPUInfo[0].Memory >= 8192 {
			// Мощное GPU с памятью >= 8 ГБ
			return baseQubits + 8
		}
		// Менее мощное GPU
		return baseQubits + 4
	}
	
	// Регулируем в зависимости от количества ядер CPU
	if h.CPUCores >= 16 {
		return baseQubits + 2
	} else if h.CPUCores >= 8 {
		return baseQubits + 1
	} else if h.CPUCores <= 2 {
		return baseQubits - 2
	}
	
	return baseQubits
}

// ShouldUseGPU определяет, следует ли использовать GPU для квантовых вычислений
func (h *HardwareInfo) ShouldUseGPU() bool {
	return h.HasGPU()
}

// Description возвращает текстовое описание аппаратного обеспечения
func (h *HardwareInfo) Description() string {
	if !h.gpuDetected {
		h.DetectHardware()
	}
	
	gpuDesc := "нет"
	if h.hasGPU && len(h.GPUInfo) > 0 {
		gpuNames := make([]string, 0, len(h.GPUInfo))
		for _, gpu := range h.GPUInfo {
			gpuNames = append(gpuNames, gpu.Name)
		}
		gpuDesc = strings.Join(gpuNames, ", ")
	}
	
	return fmt.Sprintf("CPU: %d ядер, Память: %.1f ГБ, GPU: %s", 
		h.CPUCores, h.MemoryGB, gpuDesc)
} 