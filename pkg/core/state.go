package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StateManager gerencia o estado local
type StateManager struct {
	StateDir string
}

// NewStateManager cria um novo gerenciador de estado
func NewStateManager(stateDir string) *StateManager {
	if stateDir == "" {
		home, _ := os.UserHomeDir()
		stateDir = filepath.Join(home, ".infra-operator", "state")
	}
	return &StateManager{StateDir: stateDir}
}

// EnsureDir garante que o diretório de estado existe
func (s *StateManager) EnsureDir() error {
	return os.MkdirAll(s.StateDir, 0755)
}

// GetStatePath retorna o caminho para o arquivo de estado de um recurso
func (s *StateManager) GetStatePath(kind, namespace, name string) string {
	if namespace == "" {
		namespace = "default"
	}
	return filepath.Join(s.StateDir, kind, namespace, name+".json")
}

// SaveState salva o estado de um recurso
func (s *StateManager) SaveState(state *ResourceState) error {
	if err := s.EnsureDir(); err != nil {
		return fmt.Errorf("falha ao garantir diretório de estado: %w", err)
	}

	statePath := s.GetStatePath(state.Kind, state.Namespace, state.Name)
	stateDir := filepath.Dir(statePath)

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar diretório de estado: %w", err)
	}

	state.UpdatedAt = time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = state.UpdatedAt
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("falha ao serializar estado: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("falha ao escrever arquivo de estado: %w", err)
	}

	return nil
}

// LoadState carrega o estado de um recurso
func (s *StateManager) LoadState(kind, namespace, name string) (*ResourceState, error) {
	statePath := s.GetStatePath(kind, namespace, name)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("falha ao ler arquivo de estado: %w", err)
	}

	var state ResourceState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("falha ao deserializar estado: %w", err)
	}

	return &state, nil
}

// DeleteState deleta o estado de um recurso
func (s *StateManager) DeleteState(kind, namespace, name string) error {
	statePath := s.GetStatePath(kind, namespace, name)

	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("falha ao deletar arquivo de estado: %w", err)
	}

	return nil
}

// ListStates lista todos os recursos de um tipo específico
func (s *StateManager) ListStates(kind string) ([]*ResourceState, error) {
	var states []*ResourceState

	kindDir := filepath.Join(s.StateDir, kind)
	if _, err := os.Stat(kindDir); os.IsNotExist(err) {
		return states, nil
	}

	err := filepath.Walk(kindDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var state ResourceState
			if err := json.Unmarshal(data, &state); err != nil {
				return err
			}
			states = append(states, &state)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("falha ao listar estados: %w", err)
	}

	return states, nil
}

// ListAllStates lista todos os recursos no estado
func (s *StateManager) ListAllStates() ([]*ResourceState, error) {
	var states []*ResourceState

	if _, err := os.Stat(s.StateDir); os.IsNotExist(err) {
		return states, nil
	}

	err := filepath.Walk(s.StateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var state ResourceState
			if err := json.Unmarshal(data, &state); err != nil {
				return err
			}
			states = append(states, &state)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("falha ao listar todos os estados: %w", err)
	}

	return states, nil
}
