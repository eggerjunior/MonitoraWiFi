// Package queue implementa o buffer offline do agente (Seção 3: "Fila local
// quando não houver internet; reenvio idempotente; limite de
// armazenamento"). Um arquivo JSON Lines em disco — sobrevive a reinício do
// processo, o que uma fila só em memória não faria.
package queue

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileQueue é uma fila FIFO persistida em disco como JSON Lines. Um item por
// linha; Drain lê tudo, tenta enviar, e só remove do arquivo o que o
// remetente confirmou como entregue.
type FileQueue[T any] struct {
	path     string
	maxItems int
	mu       sync.Mutex
}

func NewFileQueue[T any](path string, maxItems int) *FileQueue[T] {
	return &FileQueue[T]{path: path, maxItems: maxItems}
}

// Enqueue adiciona um item ao fim da fila. Quando o número de itens excede
// maxItems, os mais antigos são descartados — a chamada retorna
// (dropped=true) nesse caso para que o chamador registre isso como evento
// (nunca falha silenciosamente).
func (q *FileQueue[T]) Enqueue(item T) (dropped bool, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	items, err := q.readAll()
	if err != nil {
		return false, err
	}
	items = append(items, item)

	dropped = false
	if q.maxItems > 0 && len(items) > q.maxItems {
		excess := len(items) - q.maxItems
		items = items[excess:]
		dropped = true
	}

	return dropped, q.writeAll(items)
}

// Peek retorna todos os itens pendentes sem removê-los.
func (q *FileQueue[T]) Peek() ([]T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.readAll()
}

// Drain chama send com todos os itens pendentes. Se send não retornar erro,
// os itens são removidos da fila. Se send falhar, a fila permanece intacta
// para uma nova tentativa (o chamador decide o backoff).
func (q *FileQueue[T]) Drain(send func([]T) error) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	items, err := q.readAll()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	if err := send(items); err != nil {
		return err
	}

	return q.writeAll(nil)
}

func (q *FileQueue[T]) Len() (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	items, err := q.readAll()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (q *FileQueue[T]) readAll() ([]T, error) {
	f, err := os.Open(q.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("abrir fila %s: %w", q.path, err)
	}
	defer f.Close()

	var items []T
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var item T
		if err := json.Unmarshal(line, &item); err != nil {
			continue // linha corrompida — ignora em vez de travar a fila inteira
		}
		items = append(items, item)
	}
	return items, scanner.Err()
}

func (q *FileQueue[T]) writeAll(items []T) error {
	if err := os.MkdirAll(filepath.Dir(q.path), 0o700); err != nil {
		return fmt.Errorf("criar diretório da fila: %w", err)
	}

	tmpPath := q.path + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("criar arquivo temporário da fila: %w", err)
	}

	w := bufio.NewWriter(f)
	for _, item := range items {
		b, err := json.Marshal(item)
		if err != nil {
			f.Close()
			return fmt.Errorf("serializar item da fila: %w", err)
		}
		w.Write(b)
		w.WriteByte('\n')
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return fmt.Errorf("gravar fila: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("fechar arquivo temporário da fila: %w", err)
	}

	// Rename é atômico no mesmo filesystem — evita fila corrompida se o
	// processo morrer no meio da escrita.
	return os.Rename(tmpPath, q.path)
}
