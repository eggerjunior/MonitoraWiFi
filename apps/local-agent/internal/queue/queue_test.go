package queue

import (
	"errors"
	"path/filepath"
	"testing"
)

type sample struct {
	Value int `json:"value"`
}

func TestFileQueue_EnqueueAndDrain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.jsonl")
	q := NewFileQueue[sample](path, 100)

	for i := 1; i <= 3; i++ {
		if _, err := q.Enqueue(sample{Value: i}); err != nil {
			t.Fatalf("erro ao enfileirar: %v", err)
		}
	}

	var sent []sample
	err := q.Drain(func(items []sample) error {
		sent = items
		return nil
	})
	if err != nil {
		t.Fatalf("erro ao drenar: %v", err)
	}
	if len(sent) != 3 {
		t.Fatalf("esperava 3 itens enviados, recebeu %d", len(sent))
	}

	remaining, err := q.Len()
	if err != nil {
		t.Fatalf("erro ao medir fila: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("esperava fila vazia após drain com sucesso, restaram %d", remaining)
	}
}

func TestFileQueue_KeepsItemsWhenSendFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.jsonl")
	q := NewFileQueue[sample](path, 100)
	q.Enqueue(sample{Value: 1})

	err := q.Drain(func(items []sample) error {
		return errors.New("backend indisponível")
	})
	if err == nil {
		t.Fatal("esperava erro propagado quando o envio falha")
	}

	remaining, _ := q.Len()
	if remaining != 1 {
		t.Fatalf("esperava que o item permanecesse na fila após falha de envio, restaram %d", remaining)
	}
}

func TestFileQueue_DropsOldestWhenExceedsMax(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.jsonl")
	q := NewFileQueue[sample](path, 3)

	for i := 1; i <= 5; i++ {
		q.Enqueue(sample{Value: i})
	}

	items, err := q.Peek()
	if err != nil {
		t.Fatalf("erro ao inspecionar fila: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("esperava 3 itens (limite), encontrou %d", len(items))
	}
	// Os 3 mais recentes (3,4,5) devem ter sobrevivido — os mais antigos
	// (1,2) foram descartados primeiro.
	if items[0].Value != 3 || items[2].Value != 5 {
		t.Fatalf("esperava manter os itens mais recentes [3,4,5], encontrou %+v", items)
	}
}

func TestFileQueue_LastEnqueueReportsDropped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.jsonl")
	q := NewFileQueue[sample](path, 2)

	q.Enqueue(sample{Value: 1})
	dropped, _ := q.Enqueue(sample{Value: 2})
	if dropped {
		t.Fatal("não deveria descartar nada ainda (fila cheia, mas não excedida)")
	}
	dropped, _ = q.Enqueue(sample{Value: 3})
	if !dropped {
		t.Fatal("esperava dropped=true ao exceder o limite da fila")
	}
}

func TestFileQueue_PersistsAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.jsonl")

	q1 := NewFileQueue[sample](path, 100)
	q1.Enqueue(sample{Value: 42})

	// Simula reinício do processo: nova instância apontando para o mesmo arquivo.
	q2 := NewFileQueue[sample](path, 100)
	items, err := q2.Peek()
	if err != nil {
		t.Fatalf("erro ao ler fila persistida: %v", err)
	}
	if len(items) != 1 || items[0].Value != 42 {
		t.Fatalf("esperava recuperar o item enfileirado antes do 'reinício', encontrou %+v", items)
	}
}
