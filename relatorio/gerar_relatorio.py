# Gera relatorio.pdf (pip install reportlab). Uso: python relatorio/gerar_relatorio.py
# Os números das tabelas vêm das saídas em relatorio/dados/ (execuções reais dos programas).
import os

from reportlab.lib import colors
from reportlab.lib.enums import TA_JUSTIFY
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import cm
from reportlab.platypus import Paragraph, SimpleDocTemplate, Table, TableStyle

AQUI = os.path.dirname(os.path.abspath(__file__))
ss = getSampleStyleSheet()
B = ParagraphStyle("B", parent=ss["BodyText"], fontName="Helvetica", fontSize=9, leading=11.6, alignment=TA_JUSTIFY, spaceAfter=3.5)
H1 = ParagraphStyle("H1", parent=ss["Heading1"], fontName="Helvetica-Bold", fontSize=12.5, spaceBefore=7, spaceAfter=3)
H2 = ParagraphStyle("H2", parent=ss["Heading2"], fontName="Helvetica-Bold", fontSize=10, spaceBefore=4, spaceAfter=2)
T = ParagraphStyle("T", parent=B, fontSize=7.8, leading=9.4, alignment=0, spaceAfter=0)
TT = ParagraphStyle("TT", parent=T, fontName="Helvetica-Bold")
CAP = ParagraphStyle("CAP", parent=B, fontSize=8, leading=9.6, alignment=0, textColor=colors.HexColor("#444444"))
C = lambda t: f"<font face='Courier'>{t}</font>"


def P(t, st=B):
    return Paragraph(t, st)


def tabela(linhas, larguras, extra=None):
    d = [[P(str(c), TT if i == 0 else T) for c in l] for i, l in enumerate(linhas)]
    t = Table(d, colWidths=larguras, repeatRows=1)
    t.setStyle(TableStyle([("GRID", (0, 0), (-1, -1), 0.4, colors.HexColor("#999999")),
                           ("BACKGROUND", (0, 0), (-1, 0), colors.HexColor("#e8e8e8")),
                           ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
                           ("TOPPADDING", (0, 0), (-1, -1), 1.5), ("BOTTOMPADDING", (0, 0), (-1, -1), 1.5)] + (extra or [])))
    return t


s = []
s.append(P("T1: Problemas Clássicos de Concorrência", ParagraphStyle("t", parent=H1, fontSize=15)))
s.append(P("FPPD - Fundamentos de Processamento Paralelo e Distribuído (98713-04) &nbsp;|&nbsp; <b>Eduardo Hoffmann e Lucas Mocelin</b><br/>"
           "Repositório: https://github.com/theeduardohoffmann/go-concurrency-benchmarks<br/>"
           f"Ambiente: Go 1.27, Windows 11, 12 CPUs lógicas. Os dois programas foram executados com {C('go run -race')} sem nenhum aviso de data race. "
           f"Os dados abaixo estão em {C('relatorio/dados/')} (execuções reais, sem {C('-race')} para não distorcer o tempo). O {C('time.Sleep')} do Windows tem granularidade de "
           "cerca de 1 ms, então os tempos valem mais para comparar configurações entre si do que como valores absolutos."))

# ======================================================== PROBLEMA 1
s.append(P("1. Problema 1: Jantar dos Filósofos", H1))
s.append(P("1.1 Solução e decisões de projeto", H2))
s.append(P(f"Cada garfo é um {C('chan struct{}')} de capacidade 1: se há um token no channel, o garfo está livre. Pegar o garfo é receber do channel e soltar é enviar de volta. "
           f"Cada filósofo é uma goroutine que repete: pensar, pegar os garfos, comer, soltar. O programa recebe {C('-n')} (número de filósofos) e {C('-r')} (iterações). "
           f"Interpretamos R como o <b>número total de refeições da mesa</b> (um contador atômico decrementado a cada refeição): assim cada filósofo come o quanto conseguir e a distribuição "
           "das refeições mostra se a estratégia é justa. Se cada filósofo fizesse R refeições, todos terminariam com o mesmo número por construção e não haveria o que comparar."))
