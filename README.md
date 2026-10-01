# T1 - Problemas Clássicos de Concorrência (FPPD 98713-04)

Alunos: Eduardo Hoffmann e Lucas Mocelin

Dois programas Go independentes (cada pasta tem seu `go.mod` e um único `main.go`):

| Pasta | Problema |
|---|---|
| [`filosofos/`](filosofos) | Jantar dos Filósofos: versão base (deadlock) + 3 estratégias (hierarquia de recursos, limite de N-1 à mesa, garçom) |
| [`produtor-consumidor/`](produtor-consumidor) | Produtor/Consumidor com buffer limitado: versão com channel e versão com semáforos |
| [`relatorio/`](relatorio) | Relatório em PDF (`relatorio.pdf`), saídas dos experimentos (`dados/`) e o script que gera o PDF |

Requisito: Go >= 1.22. No Windows, `-race` exige um gcc de 64 bits no PATH (ex.: WinLibs ou MSYS2).

## Jantar dos Filósofos

```bash
cd filosofos
go run -race . -n 5 -r 1000                       # roda base, hierarquia, limite e garcom
go run -race . -n 7 -r 500 -estrategia garcom     # uma estratégia só
go run -race . -n 5 -r 1000 -pensar 0s            # carga de estresse (filósofos sempre famintos)
```

- `-n`: número de filósofos (>= 2). `-r`: número total de refeições da mesa.
- `-estrategia`: `base`, `hierarquia`, `limite`, `garcom` ou `todas` (padrão).
- `-pensar`, `-comer`, `-segurar`, `-timeout`: tempos (ex.: `2ms`). `-segurar` é a pausa entre pegar o 1º e o 2º garfo.
- Saída: refeições e espera média pelos garfos de cada filósofo. A versão `base` termina com "DEADLOCK" (esperado).

## Produtor/Consumidor

```bash
cd produtor-consumidor
go run -race . -p 4 -c 4 -k 1,10,100               # as duas versões, K = 1, 10 e 100
go run -race . -itens 500 -prod 3ms -cons 0s -timeout 2ms   # força timeouts do consumidor 0
```

- `-p`, `-c`: produtores e consumidores. `-k`: capacidades do buffer, separadas por vírgula.
- `-itens`: itens por produtor. `-prod`, `-cons`: tempo médio de trabalho por item.
- `-timeout`: timeout do consumidor 0 (`select` + `time.After`). `-versao`: `canal`, `semaforo` ou `ambas`.
- Cada linha termina com `OK` se total produzido == total consumido e nenhum item foi perdido ou repetido.

## Relatório

`relatorio/relatorio.pdf` (gerado por `python relatorio/gerar_relatorio.py`, requer `pip install reportlab`).
As saídas usadas nas tabelas estão em `relatorio/dados/`.
