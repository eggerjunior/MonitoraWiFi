package rdap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// URLs oficiais de bootstrap da IANA (RFC 7484) — ver
// https://www.iana.org/assignments/rdap-dns-registry.
const (
	defaultDNSBootstrapURL  = "https://data.iana.org/rdap/dns.json"
	defaultIPv4BootstrapURL = "https://data.iana.org/rdap/ipv4.json"
	defaultIPv6BootstrapURL = "https://data.iana.org/rdap/ipv6.json"
)

// ErrNoServer é retornado quando nenhum servidor RDAP autoritativo é
// encontrado no bootstrap para a consulta (TLD ou bloco IP desconhecidos).
var ErrNoServer = errors.New("nenhum servidor RDAP encontrado para esta consulta")

// Event é um evento do ciclo de vida do objeto (registro, expiração, última
// atualização) — nomes seguem o vocabulário RDAP (RFC 7483 §4.5).
type Event struct {
	Action string `json:"action"`
	Date   string `json:"date"`
}

// Result é a resposta RDAP normalizada — os campos "brutos" variam entre
// domain/ip network/autnum, então também devolvemos o JSON completo em Raw
// para qualquer informação não coberta pelos campos normalizados.
type Result struct {
	Query           string          `json:"query"`
	Server          string          `json:"server"`
	ObjectClassName string          `json:"object_class_name"`
	Handle          string          `json:"handle"`
	Name            string          `json:"name"`
	Status          []string        `json:"status"`
	Events          []Event         `json:"events"`
	Nameservers     []string        `json:"nameservers,omitempty"`
	Raw             json.RawMessage `json:"raw"`
}

type rdapObject struct {
	ObjectClassName string   `json:"objectClassName"`
	Handle          string   `json:"handle"`
	LdhName         string   `json:"ldhName"`
	Name            string   `json:"name"`
	Status          []string `json:"status"`
	Events          []struct {
		EventAction string `json:"eventAction"`
		EventDate   string `json:"eventDate"`
	} `json:"events"`
	Nameservers []struct {
		LdhName string `json:"ldhName"`
	} `json:"nameservers"`
}

// Client consulta RDAP para domínios e IPs, resolvendo o servidor
// autoritativo via bootstrap da IANA antes de cada consulta (com cache).
type Client struct {
	httpClient *http.Client
	dns        *bootstrapSource
	ipv4       *bootstrapSource
	ipv6       *bootstrapSource
}

// NewClient cria um cliente RDAP contra o registro de bootstrap real da
// IANA. Use NewClientWithBootstrapURLs em testes para apontar pra
// servidores locais.
func NewClient() *Client {
	return NewClientWithBootstrapURLs(defaultDNSBootstrapURL, defaultIPv4BootstrapURL, defaultIPv6BootstrapURL)
}

func NewClientWithBootstrapURLs(dnsURL, ipv4URL, ipv6URL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		dns:        newBootstrapSource(dnsURL),
		ipv4:       newBootstrapSource(ipv4URL),
		ipv6:       newBootstrapSource(ipv6URL),
	}
}

// Lookup consulta RDAP para `query` — detecta automaticamente se é um IP ou
// um nome de domínio, nunca adivinha o servidor: sempre resolve via
// bootstrap real antes de consultar.
func (c *Client) Lookup(ctx context.Context, query string) (Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return Result{}, errors.New("consulta vazia")
	}

	if ip := net.ParseIP(query); ip != nil {
		return c.lookupIP(ctx, query, ip)
	}
	return c.lookupDomain(ctx, query)
}

func (c *Client) lookupDomain(ctx context.Context, domain string) (Result, error) {
	labels := strings.Split(strings.Trim(domain, "."), ".")
	tld := labels[len(labels)-1]

	services, err := c.dns.resolve(ctx, c.httpClient)
	if err != nil {
		return Result{}, err
	}
	base, ok := serverForTLD(services, tld)
	if !ok {
		return Result{}, fmt.Errorf("%w: TLD .%s", ErrNoServer, tld)
	}

	return c.query(ctx, base, "domain", domain)
}

func (c *Client) lookupIP(ctx context.Context, query string, ip net.IP) (Result, error) {
	source := c.ipv4
	if ip.To4() == nil {
		source = c.ipv6
	}

	services, err := source.resolve(ctx, c.httpClient)
	if err != nil {
		return Result{}, err
	}
	base, ok := serverForIP(services, ip)
	if !ok {
		return Result{}, fmt.Errorf("%w: IP %s", ErrNoServer, query)
	}

	return c.query(ctx, base, "ip", query)
}

func (c *Client) query(ctx context.Context, base, kind, value string) (Result, error) {
	url := strings.TrimRight(base, "/") + "/" + kind + "/" + value

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/rdap+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("erro ao consultar servidor RDAP (%s): %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Result{}, fmt.Errorf("%s não encontrado no servidor RDAP (%s)", value, base)
	}
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("servidor RDAP (%s) retornou status %d", url, resp.StatusCode)
	}

	raw, err := jsonDecodeRaw(resp)
	if err != nil {
		return Result{}, fmt.Errorf("resposta RDAP (%s) com formato inválido: %w", url, err)
	}

	var obj rdapObject
	if err := json.Unmarshal(raw, &obj); err != nil {
		return Result{}, fmt.Errorf("resposta RDAP (%s) com formato inválido: %w", url, err)
	}

	name := obj.LdhName
	if name == "" {
		name = obj.Name
	}

	events := make([]Event, 0, len(obj.Events))
	for _, e := range obj.Events {
		events = append(events, Event{Action: e.EventAction, Date: e.EventDate})
	}

	nameservers := make([]string, 0, len(obj.Nameservers))
	for _, ns := range obj.Nameservers {
		nameservers = append(nameservers, ns.LdhName)
	}

	return Result{
		Query:           value,
		Server:          base,
		ObjectClassName: obj.ObjectClassName,
		Handle:          obj.Handle,
		Name:            name,
		Status:          obj.Status,
		Events:          events,
		Nameservers:     nameservers,
		Raw:             raw,
	}, nil
}

func jsonDecodeRaw(resp *http.Response) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}