s.append(P(f"<b>Versão base:</b> pega a esquerda, espera um pouco ({C('-segurar')}, 2 ms, aplicada a todas as versões) e pega a direita. A pausa faz o deadlock acontecer de forma reproduzível. "
           "<b>Estratégias que eliminam o deadlock:</b> "
           "(1) <b>hierarquia de recursos</b>: os garfos são numerados e cada filósofo pega primeiro o de menor número (só o filósofo N-1 inverte a ordem); "
           "(2) <b>limite de N-1 à mesa</b>: um channel de capacidade N-1 funciona como semáforo (a 'sala'): o filósofo entra na sala antes de pegar garfos e sai depois de comer; "
           "(3) <b>garçom</b>: uma goroutine recebe os pedidos por um channel, em ordem de chegada, retira os dois garfos do filósofo e só então responde; o filósofo come e devolve os garfos."))
s.append(P("1.2 Encerramento ordenado", H2))
s.append(P(f"O filósofo termina quando as R refeições acabam; a main espera todos com {C('sync.WaitGroup')}. No garçom, depois disso a main fecha o channel de pedidos, o {C('for range')} do garçom acaba e a main "
           f"espera um channel {C('garcomFim')}. A espera por um garfo é um {C('select')} com {C('time.After')} (1 s): se um filósofo espera tanto assim, o programa conclui que há deadlock, imprime quem "
           "segura/espera qual garfo e a goroutine retorna. Assim, até a versão que trava termina sem goroutines bloqueadas."))

s.append(P("1.3 Análise de deadlock e starvation", H2))
s.append(P("Condições de Coffman: (a) exclusão mútua, (b) posse e espera, (c) não preempção e (d) espera circular. O deadlock exige as quatro; cada estratégia quebra uma."))
linhas = [["Versão", "Deadlock?", "Justificativa (Coffman)", "Starvation?"],
          ["base", f"<b>Sim</b>: 3 de 3 execuções. Saída: filósofo i esperando o garfo i+1, para todo i (0 espera 1, 1 espera 2, ..., 4 espera 0).",
           "As quatro condições valem: garfo é exclusivo, cada um segura o da esquerda e pede o da direita, ninguém é forçado a soltar e as esperas formam um ciclo.",
           "Sim, como consequência do deadlock: ninguém mais come."],
          ["hierarquia", "Não (0 deadlocks em 6 execuções: 3 normais e 3 de estresse).",
           "Quebra a <b>espera circular</b> (d): como todos pegam em ordem crescente, um ciclo de espera exigiria uma ordem decrescente em algum ponto, o que não existe.",
           "Não há starvation total, mas é <b>injusta</b>: os filósofos 0 e 4 comem bem menos (ver 1.4)."],
          ["limite (N-1)", "Não (0 em 6 execuções).",
           "Com no máximo N-1 filósofos disputando N garfos, sempre sobra um garfo para algum deles; ele come e libera. Quebra a espera circular (d), pois o ciclo completo precisaria de N filósofos.",
           "Não: o channel-semáforo atende em ordem de chegada; refeições quase idênticas."],
          ["garçom", "Não (0 em 6 execuções).",
           "Quebra a <b>posse e espera</b> (b): só o garçom retira garfos, e o filósofo só recebe quando já tem os dois. Quem segura um garfo está comendo e vai soltá-lo.",
           "Não: fila FIFO; o pedido mais antigo é sempre o próximo, então ninguém é ultrapassado."]]
s.append(tabela(linhas, [1.7 * cm, 4.2 * cm, 7.6 * cm, 3.9 * cm]))
s.append(P("A ausência de starvation nas versões corretas depende de dois fatos: a refeição dura um tempo finito e as filas de espera dos channels do Go são servidas em ordem de chegada (comportamento da implementação do runtime).", CAP))

