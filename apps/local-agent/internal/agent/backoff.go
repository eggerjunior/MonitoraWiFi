package agent

import "time"

// Backoff exponencial simples com teto — usado quando o backend está
// inalcançável (Seção 3: "reconexão com exponential backoff"). Reseta para
// o valor inicial assim que um envio tem sucesso.
type Backoff struct {
	initial time.Duration
	max     time.Duration
	current time.Duration
}

func NewBackoff(initial, max time.Duration) *Backoff {
	return &Backoff{initial: initial, max: max, current: initial}
}

func (b *Backoff) Next() time.Duration {
	d := b.current
	b.current *= 2
	if b.current > b.max {
		b.current = b.max
	}
	return d
}

func (b *Backoff) Reset() {
	b.current = b.initial
}
