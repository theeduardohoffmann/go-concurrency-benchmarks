// Jantar dos Filósofos com garfos representados por channels.
//
// Uso: go run -race . -n 5 -r 1000 -estrategia todas
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var (
	n    = flag.Int("n", 5, "N: número de filósofos (>= 2)")
	r    = flag.Int("r", 1000, "R: número total de refeições da mesa")
	estr = flag.String("estrategia", "todas", "base | hierarquia | garcom | todas")
)

const (
	pensar  = 2 * time.Millisecond // tempo máximo pensando
	comer   = 2 * time.Millisecond // tempo máximo comendo
	segurar = 2 * time.Millisecond // pausa entre pegar o 1º e o 2º garfo (faz o deadlock da base acontecer)
	timeout = time.Second          // esperar um garfo por mais que isso = deadlock
)

// pedido enviado ao garçom; o garçom responde em ok quando os dois garfos estão com o filósofo.
type pedido struct {
	id int
	ok chan struct{}
}

type mesa struct {
	estrategia string
	garfos     []chan struct{} // garfo livre = há um token no channel (capacidade 1)
	pedidos    chan pedido     // estratégia "garcom"
	restantes  atomic.Int64    // refeições que ainda podem ser feitas
	deadlock   atomic.Bool
	refeicoes  []int           // cada filósofo só escreve na sua posição
	espera     []time.Duration // soma dos tempos de espera pelos garfos
}

func dorme(max time.Duration) {
	if max > 0 {
		time.Sleep(time.Duration(rand.Int63n(int64(max))))
	}
}

// pegar retira o garfo g (receber do channel). Se esperar demais, é deadlock.
func (m *mesa) pegar(id, g int) bool {
	select {
	case <-m.garfos[g]:
		return true
	case <-time.After(timeout):
		fmt.Printf("  filósofo %d esperou mais de %v pelo garfo %d: DEADLOCK\n", id, timeout, g)
		m.deadlock.Store(true)
		return false
	}
}

func (m *mesa) soltar(g int) { m.garfos[g] <- struct{}{} }

func (m *mesa) filosofo(id int, wg *sync.WaitGroup) {
	defer wg.Done()
	primeiro, segundo := id, (id+1)%*n // esquerda e direita
	if m.estrategia == "hierarquia" && segundo < primeiro {
		primeiro, segundo = segundo, primeiro // sempre o garfo de menor número primeiro
	}
	for {
		dorme(pensar)
		if m.restantes.Add(-1) < 0 { // acabaram as R refeições
			return
		}

		inicio := time.Now()
		switch m.estrategia {
		case "garcom":
			p := pedido{id, make(chan struct{})}
			m.pedidos <- p
			<-p.ok // o garçom já retirou os dois garfos para nós
		default:
			if !m.pegar(id, primeiro) {
				return
			}
			dorme(segurar)
			if !m.pegar(id, segundo) {
				return
			}
		}
		m.espera[id] += time.Since(inicio)

		m.refeicoes[id]++
		dorme(comer)

		m.soltar(primeiro)
		m.soltar(segundo)
	}
}

// garcom atende os pedidos em ordem de chegada (FIFO). Só ele retira garfos nesta estratégia,
// então não há espera circular; se um garfo está ocupado, quem o usa vai devolvê-lo.
func (m *mesa) garcom(fim chan<- struct{}) {
	for p := range m.pedidos {
		<-m.garfos[p.id]
		<-m.garfos[(p.id+1)%*n]
		p.ok <- struct{}{}
	}
	close(fim)
}

func executar(estrategia string) {
	m := &mesa{estrategia: estrategia, garfos: make([]chan struct{}, *n),
		refeicoes: make([]int, *n), espera: make([]time.Duration, *n)}
	for i := range m.garfos {
		m.garfos[i] = make(chan struct{}, 1)
		m.garfos[i] <- struct{}{}
	}
	m.pedidos = make(chan pedido)
	m.restantes.Store(int64(*r))

	fmt.Printf("\n=== estratégia: %s ===\n", estrategia)
	garcomFim := make(chan struct{})
	if estrategia == "garcom" {
		go m.garcom(garcomFim)
	}
	var wg sync.WaitGroup
	for i := 0; i < *n; i++ {
		wg.Add(1)
		go m.filosofo(i, &wg)
	}
	wg.Wait() // encerramento: todos os filósofos terminaram (ou detectaram deadlock)
	if estrategia == "garcom" {
		close(m.pedidos) // o garçom sai do range e termina
		<-garcomFim
	}

	fmt.Printf("%-10s %10s %20s\n", "filósofo", "refeições", "espera média (ms)")
	total := 0
	for i := 0; i < *n; i++ {
		media := 0.0
		if m.refeicoes[i] > 0 {
			media = float64(m.espera[i]) / float64(m.refeicoes[i]) / 1e6
		}
		fmt.Printf("%-10d %10d %20.2f\n", i, m.refeicoes[i], media)
		total += m.refeicoes[i]
	}
	fmt.Printf("total: %d refeições\n", total)
	if m.deadlock.Load() {
		fmt.Println("RESULTADO: deadlock detectado (todos os filósofos ficaram bloqueados).")
	} else {
		fmt.Println("RESULTADO: sem deadlock; todas as goroutines terminaram.")
	}
}

func main() {
	flag.Parse()
	if *n < 2 || *r < 1 {
		fmt.Println("use n >= 2 e r >= 1")
		return
	}
	lista := []string{"base", "hierarquia", "garcom"}
	if *estr != "todas" {
		if *estr != "base" && *estr != "hierarquia" && *estr != "garcom" {
			fmt.Println("estratégia inválida:", *estr)
			return
		}
		lista = []string{*estr}
	}
	fmt.Printf("N=%d R=%d\n", *n, *r)
	for _, e := range lista {
		executar(e)
	}
}