s.append(P("1.4 Dados de execução e fairness", H2))
dados = [["Estratégia", "Refeições por filósofo (F0 F1 F2 F3 F4)", "Espera média pelos garfos em ms (F0 ... F4)", "Tempo total"],
         [P("<b>Carga normal</b> (pensa até 2 ms), N=5, R=1000, execução 1", T), "", "", ""],
         ["base", "0 0 0 0 0 (deadlock)", "-", "1,0 s (timeout)"],
         ["hierarquia", "163 207 218 248 164", "4,67  3,13  2,76  2,07  4,69", "1,26 s"],
         ["limite (N-1)", "199 199 201 201 200", "4,69  4,61  4,64  4,58  4,67", "1,54 s"],
         ["garçom", "198 201 200 204 197", "1,68  1,70  1,70  1,66  1,79", "0,91 s"],
         [P("<b>Carga de estresse</b> (não pensa: sempre faminto), N=5, R=1000, execução 1", T), "", "", ""],
         ["base", "0 0 0 0 0 (deadlock)", "-", "1,0 s (timeout)"],
         ["hierarquia", "135 193 218 318 136", "7,68  4,94  4,23  2,49  7,61", "1,24 s"],
         ["limite (N-1)", "200 200 200 200 200", "6,20  6,13  6,25  6,15  6,25", "1,55 s"],
         ["garçom", "200 200 200 200 200", "6,81  6,68  6,77  6,77  6,76", "1,69 s"]]
s.append(tabela(dados, [2.6 * cm, 5.3 * cm, 6.2 * cm, 3.3 * cm],
                [("SPAN", (0, 1), (-1, 1)), ("SPAN", (0, 6), (-1, 6)),
                 ("BACKGROUND", (0, 1), (-1, 1), colors.HexColor("#f3f3f3")), ("BACKGROUND", (0, 6), (-1, 6), colors.HexColor("#f3f3f3"))]))
s.append(P("Cada configuração foi executada 3 vezes e o padrão se repetiu (hierarquia: de 157 a 163 refeições do filósofo 0 e de 248 a 257 do filósofo 3 na carga normal; de 131 a 137 contra 316 a 321 sob estresse). "
           "A tabela mostra a execução 1. Nas execuções 1 e 2 da base na carga normal ninguém chegou a comer e na 3 foram só 2 refeições.", CAP))
s.append(P("<b>Hierarquia</b> evita o deadlock mas é a estratégia menos justa. Sob estresse, os filósofos 0 e 4 comem cerca de 135 vezes, contra cerca de 318 do filósofo 3, e esperam 3 vezes mais. "
           "Nossa hipótese (não testada separadamente): os filósofos 0 e 4 pegam ambos primeiro o garfo 0 e o seguram enquanto esperam o segundo garfo, atrapalhando-se mutuamente, enquanto o filósofo 3 "
           "tem menos concorrência. <b>Limite (N-1)</b> é justo (200 refeições para todos sob estresse) porque a fila da sala é FIFO. <b>Garçom</b> também é justo. "
           "Sobre desempenho, o garçom foi o mais rápido na carga normal (~0,9-1,0 s contra ~1,4-1,5 s do limite e ~1,2 s da hierarquia), mas sob estresse ficou o mais lento (~1,6-1,7 s): "
           "como atende um pedido de cada vez em ordem estrita, o garçom pode ficar esperando um garfo ocupado enquanto outro pedido atrás dele poderia ser atendido; quando os filósofos "
           "pensam, essa fila quase não se forma. Concluímos que hierarquia é a mais simples, mas a menos justa; limite e garçom são justas, com custos de desempenho diferentes."))

