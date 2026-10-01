# T1 - Problemas Clássicos de Concorrência (FPPD 98713-04)

Alunos: Eduardo Hoffmann e Lucas Mocelin

Dois programas Go independentes, cada um com seu `go.mod`:

| Pasta | Problema |
|---|---|
| [`filosofos/`](filosofos) | Jantar dos Filósofos: versão base (deadlock) e estratégias hierarquia de recursos e garçom |
| [`produtor-consumidor/`](produtor-consumidor) | Produtor/Consumidor com buffer limitado: versão com channel e versão com semáforos |
| [`relatorio/relatorio.pdf`](relatorio/relatorio.pdf) | Relatório |

## Requisitos

- Go 1.22 ou mais novo.
- Para usar `-race` no Windows é preciso um gcc de 64 bits no PATH (por exemplo, WinLibs ou MSYS2).

## Jantar dos Filósofos

```bash
cd filosofos
go run -race . -n 5 -r 1000
```

| Opção | Significado |
|---|---|
| `-n` | N: número de filósofos (mínimo 2). Padrão: 5 |
| `-r` | R: número total de refeições da mesa. Padrão: 1000 |
| `-estrategia` | `base`, `hierarquia`, `garcom` ou `todas` (padrão) |

Para cada estratégia o programa imprime as refeições e a espera média pelos garfos de cada filósofo. A versão `base` termina com a mensagem de DEADLOCK, que é o comportamento esperado.

## Produtor/Consumidor

```bash
cd produtor-consumidor
go run -race . -p 4 -c 4 -k 1,10,100
```

| Opção | Significado |
|---|---|
| `-p`, `-c` | número de produtores e de consumidores. Padrão: 4 e 4 |
| `-k` | capacidades do buffer, separadas por vírgula. Padrão: `1,10,100` |
| `-itens` | itens produzidos por cada produtor. Padrão: 2000 |
| `-prod`, `-cons` | tempo médio de trabalho por item (ex.: `500us`, `0s`). Padrão: `500us` |
| `-timeout` | timeout do consumidor 0 (`select` com `time.After`). Padrão: `5ms` |

Cada linha mostra a versão (channel ou semáforos), K, itens por segundo, ocupação média do buffer, timeouts do consumidor 0 e itens consumidos por consumidor. Ela termina com `OK` quando o total produzido é igual ao consumido e nenhum item foi perdido ou repetido.

Exemplo que força timeouts do consumidor 0:

```bash
go run -race . -k 10 -itens 500 -prod 3ms -cons 0s -timeout 2ms
```
