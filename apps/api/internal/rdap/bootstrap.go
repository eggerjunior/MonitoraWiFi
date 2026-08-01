// Package rdap implementa consultas RDAP (RFC 7482/7483) para domínios e
// IPs — a ferramenta "RDAP/WHOIS" da Fase 5. Roda direto no backend, sem
// envolver o agente local, porque a informação é pública na internet (não
// depende de acesso à LAN do usuário).
package rdap

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// bootstrapSource resolve, a partir de uma consulta (TLD ou IP), qual
// servidor RDAP autoritativo consultar — segue o registro de bootstrap da
// IANA (RFC 7484), nunca um servidor fixo escolhido a dedo.
type bootstrapSource struct {
	url string

	mu        sync.Mutex
	services  []bootstrapService
	fetchedAt time.Time
}

type bootstrapService struct {
	keys []string
	urls []string
}

type rawBootstrapDoc struct {
	Services [][]json.RawMessage `json:"services"`
}

func newBootstrapSource(url string) *bootstrapSource {
	return &bootstrapSource{url: url}
}

// resolve retorna a lista de serviços do bootstrap, buscando (ou reusando
// do cache em memória, válido por 24h — o registro da IANA muda raramente)
// a partir da URL real da IANA.
func (b *bootstrapSource) resolve(ctx context.Context, httpClient *http.Client) ([]bootstrapService, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.services != nil && time.Since(b.fetchedAt) < 24*time.Hour {
		return b.services, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar registro de bootstrap RDAP (%s): %w", b.url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registro de bootstrap RDAP (%s) retornou status %d", b.url, resp.StatusCode)
	}

	var doc rawBootstrapDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("registro de bootstrap RDAP (%s) com formato inválido: %w", b.url, err)
	}

	services := make([]bootstrapService, 0, len(doc.Services))
	for _, entry := range doc.Services {
		if len(entry) != 2 {
			continue
		}
		var keys, urls []string
		if err := json.Unmarshal(entry[0], &keys); err != nil {
			continue
		}
		if err := json.Unmarshal(entry[1], &urls); err != nil {
			continue
		}
		services = append(services, bootstrapService{keys: keys, urls: urls})
	}

	b.services = services
	b.fetchedAt = time.Now()
	return services, nil
}

// serverForTLD retorna a URL base do servidor RDAP autoritativo para o TLD
// informado (ex.: "com", "br"), comparando sem diferenciar maiúsculas.
func serverForTLD(services []bootstrapService, tld string) (string, bool) {
	tld = strings.ToLower(tld)
	for _, svc := range services {
		for _, key := range svc.keys {
			if strings.ToLower(key) == tld && len(svc.urls) > 0 {
				return svc.urls[0], true
			}
		}
	}
	return "", false
}

// serverForIP retorna a URL base do servidor RDAP autoritativo (RIR) cujo
// bloco CIDR contém o IP informado.
func serverForIP(services []bootstrapService, ip net.IP) (string, bool) {
	for _, svc := range services {
		for _, key := range svc.keys {
			_, cidr, err := net.ParseCIDR(key)
			if err != nil {
				continue
			}
			if cidr.Contains(ip) && len(svc.urls) > 0 {
				return svc.urls[0], true
			}
		}
	}
	return "", false
}
