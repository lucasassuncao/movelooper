# Sugestões de Melhoria — movelooper

## Correção / Corretude

| # | Arquivo | Linha | Problema | Sugestão |
|---|---------|-------|----------|----------|
| 1 | `history/history.go` | 88–91 | `prune()` roda antes de `save()`. Se o save falhar, o estado em memória já foi podado mas o disco não foi atualizado — desincroniza no próximo load. | Só chamar `prune()` após `save()` ter sucesso. |
| 2 | `history/history.go` | 182–194 | `RemoveBatch()` muta `h.Entries` antes de `save()`. Se o save falhar, estado em memória diverge do disco. | Construir a nova lista numa variável temporária, salvar primeiro, só então atribuir a `h.Entries`. Mesmo padrão para `RemoveCategoryFromBatch()`. |
| 3 | `fileops/fileops.go` | 205–226 | Cross-device move: se copy OK mas `os.Remove(src)` falha, o arquivo fica duplicado nos dois locais sem nenhum cleanup. | Ao falhar o remove, tentar `os.Remove(dst)` para desfazer a cópia antes de retornar o erro. |
| 4 | `config/builder.go` | 96–139 | Nenhuma validação impede source e destination iguais na mesma categoria, o que causaria loop infinito na execução. | Adicionar check `if source.Path == destination.Path { return error }` na validação de diretórios. |
| 5 | `scanner/walk.go` | 71–80 | `MaxDepth` negativo não é validado e pode causar comportamento inesperado. | Adicionar guard `if source.MaxDepth < 0 { return error }` no início de `WalkSource()`. |
| 6 | `cmd/undo.go` | 65–127 | `RemoveCategoryFromBatch()` não retorna quantas entradas foram removidas. Nome errado de categoria passa silenciosamente. | Retornar `int` (count) e emitir warn se for 0. |
| 7 | `config/imports.go` | 100–102 | Erro de import circular inclui só o arquivo atual, não o ciclo completo (`A → B → A`). | Acumular o caminho percorrido e incluir no erro: `"circular import: A → B → A"`. |
| 8 | `cmd/root.go` | 295–307 | `matchesCategory()` ignora erro de `file.Info()` e retorna `false` silenciosamente. Pode mascarar problemas de permissão. | Retornar `(bool, error)` para o caller logar adequadamente. |
| 9 | `cmd/undo.go` | 204–240 | Se um restore falha no meio do batch, o caller não sabe quais entradas foram restauradas parcialmente. | Retornar lista de entradas restauradas com sucesso para permitir partial-undo report. |

## Performance

| # | Arquivo | Linha | Problema | Sugestão |
|---|---------|-------|----------|----------|
| 10 | `cmd/root.go` | 193–199 | `file.Info()` pode ser chamado múltiplas vezes por arquivo (loop de extensões + filtros). | Cachear o resultado de `file.Info()` uma vez por arquivo antes do loop. |
| 11 | `scanner/walk.go` | 48–98 | `os.ReadDir()` é chamado antes de checar se o diretório está excluído. | Mover o check `isExcluded()` para antes do `os.ReadDir()`. |
| 12 | `updater/selfupdate.go` | 121–152 | `selectAsset()` itera candidatos duas vezes: uma para pontuar, outra para achar o melhor. | Combinar em um único loop determinando o melhor asset durante o scoring. |

## API & Usabilidade

| # | Arquivo | Linha | Problema | Sugestão |
|---|---------|-------|----------|----------|
| 13 | `fileops/fileops.go` | 46–54 | Callers devem construir `MoveRequest.SourceDir` manualmente, fácil de errar em modo recursivo. | Adicionar `NewMoveRequest()` que calcula `SourceDir` automaticamente a partir de um `FileEntry`. |
| 14 | `tokens/resolve.go` | 83–114 | `ResolveRename()` exige que `DestDir` e `SourcePath` estejam preenchidos no `TokenContext`, mas o tipo não distingue isso de `ResolveGroupBy()`. Fácil esquecer. | Criar `RenameContext` com os dois campos extras, deixando o compilador detectar o setup incompleto. |
| 15 | `cmd/categories.go` | 37–69 | `filterCategories()` não retorna quantas categorias foram silenciosamente ignoradas por `enabled: false`. | Retornar contagem de categorias puladas para melhor observabilidade. |
| 16 | `history/history.go` | 95–122 | Sem método público para consultar tamanho do histórico. Operadores não conseguem monitorar crescimento do arquivo. | Adicionar `GetEntryCount()` e `GetBatchCount()` exportados. |
| 17 | `fileops/conflict.go` | 40–207 | `conflictResolvers` é um map fechado — adicionar nova estratégia exige modificar o arquivo. | Expor função `RegisterConflictResolver(name string, r ConflictResolver)` para permitir extensão. |

