// Produtor/Consumidor com buffer limitado: versão com channel e versão com semáforos.
//
// Uso: go run -race . -p 4 -c 4 -k 1,10,100
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

const fim = -1 // "pílula" de encerramento: um item especial que manda o consumidor parar

// buffer: as duas versões implementam esta interface.
type buffer interface {
	put(x int)
	get() int
	getTimeout(d time.Duration) (int, bool) // false = nenhum item chegou dentro de d
	tamanho() int
}

// ---------------------------------------------------------------- versão 1: channel
// O buffer é o próprio channel: Go bloqueia o produtor se estiver cheio e o consumidor se vazio.

type bufCanal struct{ ch chan int }

func (b *bufCanal) put(x int) { b.ch <- x }
func (b *bufCanal) get() int  { return <-b.ch }
func (b *bufCanal) tamanho() int {
	return len(b.ch)
}
func (b *bufCanal) getTimeout(d time.Duration) (int, bool) {
	select {
	case x := <-b.ch:
		return x, true
	case <-time.After(d):
		return 0, false
	}
}

// ---------------------------------------------------------------- versão 2: semáforos
// Semáforo de contagem: o número de tokens no channel é o valor do semáforo.
// P (wait) retira um token e bloqueia se não houver; V (signal) devolve um token.
type sem chan struct{}

func novoSem(capacidade, inicial int) sem {
	s := make(sem, capacidade)
	for i := 0; i < inicial; i++ {
		s <- struct{}{}
	}
	return s
}
func (s sem) P() { <-s }
func (s sem) V() { s <- struct{}{} }

// bufSem é o algoritmo clássico:
//
//	put: P(notFull);  Lock(mEntra); itens[entra]=x; entra++; Unlock(mEntra); V(notEmpty)
//	get: P(notEmpty); Lock(mSai);   x=itens[sai];   sai++;   Unlock(mSai);   V(notFull)
//
// notFull conta posições livres (começa em K) e notEmpty conta itens (começa em 0).
// Só os semáforos não bastam com vários produtores/consumidores: sem mEntra dois produtores
// escreveriam na mesma posição; sem mSai dois consumidores retirariam o mesmo item.
type bufSem struct {
	itens             []int
	entra, sai        int
	mEntra, mSai      sync.Mutex
	notFull, notEmpty sem
}

func novoBufSem(k int) *bufSem {
	return &bufSem{itens: make([]int, k), notFull: novoSem(k, k), notEmpty: novoSem(k, 0)}
}

func (b *bufSem) put(x int) {
	b.notFull.P()
	b.mEntra.Lock()
	b.itens[b.entra] = x
	b.entra = (b.entra + 1) % len(b.itens)
	b.mEntra.Unlock()
	b.notEmpty.V()
}

func (b *bufSem) retirar() int { // já temos um token de notEmpty
	b.mSai.Lock()
	x := b.itens[b.sai]
	b.sai = (b.sai + 1) % len(b.itens)
	b.mSai.Unlock()
	b.notFull.V()
	return x
}

func (b *bufSem) get() int {
	b.notEmpty.P()
	return b.retirar()
}

func (b *bufSem) getTimeout(d time.Duration) (int, bool) {
	select {
	case <-b.notEmpty:
		return b.retirar(), true
	case <-time.After(d):
		return 0, false
	}
}

func (b *bufSem) tamanho() int { return len(b.notEmpty) }

// ---------------------------------------------------------------- simulação

func trabalho(media time.Duration) {
	if media > 0 {
		time.Sleep(time.Duration(rand.Int63n(int64(2 * media))))
	}
}

