// Package config stores and exports the configuration for server-side use and
// the public availability JSON struct, which includes a small subset of the
// server configuration.
package config

import (
	"sort"
	"sync"

	"meguca/common"
)

var (
	// Ensures no reads happen, while the configuration is reloading
	configMu, boardMu sync.RWMutex

	// Contains currently loaded global server configuration
	config *ServerConfig

	// JSON of client-accessible configuration
	configJSON []byte

	// Map of board IDs to their configuration structs
	boardConfigs = map[string]BoardConfig{}

	// Defaults contains the default server configuration values
	DefaultServerConfig = ServerConfig{
		ServerPublic: ServerPublic{
			DisableUserBoards: true,
			MaxSize:           common.DefaultMaxSize,
			MaxFiles:          common.DefaultMaxFiles,
			DefaultLang:       common.DefaultLang,
			DefaultCSS:        common.DefaultCSS,
		},
	}
)

func Get() *ServerConfig {
	return config
}

func GetJSON() []byte {
	return configJSON
}

func Set(c ServerConfig) (err error) {
	data, err := c.ServerPublic.MarshalJSON()
	if err != nil {
		return
	}
	configMu.Lock()
	defer configMu.Unlock()
	configJSON = data
	config = &c
	return
}

func GetBoardsJSON() (data []byte) {
	boardMu.RLock()
	defer boardMu.RUnlock()
	data = append(data, '[')
	i := 0
	for _, conf := range boardConfigs {
		if conf.ID == "all" {
			continue
		}
		if conf.ModOnly {
			continue
		}
		if i > 0 {
			data = append(data, ',')
		}
		data = append(data, conf.json...)
		i++
	}
	data = append(data, ']')
	return
}

func GetBoardConfig(b string) BoardConfig {
	boardMu.RLock()
	defer boardMu.RUnlock()
	return boardConfigs[b]
}

func GetBoardConfigs() (cs BoardConfigs) {
	boardMu.RLock()
	defer boardMu.RUnlock()
	for _, conf := range boardConfigs {
		if conf.ID == "all" {
			continue
		}
		cs = append(cs, conf)
	}
	return
}

// All boards including /all/.
func GetAdminBoardIDs() (ids []string) {
	boardMu.RLock()
	defer boardMu.RUnlock()
	for id := range boardConfigs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return
}

// All boards including /all/.
func GetModBoardConfigsByID(ids []string) (cs BoardConfigs) {
	boardMu.RLock()
	defer boardMu.RUnlock()
	for _, id := range ids {
		if conf, ok := boardConfigs[id]; ok {
			cs = append(cs, conf)
		}
	}
	sort.Sort(cs)
	return
}

// Set configurations for a specific board as well as pregenerate its
// public JSON. Return if any changes were made to the configs.
func SetBoardConfig(conf BoardConfig) (err error) {
	data, err := conf.BoardPublic.MarshalJSON()
	if err != nil {
		return
	}
	boardMu.Lock()
	defer boardMu.Unlock()
	conf.json = data
	boardConfigs[conf.ID] = conf
	return
}

// RemoveBoard removes a board from the exiting board list and deletes its
// configurations. To be called, when a board is deleted.
func RemoveBoard(b string) {
	boardMu.Lock()
	defer boardMu.Unlock()
	delete(boardConfigs, b)
}

func IsBoard(b string) bool {
	boardMu.RLock()
	defer boardMu.RUnlock()
	_, ok := boardConfigs[b]
	return ok
}

func IsReadOnlyBoard(b string) bool {
	boardMu.RLock()
	defer boardMu.RUnlock()
	conf, ok := boardConfigs[b]
	return ok && conf.ReadOnly
}

func IsModOnlyBoard(b string) bool {
	boardMu.RLock()
	defer boardMu.RUnlock()
	conf, ok := boardConfigs[b]
	return ok && conf.ModOnly
}