# ======================================================== PROBLEMA 2
s.append(P("2. Problema 2: Produtor/Consumidor com buffer limitado", H1))
s.append(P("2.1 Solução e decisões de projeto", H2))
s.append(P(f"As duas versões implementam a mesma interface ({C('put, get, getTimeout, tamanho')}), então os produtores e consumidores são o mesmo código. "
           f"<b>Versão com channel:</b> o buffer é {C('make(chan int, K)')}; o próprio Go bloqueia o produtor com o buffer cheio e o consumidor com o buffer vazio. "
           f"<b>Versão com semáforos:</b> o buffer é um slice circular de K posições com dois semáforos de contagem, {C('notFull')} (começa em K) e {C('notEmpty')} (começa em 0), e dois mutexes. "
           f"O semáforo é implementado com um channel de tokens ({C('P')} = receber, {C('V')} = enviar), pois a biblioteca padrão não tem semáforo. O algoritmo é: "
           f"{C('put: P(notFull); Lock(mEntra); itens[entra]=x; entra++; Unlock(mEntra); V(notEmpty)')} e "
           f"{C('get: P(notEmpty); Lock(mSai); x=itens[sai]; sai++; Unlock(mSai); V(notFull)')}. "
           f"Com vários produtores e consumidores os semáforos sozinhos não bastam: sem {C('mEntra')} dois produtores poderiam receber a mesma posição de escrita e um sobrescreveria o outro; "
           f"sem {C('mSai')} dois consumidores poderiam retirar o mesmo item. Usamos dois mutexes diferentes para que um produtor e um consumidor possam trabalhar ao mesmo tempo. "
           f"O semáforo é obtido antes do mutex para ninguém bloquear esperando vaga/item enquanto segura o mutex. Não há data race entre a escrita e a leitura da mesma posição porque o {C('V(notEmpty)')} só ocorre "
           f"depois da escrita (e o {C('V(notFull)')} depois da leitura), o que cria a relação happens-before do modelo de memória de Go (confirmado com {C('-race')})."))
s.append(P(f"<b>Consumo com timeout:</b> o consumidor 0 usa {C('select { case x := <-...: ...; case <-time.After(d): ... }')} nas duas versões. Se nenhum item chega em d, ele executa o comportamento alternativo "
           "(conta o timeout e imprime uma mensagem nas 3 primeiras vezes) e volta a tentar. "
           "<b>Verificação:</b> cada item tem um identificador único (produtor*itens + sequência); ao final o programa confere se o total consumido é igual ao produzido e se nenhum identificador foi consumido duas vezes ou perdido. "
           "A saída mostra OK ou ERRO em cada linha; em todas as execuções deste relatório foi OK."))
s.append(P("2.2 Encerramento ordenado", H2))
s.append(P(f"Depois que todos os produtores terminam ({C('WaitGroup')}), a main coloca no buffer uma <b>pílula de fim</b> (valor especial -1) para cada consumidor. Como o buffer é FIFO, as pílulas ficam atrás de todos os itens reais: "
           "um consumidor só recebe a pílula depois que os itens anteriores já foram retirados, ou seja, o buffer é drenado e nenhum item se perde. Cada consumidor encerra ao receber uma pílula, e a main espera os consumidores. "
           "É o mesmo mecanismo nas duas versões. Não há deadlock porque ninguém fica esperando um item que nunca virá: há exatamente uma pílula por consumidor. "
           "O consumidor 0 (com timeout) também sai ao receber a pílula."))
s.append(P("2.3 Deadlock e starvation", H2))
s.append(P("Não ocorre deadlock em nenhuma das versões: todas as execuções terminaram, também com K=1 (o caso de mais bloqueio) e com o encerramento por pílulas. "
           "Os produtores só esperam vaga, que os consumidores liberam, e os consumidores só esperam itens, que os produtores entregam; não há dependência circular entre dois recursos e nenhum "
           "mutex é mantido durante uma espera. Starvation: as filas de espera dos channels são servidas em ordem de chegada, então ninguém é ignorado indefinidamente; no cenário equilibrado cada consumidor processou entre 1975 e 2024 dos ~2000 itens. "
           "A exceção é o consumidor 0 com timeout: a cada timeout ele volta ao fim da fila de espera e, no cenário 'consumidor rápido', recebeu só 128 a 156 itens contra ~615 dos outros. Isso é efeito do timeout, não do buffer."))

