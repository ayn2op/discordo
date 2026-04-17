# 🔴 BUGS CRÍTICOS - Action Required

## 1. Vazamento de Arquivo - Logger
- **Arquivo:** `internal/logger/logger.go:26`
- **Problema:** `file.Close()` nunca chamado
- **Solução:** Adicionar `defer file.Close()` após `os.OpenFile()`
- **Risco:** "too many open files" crash

## 2. Type Assertion Insegura - Cache  
- **Arquivo:** `internal/cache/cache.go:24`
- **Problema:** `i.(uint)` sem verificação, panic se nil
- **Solução:** Retornar `(uint, bool)` em vez de `uint`
- **Risco:** Crash em tempo de execução

## 3. Race Condition - typingTimer
- **Arquivo:** `internal/ui/chat/message_input.go:138-145`
- **Problema:** Acesso a `typingTimer` sem lock em Update()
- **Solução:** Usar `typingTimerMu` para todas as operações
- **Risco:** Crash aleatório durante digitação

---

# 🟠 BUGS ALTOS - Próxima Sprint

| # | Arquivo | Problema | Severidade |
|---|---------|----------|-----------|
| 4 | chat/model.go:187 | Goroutine leak em closeState() | Vazamento |
| 5 | messages_list.go:1305 | Race em waitForChunkEvent() | Deadlock |
| 6 | config/config.go:132 | Return prematuro | Dead code |
| 7 | message_input.go:145 | Goroutine leak typing | Memory leak |
| 8 | messages_list.go:1190 | Erro ignorado openAttachment | UX ruim |

---

# 🟡 BUGS MÉDIOS - Backlog

- 2x Race conditions em Login/QR
- 3x Erros ignorados (setClipboard, openState, getToken)
- 1x Deadlock potencial em listen()

---

# 📋 Checklist de Correção

- [ ] Bug #1: Logger file close
- [ ] Bug #2: Cache type assertion
- [ ] Bug #3: typingTimer synchronization
- [ ] Bug #4: clearTypers on closeState
- [ ] Bug #5: waitForChunkEvent mutex
- [ ] Bug #6: config.go return statement
- [ ] Add `-race` to test pipeline
- [ ] Implement error propagation to UI
- [ ] Add goroutine tracking
- [ ] Documentation review

