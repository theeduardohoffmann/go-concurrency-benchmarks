package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const fim = -1

type buffer interface {
	put(x int)
	get() int
	getTimeout(d time.Duration) (int, bool)
	tamanho() int
}

type bufCanal chan int

func (b bufCanal) put(x int)    { b <- x }
func (b bufCanal) get() int     { return <-b }
func (b bufCanal) tamanho() int { return len(b) }

func (b bufCanal) getTimeout(d time.Duration) (int, bool) {
	select {
	case x := <-b:
		return x, true
	case <-time.After(d):
		return 0, false
	}
}

type sem chan struct{}

func novoSem(max, inicial int) sem {
	s := make(sem, max)
	for i := 0; i < inicial; i++ {
		s.V()
	}
	return s
}

func (s sem) P() { <-s }
func (s sem) V() { s <- struct{}{} }

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

func (b *bufSem) retirar() int {
	b.mSai.Lock()
	x := b.itens[b.sai]
	b.sai = (b.sai + 1) % len(b.itens)
	b.mSai.Unlock()
	b.notFull.V()
	return x
}

func (b *bufSem) tamanho() int { return len(b.notEmpty) }

func trabalho(media time.Duration) {
	if media > 0 {
		time.Sleep(time.Duration(rand.Int63n(int64(2 * media))))
	}
}

func rodar(nome string, b buffer, k, p, c, itens int, prod, cons, tmo time.Duration) {
	var prodWG, consWG sync.WaitGroup
	consumido := make([][]int, c)
	timeouts := 0

	parar, media := make(chan struct{}), make(chan float64, 1)
	go func() {
		var soma, fotos float64
		for tick := time.NewTicker(200 * time.Microsecond); ; {
			select {
			case <-tick.C:
				soma += float64(b.tamanho())
				fotos++
			case <-parar:
				media <- soma / max(fotos, 1)
				return
			}
		}
	}()

	inicio := time.Now()
	for i := 0; i < p; i++ {
		prodWG.Add(1)
		go func() {
			defer prodWG.Done()
			for s := 0; s < itens; s++ {
				trabalho(prod)
				b.put(i*itens + s)
			}
		}()
	}
	for i := 0; i < c; i++ {
		consWG.Add(1)
		go func() {
			defer consWG.Done()
			for {
				x, chegou := 0, true
				if i == 0 {
					x, chegou = b.getTimeout(tmo)
				} else {
					x = b.get()
				}
				if !chegou {
					if timeouts++; timeouts <= 3 {
						fmt.Printf("  consumidor 0: nenhum item em %v, executando tarefa alternativa\n", tmo)
					}
					continue
				}
				if x == fim {
					return
				}
				consumido[i] = append(consumido[i], x)
				trabalho(cons)
			}
		}()
	}

	prodWG.Wait()
	for i := 0; i < c; i++ {
		b.put(fim)
	}
	consWG.Wait()
	duracao := time.Since(inicio)
	close(parar)

	vistos, total, ok := make([]bool, p*itens), 0, true
	porConsumidor := make([]int, c)
	for i, lista := range consumido {
		porConsumidor[i] = len(lista)
		for _, x := range lista {
			ok = ok && !vistos[x]
			vistos[x] = true
			total++
		}
	}
	ok = ok && total == p*itens
	fmt.Printf("%-8s K=%-3d %8.0f itens/s  ocupação=%5.2f  timeouts=%-3d  por consumidor=%v  produzidos=%d consumidos=%d ok=%v\n",
		nome, k, float64(total)/duracao.Seconds(), <-media, timeouts, porConsumidor, p*itens, total, ok)
}

func main() {
	p := flag.Int("p", 4, "P: número de produtores")
	c := flag.Int("c", 4, "C: número de consumidores")
	k := flag.Int("k", 0, "K: capacidade do buffer (0 = testa 1, 10 e 100)")
	itens := flag.Int("itens", 500, "itens produzidos por cada produtor")
	prod := flag.Duration("prod", 500*time.Microsecond, "tempo médio para produzir um item")
	cons := flag.Duration("cons", 500*time.Microsecond, "tempo médio para consumir um item")
	tmo := flag.Duration("timeout", 5*time.Millisecond, "timeout do consumidor 0")
	flag.Parse()
	if *p < 1 || *c < 1 || *k < 0 || *itens < 1 {
		fmt.Println("use p >= 1, c >= 1, k >= 0 e itens >= 1")
		return
	}

	capacidades := []int{1, 10, 100}
	if *k > 0 {
		capacidades = []int{*k}
	}
	for _, cap := range capacidades {
		rodar("canal", make(bufCanal, cap), cap, *p, *c, *itens, *prod, *cons, *tmo)
		rodar("semaforo", novoBufSem(cap), cap, *p, *c, *itens, *prod, *cons, *tmo)
	}
}