s.append(P("2.4 Efeito da capacidade K", H2))
d2 = [["Cenário (P=C=4)", "Versão", "K", "Itens/s", "Ocupação média", "Espera média no put", "Itens por consumidor (C0 C1 C2 C3)"]]
rows = [
    ("Equilibrado (prod=cons=0,5 ms, 2000 itens/prod.)", [
        ("channel", 1, "4443", "0,52", "59 µs", "2017 1994 2006 1983"), ("channel", 10, "4146", "4,61", "9 µs", "1975 2015 2008 2002"),
        ("channel", 100, "4301", "24,85", "0", "1988 2024 1979 2009"), ("semáforos", 1, "4127", "0,50", "62 µs", "2020 1978 2010 1992"),
        ("semáforos", 10, "4413", "4,37", "6 µs", "2002 2009 2006 1983"), ("semáforos", 100, "4336", "36,28", "0", "2010 2000 2005 1985")]),
    ("Sem trabalho (só sincronização, 50000 itens/prod.)", [
        ("channel", 1, "2.289.673", "0,59", "2 µs", "30599 58182 55907 55312"), ("channel", 10, "3.681.397", "5,48", "1 µs", "23851 57451 59821 58877"),
        ("channel", 100, "4.912.315", "46,34", "1 µs", "20113 57313 60421 62153"), ("semáforos", 1, "1.401.108", "0,00", "3 µs", "48867 50430 50456 50247"),
        ("semáforos", 10, "1.844.391", "2,73", "2 µs", "40613 54750 53000 51637"), ("semáforos", 100, "2.556.515", "55,61", "2 µs", "43396 53321 50780 52503")]),
    ("Produtor rápido (prod=0, cons=1 ms)", [
        ("channel", 1, "2981", "1,00", "1,34 ms", "2022 2003 1988 1987"), ("channel", 10, "2779", "9,99", "1,44 ms", "1997 1975 2046 1982"),
        ("channel", 100, "2837", "99,46", "1,37 ms", "2013 1974 1999 2014"), ("semáforos", 1, "2905", "0,99", "1,38 ms", "1974 2019 1975 2032"),
        ("semáforos", 10, "2859", "9,98", "1,40 ms", "2002 2008 2013 1977"), ("semáforos", 100, "2830", "99,36", "1,38 ms", "1984 2024 1983 2009")]),
    ("Consumidor rápido (prod=3 ms, cons=0, timeout 2 ms, 500 itens/prod.)", [
        ("channel", 1, "1170", "0,00", "0", "155 615 614 616 (623 timeouts)"), ("channel", 10, "1154", "0,00", "0", "128 624 624 624 (632)"),
        ("channel", 100, "1144", "0,00", "0", "145 619 617 619 (624)"), ("semáforos", 1, "1176", "0,00", "0", "146 618 618 618 (621)"),
        ("semáforos", 10, "1120", "0,00", "0", "148 617 617 618 (643)"), ("semáforos", 100, "1130", "0,00", "0", "156 615 614 615 (621)")]),
]
extra = []
for nome, rs in rows:
    ini = len(d2)
    for i, r in enumerate(rs):
        d2.append([nome if i == 0 else "", *r])
    extra += [("SPAN", (0, ini), (0, len(d2) - 1)), ("LINEABOVE", (0, ini), (-1, ini), 1.1, colors.black)]
