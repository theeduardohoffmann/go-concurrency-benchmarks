# T1 - Problemas Clássicos de Concorrência

FPPD - Fundamentos de Processamento Paralelo e Distribuído (98713-04)

Alunos: Eduardo Hoffmann e Lucas Mocelin

Dois programas em Go, independentes entre si, cada um com seu `go.mod` e um único `main.go`:

| Pasta | Problema |
|---|---|
| [`filosofos/`](filosofos) | Jantar dos Filósofos |
| [`produtor-consumidor/`](produtor-consumidor) | Produtor/Consumidor com buffer limitado |
| [`relatorio/relatorio.pdf`](relatorio/relatorio.pdf) | Relatório com as análises e os dados coletados |

## Requisitos

- Go 1.22 ou mais novo.
- Para usar `-race` no Windows é preciso um gcc de 64 bits no PATH (por exemplo, WinLibs ou MSYS2).

## Problema 1: Jantar dos Filósofos

Os N filósofos ficam numa mesa circular e cada garfo é um channel de capacidade 1 (token no channel = garfo livre).

| Versão | Como funciona | Deadlock |
|---|---|---|
| `base` | todos pegam o garfo da esquerda e depois o da direita | acontece |
| `hierarquia` | os garfos são numerados e todos pegam primeiro o de menor número | não acontece |
| `garcom` | uma goroutine garçom recebe os pedidos em fila e entrega os dois garfos de uma vez | não acontece |

R é o número total de refeições da mesa: cada filósofo come o quanto conseguir, e assim a distribuição das refeições mostra se a estratégia é justa. O deadlock é detectado quando um filósofo espera mais de 1 segundo por um garfo (`select` com `time.After`); nesse caso ele desiste e o programa termina normalmente. No fim, o programa espera todas as goroutines antes de imprimir os dados.

### Compilar e executar

```bash
cd filosofos
go build
go run -race .
```

| Opção | Significado | Padrão |
|---|---|---|
| `-n` | N: número de filósofos (mínimo 2) | 5 |
| `-r` | R: número total de refeições da mesa | 300 |
| `-estrategia` | `base`, `hierarquia`, `garcom` ou `todas` | `todas` |

Para cada versão o programa mostra as refeições e a espera média pelos garfos (em ms) de cada filósofo, o total de refeições e se houve deadlock. Os dados do relatório foram coletados com `-r 1000`.

## Problema 2: Produtor/Consumidor com buffer limitado

P produtores e C consumidores compartilham um buffer de capacidade K. O produtor bloqueia com o buffer cheio e o consumidor bloqueia com o buffer vazio.

| Versão | Como funciona |
|---|---|
| `canal` | o buffer é um channel com capacidade K |
| `semaforo` | o buffer é um slice circular protegido pelos semáforos de contagem `notFull` e `notEmpty` e por dois mutexes, um para a posição de escrita e outro para a de leitura |

- **Encerramento:** quando os produtores terminam, a main coloca uma "pílula" (`-1`) no buffer para cada consumidor. Como o buffer é FIFO, a pílula fica atrás de todos os itens, então os consumidores esvaziam o buffer antes de parar.
- **Timeout:** o consumidor 0 usa `select` com `time.After`. Se nenhum item chega no intervalo, ele conta o timeout, avisa e tenta de novo.
- **Verificação:** cada item tem um identificador único. No fim, o programa confere que o total consumido é igual ao produzido e que nenhum item foi perdido ou repetido (`ok=true`).

### Compilar e executar

```bash
cd produtor-consumidor
go build
go run -race .
```

| Opção | Significado | Padrão |
|---|---|---|
| `-p`, `-c` | P e C: número de produtores e de consumidores | 4 e 4 |
| `-k` | K: capacidade do buffer (0 = testa K = 1, 10 e 100) | 0 |
| `-itens` | itens produzidos por cada produtor | 500 |
| `-prod`, `-cons` | tempo médio de trabalho por item (ex.: `500us`, `0s`) | `500us` |
| `-timeout` | timeout do consumidor 0 | `5ms` |

Cada linha da saída mostra a versão, K, itens por segundo, ocupação média do buffer, timeouts do consumidor 0, itens consumidos por cada consumidor, o total produzido e consumido e o resultado da verificação. Os dados do relatório foram coletados com `-itens 2000`.

Exemplo que força timeouts do consumidor 0 (produtores lentos, consumidores rápidos):

```bash
go run -race . -k 10 -itens 100 -prod 3ms -cons 0s -timeout 2ms
```

Exemplo sem trabalho simulado (mostra a diferença entre channel e semáforos):

```bash
go run -race . -k 10 -prod 0s -cons 0s
```
