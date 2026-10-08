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
	n    = flag.Int("n", 5, "N: número de filósofos")
	r    = flag.Int("r", 300, "R: número total de refeições da mesa")
	estr = flag.String("estrategia", "todas", "base | hierarquia | garcom | todas")
)

const (
	pensar  = 2 * time.Millisecond
	comer   = 2 * time.Millisecond
	segurar = 2 * time.Millisecond
	timeout = time.Second
)

type mesa struct {
	estrategia string
	garfos     []chan struct{}
	pedidos    chan int
	ok         []chan struct{}
	restantes  atomic.Int64
	deadlock   atomic.Bool
	refeicoes  []int
	espera     []time.Duration
}

func dorme(max time.Duration) {
	time.Sleep(time.Duration(rand.Int63n(int64(max))))
}

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

func (m *mesa) filosofo(id int, wg *sync.WaitGroup) {
	defer wg.Done()
	primeiro, segundo := id, (id+1)%*n
	if m.estrategia == "hierarquia" && segundo < primeiro {
		primeiro, segundo = segundo, primeiro
	}
	for {
		dorme(pensar)
		if m.restantes.Add(-1) < 0 {
			return
		}
		inicio := time.Now()
		if m.estrategia == "garcom" {
			m.pedidos <- id
			<-m.ok[id]
		} else {
			if !m.pegar(id, primeiro) {
				return
			}
			time.Sleep(segurar)
			if !m.pegar(id, segundo) {
				return
			}
		}
		m.espera[id] += time.Since(inicio)
		m.refeicoes[id]++
		dorme(comer)
		m.garfos[primeiro] <- struct{}{}
		m.garfos[segundo] <- struct{}{}
	}
}

func (m *mesa) garcom(fim chan<- struct{}) {
	for id := range m.pedidos {
		<-m.garfos[id]
		<-m.garfos[(id+1)%*n]
		m.ok[id] <- struct{}{}
	}
	close(fim)
}

func executar(estrategia string) {
	m := &mesa{estrategia: estrategia, pedidos: make(chan int), garfos: make([]chan struct{}, *n),
		ok: make([]chan struct{}, *n), refeicoes: make([]int, *n), espera: make([]time.Duration, *n)}
	for i := range m.garfos {
		m.garfos[i] = make(chan struct{}, 1)
		m.garfos[i] <- struct{}{}
		m.ok[i] = make(chan struct{})
	}
	m.restantes.Store(int64(*r))

	fmt.Printf("\n=== %s ===\n", estrategia)
	garcomFim := make(chan struct{})
	if estrategia == "garcom" {
		go m.garcom(garcomFim)
	}
	var wg sync.WaitGroup
	for i := 0; i < *n; i++ {
		wg.Add(1)
		go m.filosofo(i, &wg)
	}
	wg.Wait()
	if estrategia == "garcom" {
		close(m.pedidos)
		<-garcomFim
	}

	fmt.Println("filósofo  refeições  espera média (ms)")
	total := 0
	for i := range m.refeicoes {
		media := 0.0
		if m.refeicoes[i] > 0 {
			media = float64(m.espera[i]) / float64(m.refeicoes[i]) / 1e6
		}
		fmt.Printf("%8d  %9d  %17.2f\n", i, m.refeicoes[i], media)
		total += m.refeicoes[i]
	}
	if m.deadlock.Load() {
		fmt.Printf("total: %d refeições -> DEADLOCK\n", total)
	} else {
		fmt.Printf("total: %d refeições -> sem deadlock\n", total)
	}
}

func main() {
	flag.Parse()
	lista := []string{"base", "hierarquia", "garcom"}
	switch *estr {
	case "base", "hierarquia", "garcom":
		lista = []string{*estr}
	case "todas":
	default:
		fmt.Println("estrategia deve ser base, hierarquia, garcom ou todas")
		return
	}
	if *n < 2 || *r < 1 {
		fmt.Println("use n >= 2 e r >= 1")
		return
	}
	for _, e := range lista {
		executar(e)
	}
}