s.append(tabela(d2, [3.7 * cm, 1.7 * cm, 1.0 * cm, 1.9 * cm, 2.1 * cm, 2.3 * cm, 4.7 * cm], extra))
s.append(P("Uma execução por configuração. Ocupação = média de amostras do tamanho do buffer a cada 200 µs. Itens/s = itens consumidos / tempo total. Espera no put = tempo médio que o produtor ficou bloqueado em cada put.", CAP))
s.append(P("<b>Efeito de K e desacoplamento.</b> K é a folga que deixa produtores e consumidores terem ritmos diferentes sem bloquear um ao outro. "
           "No cenário <b>equilibrado</b>, a vazão fica em ~4100-4400 itens/s para qualquer K (a diferença entre K=1 e K=100 é ruído: a maior vazão foi justamente a de K=1 no channel), "
           "porque o limite é o tempo de trabalho simulado. O efeito de K aparece no bloqueio: com K=1 o produtor espera ~60 µs por item (quase um encontro entre produtor e consumidor), "
           "com K=10 cai para 6-9 µs e com K=100 para ~0, enquanto a ocupação média sobe de 0,5 para ~4,5 e para 25-36: o buffer absorve a variação aleatória dos tempos. "
           "No <b>produtor rápido</b> o buffer fica sempre cheio (ocupação ≈ K) e o produtor bloqueia ~1,4 ms por item em qualquer K: quem manda é o consumidor (~2800-3000 itens/s), e um K maior só "
           "guarda mais itens esperando. No <b>consumidor rápido</b> acontece o oposto: o buffer fica vazio (ocupação 0), os consumidores é que esperam e K é irrelevante; nesse cenário o consumidor 0 estoura o timeout ~620-640 vezes e executa a tarefa alternativa. "
           "No cenário <b>sem trabalho</b>, em que só se mede o custo da sincronização, K tem grande efeito: no channel a vazão sobe de 2,3 para 4,9 milhões de itens/s (2,1x) de K=1 para K=100, "
           "e nos semáforos de 1,4 para 2,6 milhões (1,8x), porque com mais folga há menos bloqueios e trocas de contexto entre goroutines. Ou seja, o desacoplamento só vira ganho de vazão quando a sincronização é o gargalo."))
s.append(P("<b>Channel x semáforos.</b> Com trabalho simulado as duas versões são indistinguíveis dentro do ruído. Sem trabalho, o channel foi de 1,6x a 2,0x mais rápido: cada item custa uma operação de channel, que o runtime do Go "
           "já implementa com um buffer circular e um lock interno, contra duas operações de semáforo (P e V, cada uma um channel) mais um mutex na versão manual. A versão com semáforos tem o mesmo comportamento de bloqueio, "
           "mas com mais trabalho de sincronização por item. Observação: nesse cenário o consumidor 0 consumiu bem menos itens (20-31 mil contra ~57 mil no channel), o que atribuímos (hipótese) ao custo de criar um timer a cada "
           "chamada de <font face='Courier'>time.After</font>, e não a injustiça do buffer. Como cada configuração foi medida uma vez, diferenças pequenas entre linhas não devem ser interpretadas."))

# ======================================================== IA
s.append(P("3. Uso de ferramentas de IA", H1))
s.append(P("Foi utilizado o <b>Claude Code (modelo Claude Sonnet 5.5, da Anthropic)</b>. Etapas: <b>projeto da solução</b> (escolha das estratégias, do protocolo de encerramento e da interpretação de R); "
           "<b>geração de código</b> (os dois programas em Go e os scripts de coleta); <b>depuração e validação</b> (instalação do Go e do gcc de 64 bits, execução com <font face='Courier'>-race</font>, e uma simplificação do código a pedido dos alunos "
           "para facilitar a compreensão e a apresentação); <b>análise dos dados</b> (leitura das saídas e rascunho das interpretações); <b>redação</b> do relatório e do README. "
           "Os experimentos foram realmente executados e todos os números deste relatório vêm das saídas guardadas em <font face='Courier'>relatorio/dados/</font>. "
           "As explicações marcadas como hipótese não foram verificadas por experimento próprio. Cada integrante deve ser capaz de explicar qualquer parte do código e das análises."))
s.append(P("4. Como compilar e executar", H1))
s.append(P(f"Go >= 1.22 (para {C('-race')} no Windows é preciso um gcc de 64 bits no PATH). "
           f"{C('cd filosofos; go run -race . -n 5 -r 1000 -estrategia todas')} &nbsp;|&nbsp; "
           f"{C('cd produtor-consumidor; go run -race . -p 4 -c 4 -k 1,10,100 -versao ambas')} &nbsp;|&nbsp; {C('-h')} lista todas as opções. Mais detalhes no README."))

doc = SimpleDocTemplate(os.path.join(AQUI, "relatorio.pdf"), pagesize=A4, leftMargin=1.7 * cm, rightMargin=1.7 * cm,
                        topMargin=1.5 * cm, bottomMargin=1.4 * cm, title="T1 - Problemas Clássicos de Concorrência",
                        author="Eduardo Hoffmann e Lucas Mocelin")
doc.build(s)
print("ok")