## Manutenibilidade

| # | Arquivo | Linha | Problema | Sugestão |
|---|---------|-------|----------|----------|
| 18 | `config/builder.go` | 12–109 | Erros no `AppBuilder` são acumulados em `b.err` sem saída antecipada. Steps subsequentes rodam mesmo com erro anterior. | Documentar o comportamento, ou adicionar `HasError()` e retorno antecipado no primeiro erro. |
| 19 | `fileops/fileops.go` | 57–142 | `MoveFiles()` tem 85 linhas fazendo checagem de extensão, resolução de destino, conflito, dispatch, histórico e log. | Extrair em: `resolveDestination()`, `executeAction()`, `recordHistory()`. |
| 20 | `tokens/resolve.go` | 100–114 | A ordem de `preProcess*` em `ResolveRename()` importa mas não é documentada. | Adicionar comentário ou extrair `preProcessAll()` com a sequência correta explícita. |
| 21 | `cmd/watch.go` | 113–117 | Construção do `watchConfig` está inline em `runWatch()` sem validação. | Extrair `NewWatchConfig()` com validação explícita. |
| 22 | `filters/filters.go` | 159–197 | `MatchesFilter()` é recursivo sem limite de profundidade. Configuração muito aninhada pode estourar stack. | Adicionar `maxDepth` e retornar erro se ultrapassado, ou reescrever iterativamente. |
| 23 | `cmd/init.go` | 83–153 | `runInit()` repete `os.WriteFile()` nos três branches (scan, interactive, template). | Extrair `writeConfigFile(path string, data []byte) error`. |

## Testabilidade

| # | Arquivo | Linha | Problema | Sugestão |
|---|---------|-------|----------|----------|
| 24 | `cmd/watch.go` | 107 | `fsnotify.NewWatcher()` é chamado diretamente, impossibilitando testes sem FS real. | Aceitar `WatcherFactory` como parâmetro ou variável de pacote substituível em testes. |
| 25 | `hooks/hooks.go` | 19–44 | `exec.CommandContext()` direto — testes precisam de shell real. | Injetar `type Executor interface { Run(ctx, name, args) error }` para permitir mock. |
| 26 | `tokens/system.go` | 17–31 | Hostname/username/OS são inicializados uma vez e cacheados. Testes não conseguem sobrescrever. | Adicionar `InitSystemContext(hostname, username, os string)` para uso em testes. |
| 27 | `scanner/walk.go` | 22–98 | `WalkSource()` recebe `models.CategorySource` concreto, exigindo config completa em testes. | Aceitar interface com métodos `Path()`, `IsRecursive()`, etc. |

## Mensagens de Erro / Contexto Perdido

| # | Arquivo | Linha | Problema | Sugestão |
|---|---------|-------|----------|----------|
| 28 | `cmd/root.go` | 328–352 | Erros de config são wrappados com `%s` em vez de `%w`, quebrando a cadeia de erros. | Trocar por `%w` para preservar `errors.Is()` e `errors.As()`. |
| 29 | `fileops/fileops.go` | 112–120 | Log de falha de action inclui apenas `action`, não `destPath` nem a conflict strategy usada. | Adicionar `destPath` e `strategy` nos args do log. |
| 30 | `fileops/fileops.go` | 293–307 | `getUniqueDestinationPath()` retorna erro genérico sem indicar qual arquivo esgotou as 1000 tentativas. | Incluir `destDir` e `fileName` no erro para facilitar diagnóstico. |

---

## Prioridade

| Prioridade | Itens | Motivo |
|------------|-------|--------|
| Alta | #1, #2, #3, #4 | Perda de dados ou estado corrompido |
| Média | #6, #7, #8, #10, #11, #18, #19, #28 | Impacto direto em debug e comportamento |
| Baixa | demais | Testabilidade, extensibilidade e clareza de API |
