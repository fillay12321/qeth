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

package vm

import (
	"sync"
)

// Extension options for Config
type configExtensions struct {
	// UseQuestProcessor указывает, использовать ли процессор Quest вместо стандартного EVM
	EnableQuest bool

	// QuestOptions дополнительные настройки для квантового процессора Quest
	QuestOptions map[string]interface{}
}

var (
	// extensionsMutex защищает доступ к extensions
	extensionsMutex sync.RWMutex

	// extensions содержит список функций расширения конфигурации
	extensions []func(*Config)
)

// RegisterConfigExtension регистрирует функцию для расширения конфигурации
func RegisterConfigExtension(extension func(*Config)) {
	extensionsMutex.Lock()
	defer extensionsMutex.Unlock()
	extensions = append(extensions, extension)
}

// applyConfigExtensions применяет все зарегистрированные расширения к конфигурации
func applyConfigExtensions(config *Config) {
	extensionsMutex.RLock()
	defer extensionsMutex.RUnlock()
	for _, extension := range extensions {
		extension(config)
	}
}

// Методы доступа к расширениям Config

// SetEnableQuest включает или выключает использование квантового процессора Quest
func (c *Config) SetEnableQuest(enable bool) {
	c.EnableQuest = enable
}

// SetQuestOption устанавливает опцию для квантового процессора Quest
func (c *Config) SetQuestOption(key string, value interface{}) {
	if c.QuestOptions == nil {
		c.QuestOptions = make(map[string]interface{})
	}
	c.QuestOptions[key] = value
}

// GetQuestOption возвращает опцию квантового процессора Quest
func (c *Config) GetQuestOption(key string) (interface{}, bool) {
	if c.QuestOptions == nil {
		return nil, false
	}
	val, ok := c.QuestOptions[key]
	return val, ok
} 