func rodar(versao string, k, p, c, itens int, prod, cons, tmo time.Duration) {
	var b buffer
	if versao == "canal" {
		b = &bufCanal{make(chan int, k)}
	} else {
		b = novoBufSem(k)
	}

	var prodWG, consWG sync.WaitGroup
	consumidos := make([][]int, c) // cada consumidor só escreve na sua posição
	timeouts := 0                  // só o consumidor 0 escreve

	// amostrador: mede a ocupação do buffer a cada 200µs
	parar, amostradorFim := make(chan struct{}), make(chan struct{})
	var soma, amostras float64
	go func() {
		defer close(amostradorFim)
		t := time.NewTicker(200 * time.Microsecond)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				soma += float64(b.tamanho())
				amostras++
			case <-parar:
				return
			}
		}
	}()

	inicio := time.Now()
	for i := 0; i < p; i++ {
		prodWG.Add(1)
		go func(i int) {
			defer prodWG.Done()
			for s := 0; s < itens; s++ {
				trabalho(prod)
				b.put(i*itens + s) // identificador único do item
			}
		}(i)
	}
	for i := 0; i < c; i++ {
		consWG.Add(1)
		go func(i int) {
			defer consWG.Done()
			for {
				var x int
				if i == 0 && tmo > 0 { // consumidor 0: select + time.After
					var ok bool
					x, ok = b.getTimeout(tmo)
					if !ok { // comportamento alternativo: nenhum item chegou a tempo
						timeouts++
						if timeouts <= 3 {
							fmt.Printf("    consumidor 0: nenhum item em %v, executando tarefa alternativa\n", tmo)
						}
						continue
					}
				} else {
					x = b.get()
				}
				if x == fim {
					return
				}
				consumidos[i] = append(consumidos[i], x)
				trabalho(cons)
			}
		}(i)
	}

	// Encerramento: quando os produtores terminam, coloca uma pílula por consumidor no buffer.
	// O buffer é FIFO, então as pílulas ficam atrás de todos os itens: cada consumidor só
	// encerra depois que os itens restantes foram retirados.
	prodWG.Wait()
	for i := 0; i < c; i++ {
		b.put(fim)
	}
	consWG.Wait()
	dur := time.Since(inicio)
	close(parar)
	<-amostradorFim

	// verificação: total produzido == total consumido e cada item exatamente uma vez
	vistos := make([]bool, p*itens)
	total, ok := 0, true
	var porConsumidor []string
	for _, lista := range consumidos {
		porConsumidor = append(porConsumidor, strconv.Itoa(len(lista)))
		for _, x := range lista {
			if vistos[x] {
				ok = false // consumido duas vezes
			}
			vistos[x] = true
			total++
		}
	}
	if total != p*itens {
		ok = false // algum item foi perdido
	}
	fmt.Printf("%-9s %4d %13.0f %11.2f %9d  [%s]  produzidos=%d consumidos=%d %s\n",
		versao, k, float64(total)/dur.Seconds(), soma/max(amostras, 1), timeouts,
		strings.Join(porConsumidor, " "), p*itens, total, map[bool]string{true: "OK", false: "ERRO!"}[ok])
}

func main() {
	pf := flag.Int("p", 4, "número de produtores")
	cf := flag.Int("c", 4, "número de consumidores")
	ks := flag.String("k", "1,10,100", "capacidades do buffer, separadas por vírgula")
	itens := flag.Int("itens", 2000, "itens produzidos por cada produtor")
	prod := flag.Duration("prod", 500*time.Microsecond, "tempo médio para produzir um item")
	cons := flag.Duration("cons", 500*time.Microsecond, "tempo médio para consumir um item")
	tmo := flag.Duration("timeout", 5*time.Millisecond, "timeout do consumidor 0 (0 desliga)")
	flag.Parse()
	if *pf < 1 || *cf < 1 || *itens < 1 {
		fmt.Println("use p >= 1, c >= 1 e itens >= 1")
		return
	}

	fmt.Printf("P=%d C=%d itens/produtor=%d prod=%v cons=%v timeout=%v\n", *pf, *cf, *itens, *prod, *cons, *tmo)
	fmt.Printf("%-9s %4s %13s %11s %9s  %s\n", "versão", "K", "itens/s", "ocup. média", "timeouts", "itens por consumidor")
	for _, s := range strings.Split(*ks, ",") {
		k, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || k < 1 {
			fmt.Println("capacidade inválida:", s)
			return
		}
		for _, v := range []string{"canal", "semaforo"} {
			rodar(v, k, *pf, *cf, *itens, *prod, *cons, *tmo)
		}
	}
}
