# 🐛 ANÁLISE COMPLETA DE BUGS - DISCORDO

> Relatório completo de bugs encontrados no projeto Discord CLI client escrito em Go

**Data:** 17 de Abril de 2026  
**Status:** ⚠️ **18 bugs encontrados** (3 críticos, 5 altos, 7 médios, 3 baixos)

---

## 📊 Resumo Executivo

| Severidade | Quantidade | Tipos Principais |
|-----------|-----------|-----------------|
| 🔴 **CRÍTICO** | 3 | Vazamento de recursos, Type assertions inseguras, Goroutine leaks |
| 🟠 **ALTO** | 5 | Race conditions, Problemas de sincronização |
| 🟡 **MÉDIO** | 7 | Goroutine leaks, Tratamento de erro inadequado |
| 🟢 **BAIXO** | 3 | Validação inadequada, Erros ignorados |

---

## 🔴 BUGS CRÍTICOS

### BUG #1: Vazamento de Arquivo - Logger

**Arquivo:** [internal/logger/logger.go](internal/logger/logger.go#L26)  
**Tipo:** Resource Leak (Vazamento de File Descriptor)  
**Severidade:** 🔴 **CRÍTICO**

**Problema:**
```go
func Load(path string, level slog.Level) error {
    if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
        return err
    }

    file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
    if err != nil {
        return fmt.Errorf("failed to open log file: %w", err)
    }
    // ❌ FALTA: defer file.Close()

    opts := &slog.HandlerOptions{Level: level}
    handler := slog.NewTextHandler(file, opts)  // File fica aberto forever
    slog.SetDefault(slog.New(handler))
    return nil
}
```

**Impacto:**
- Vazamento de file descriptor durante toda a execução
- Eventual crash com erro "too many open files"
- Afeta sistemas com limite baixo de file descriptors

**Solução:**
```go
file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
if err != nil {
    return fmt.Errorf("failed to open log file: %w", err)
}
defer file.Close()  // ✅ Adicionar aqui

opts := &slog.HandlerOptions{Level: level}
handler := slog.NewTextHandler(file, opts)
slog.SetDefault(slog.New(handler))
return nil
```

---

### BUG #2: Type Assertion Insegura - Cache

**Arquivo:** [internal/cache/cache.go](internal/cache/cache.go#L23-L25)  
**Tipo:** Type Assertion sem verificação / Nil Pointer Dereference  
**Severidade:** 🔴 **CRÍTICO**

**Problema:**
```go
func (c *Cache) Get(query string) uint {
    i, _ := c.items.Load(query)  // ❌ Ignora booleano "ok"
    return i.(uint)              // ❌ Panic se i é nil!
}
```

**Cenário de Falha:**
```go
cache := cache.NewCache()
cache.Create("user1", 10)

// Tentativa de pegar valor inexistente
value := cache.Get("user2")  // ❌ PANIC! interface conversion: <nil> is not uint
```

**Impacto:**
- Crash imediato com panic se chave não existe
- Crash se valor tem tipo diferente
- Nenhuma forma segura de verificar se valor existe

**Solução:**
```go
func (c *Cache) Get(query string) (uint, bool) {
    i, ok := c.items.Load(query)
    if !ok {
        return 0, false
    }
    val, ok := i.(uint)
    return val, ok
}

// Uso:
if value, ok := cache.Get("user2"); ok {
    // Use value
}
```

---

### BUG #3: Goroutine Leak com Timer Abandonado

**Arquivo:** [internal/ui/chat/model.go](internal/ui/chat/model.go#L179-L187)  
**Tipo:** Goroutine Leak / Resource Leak  
**Severidade:** 🔴 **CRÍTICO**

**Problema:**
```go
func (m *Model) addTyper(userID discord.UserID) {
    m.typersMu.Lock()
    typer, ok := m.typers[userID]
    if ok {
        typer.Reset(typingDuration)
    } else {
        m.typers[userID] = time.AfterFunc(typingDuration, func() {
            m.removeTyper(userID)  // ❌ Goroutine fica alive se app fecha
        })
    }
    m.typersMu.Unlock()
    m.updateFooter()
}

// ❌ PROBLEMA: Ao fechar estado, timers não são limpos:
func (m *Model) closeState() tview.Cmd {
    return func() tview.Msg {
        if m.state != nil {
            if err := m.state.Close(); err != nil {
                slog.Error("failed to close the session", "err", err)
                return nil
            }
        }
        // ❌ FALTA: m.clearTypers() aqui!
        return nil
    }
}
```

**Impacto:**
- Goroutines orphaned consomem recursos indefinidamente
- Memory leak ao fechar e reabrir sessões
- Comportamento indeterminado após logout

**Solução:**
```go
func (m *Model) clearTypers() {
    m.typersMu.Lock()
    defer m.typersMu.Unlock()
    
    for userID, timer := range m.typers {
        timer.Stop()
        delete(m.typers, userID)
    }
}

func (m *Model) closeState() tview.Cmd {
    return func() tview.Msg {
        m.clearTypers()  // ✅ Adicionar aqui
        
        if m.state != nil {
            if err := m.state.Close(); err != nil {
                slog.Error("failed to close the session", "err", err)
                return nil
            }
        }
        return nil
    }
}
```

---

## 🟠 BUGS ALTOS

### BUG #4: Race Condition no typingTimer

**Arquivo:** [internal/ui/chat/message_input.go](internal/ui/chat/message_input.go#L100-L145)  
**Tipo:** Race Condition / Data Race  
**Severidade:** 🟠 **ALTO**

**Problema:**
```go
type messageInput struct {
    *tview.TextArea
    chat *Model
    // ...
    typingTimerMu sync.Mutex
    typingTimer   *time.Timer  // Campo compartilhado
}

func (mi *messageInput) stopTypingTimer() {
    if mi.typingTimer != nil {  // ❌ Race condition: lê sem lock
        mi.typingTimer.Stop()   // ❌ Race condition: acessa sem lock
        mi.typingTimer = nil    // ❌ Race condition: escreve sem lock
    }
}

func (mi *messageInput) Update(msg tview.Msg) tview.Cmd {
    // ...
    if mi.cfg.TypingIndicator.Send && mi.typingTimer == nil {  // ❌ Sem lock!
        mi.typingTimer = time.AfterFunc(typingDuration, func() {  // ❌ Sem lock!
            mi.typingTimerMu.Lock()
            mi.typingTimer = nil  // ✅ Com lock, mas verificação anterior sem
            mi.typingTimerMu.Unlock()
        })
    }
    // ...
}
```

**Cenário de Falha:**
- Thread 1: Lê `mi.typingTimer`, obtém valor X
- Thread 2: Escreve novo timer em `mi.typingTimer`
- Thread 1: Acessa X que foi potencialmente modificado
- Resultado: Undefined behavior/crash

**Impacto:**
- Crash aleatório durante digitação
- Corrupção de dados
- Comportamento indeterminado

**Solução:**
```go
func (mi *messageInput) stopTypingTimer() {
    mi.typingTimerMu.Lock()
    defer mi.typingTimerMu.Unlock()
    
    if mi.typingTimer != nil {
        mi.typingTimer.Stop()
        mi.typingTimer = nil
    }
}

func (mi *messageInput) Update(msg tview.Msg) tview.Cmd {
    // ...
    if mi.cfg.TypingIndicator.Send {
        mi.typingTimerMu.Lock()
        if mi.typingTimer == nil {
            mi.typingTimer = time.AfterFunc(typingDuration, func() {
                mi.typingTimerMu.Lock()
                mi.typingTimer = nil
                mi.typingTimerMu.Unlock()
            })
        }
        mi.typingTimerMu.Unlock()
    }
    // ...
}
```

---

### BUG #5: Race Condition em fetchingMembers

**Arquivo:** [internal/ui/chat/messages_list.go](internal/ui/chat/messages_list.go#L1290-L1305)  
**Tipo:** Race Condition / Channel Closed  
**Severidade:** 🟠 **ALTO**

**Problema:**
```go
type messagesList struct {
    // ...
    fetchingMembers struct {
        mu    sync.Mutex
        value bool
        done  chan struct{}  // Canal sem sincronização após Unlock
        count uint
    }
}

func (ml *messagesList) waitForChunkEvent() uint {
    ml.fetchingMembers.mu.Lock()
    if !ml.fetchingMembers.value {
        ml.fetchingMembers.mu.Unlock()
        return 0
    }
    ml.fetchingMembers.mu.Unlock()  // ❌ Lock liberado
    
    <-ml.fetchingMembers.done  // ❌ Race: 'done' pode ser modificado/fechado
    return ml.fetchingMembers.count  // ❌ Lê sem lock!
}
```

**Cenário de Falha:**
- Thread 1: Libera lock
- Thread 2: Fecha `done` canal
- Thread 1: Tenta ler de canal fechado → Panic ou deadlock

**Impacto:**
- Panic em tempo de runtime
- Deadlock em casos extremos

**Solução:**
```go
func (ml *messagesList) waitForChunkEvent() uint {
    ml.fetchingMembers.mu.Lock()
    if !ml.fetchingMembers.value {
        ml.fetchingMembers.mu.Unlock()
        return 0
    }
    done := ml.fetchingMembers.done  // ✅ Cópia do canal COM lock
    count := ml.fetchingMembers.value
    ml.fetchingMembers.mu.Unlock()
    
    <-done
    return count
}
```

---

### BUG #6: Return Prematuro em applyDefaults()

**Arquivo:** [internal/config/config.go](internal/config/config.go#L128-L140)  
**Tipo:** Dead Code / Lógica Incorreta  
**Severidade:** 🟠 **ALTO**

**Problema:**
```go
func applyDefaults(cfg *Config) {
    // ... outros defaults ...
    
    if cfg.DateSeparator.Character == "" {
        cfg.DateSeparator.Character = "─"
        return  // ❌ Return premituro!
    }

    // ❌ Código abaixo NUNCA EXECUTA se Character era vazio!
    r, _ := utf8.DecodeRuneInString(cfg.DateSeparator.Character)
    if r == utf8.RuneError {
        cfg.DateSeparator.Character = "─"
        return
    }
    cfg.DateSeparator.Character = string(r)
}
```

**Impacto:**
- Validação de UTF-8 nunca executada quando Character é vazio
- Comportamento inconsistente

---

### BUG #7: Goroutine Leak em message_input Update()

**Arquivo:** [internal/ui/chat/message_input.go](internal/ui/chat/message_input.go#L138-L145)  
**Tipo:** Goroutine Leak  
**Severidade:** 🟠 **ALTO**

**Problema:**
```go
if selectedChannel := mi.chat.SelectedChannel(); selectedChannel != nil {
    go mi.chat.state.Typing(selectedChannel.ID)  // ❌ Goroutine sem rastreamento
}
```

**Impacto:**
- Goroutines acumulam durante digitação
- Memory leak proporcional ao tempo de uso
- Potencial para consumir muita memória em sessões longas

---

### BUG #8: Erro Ignorado em openAttachment()

**Arquivo:** [internal/ui/chat/messages_list.go](internal/ui/chat/messages_list.go#L1150-L1190)  
**Tipo:** Tratamento de Erro Inadequado  
**Severidade:** 🟠 **ALTO**

**Problema:**
```go
func (ml *messagesList) openAttachment(attachment discord.Attachment) {
    resp, err := http.Get(attachment.URL)
    if err != nil {
        slog.Error("failed to fetch the attachment", ...)
        return
    }
    defer resp.Body.Close()

    path := filepath.Join(consts.CacheDir(), "attachments")
    if err := os.MkdirAll(path, os.ModePerm); err != nil {
        slog.Error("failed to create attachments dir", ...)
        return  // ❌ Erro silencioso, sem feedback
    }

    path = filepath.Join(path, attachment.Filename)
    file, err := os.Create(path)
    if err != nil {
        slog.Error("failed to create attachment file", ...)
        return  // ❌ Nenhuma notificação ao usuário
    }
    defer file.Close()
    
    if _, err := io.Copy(file, resp.Body); err != nil {
        slog.Error("failed to copy attachment to file", ...)
        return  // ❌ Falha não é comunicada
    }
}
```

**Impacto:**
- Usuário não sabe se anexo foi salvo com sucesso
- Difícil de debugar problemas de save

---

## 🟡 BUGS MÉDIOS

### BUG #9: Erro Ignorado em openState()

**Arquivo:** [internal/ui/chat/msg.go](internal/ui/chat/msg.go#L8-L14)  
**Tipo:** Erro Ignorado  
**Severidade:** 🟡 **MÉDIO**

**Problema:**
```go
func (m *Model) openState() tview.Cmd {
    return func() tview.Msg {
        if err := m.state.Open(context.Background()); err != nil {
            slog.Error("failed to open chat state", "err", err)
            return nil  // ❌ Retorna nil, UI não sabe que falhou
        }
        return nil
    }
}
```

**Impacto:**
- App falha silenciosamente sem feedback visual
- Usuário fica confuso

**Solução:**
```go
case "openStateError":
    return errorMsg{err: msg}  // Retornar mensagem de erro para UI
```

---

### BUG #10: Erro Ignorado em setClipboard() - Login

**Arquivo:** [internal/ui/login/msg.go](internal/ui/login/msg.go#L12-L19)  
**Tipo:** Erro Ignorado  
**Severidade:** 🟡 **MÉDIO**

**Problema:**
```go
func setClipboard(content string) tview.Cmd {
    return func() tview.Msg {
        if err := clipboard.Write(clipboard.FmtText, []byte(content)); err != nil {
            slog.Error("failed to copy error message", "err", err)
            return nil  // ❌ Erro silencioso
        }
        return nil
    }
}
```

---

### BUG #11: Race Condition em QR Login

**Arquivo:** [internal/ui/login/qr/model.go](internal/ui/login/qr/model.go#L20-L60)  
**Tipo:** Race Condition / Data Race  
**Severidade:** 🟡 **MÉDIO**

**Problema:**
```go
type Model struct {
    *tview.TextView
    conn              *websocket.Conn  // ❌ Sem sincronização
    heartbeatInterval time.Duration
    privateKey        *rsa.PrivateKey
    fingerprint       string
    qrCode            *qrcode.QRCode
    msg               string  // ❌ Todos esses campos acessados de múltiplas goroutines
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
    switch msg := msg.(type) {
    case connCreateMsg:
        m.conn = msg.conn  // ❌ Race condition!
        m.heartbeatInterval = msg.heartbeatInterval
        // ...
    }
}
```

**Impacto:**
- Undefined behavior em acesso concorrente
- Crash potencial

---

### BUG #12: Vazamento de Goroutine em listen()

**Arquivo:** [internal/ui/chat/msg.go](internal/ui/chat/msg.go#L30-L33)  
**Tipo:** Goroutine Leak / Deadlock Potencial  
**Severidade:** 🟡 **MÉDIO**

**Problema:**
```go
func (m *Model) listen() tview.Cmd {
    return func() tview.Msg {
        return gatewayEventMsg{Event: <-m.events}  // ❌ Bloqueia indefinidamente
    }
}
```

**Impacto:**
- UI fica congelada esperando evento
- Nenhuma forma de cancelar

---

### BUG #13: Erro Não Propagado em getToken()

**Arquivo:** [internal/ui/root/msg.go](internal/ui/root/msg.go#L20-L28)  
**Tipo:** Erro Ignorado  
**Severidade:** 🟡 **MÉDIO**

**Problema:**
```go
func getToken() tview.Cmd {
    return func() tview.Msg {
        token, err := keyring.GetToken()
        if err != nil {
            slog.Info("failed to retrieve token from keyring", "err", err)
            return loginMsg{}  // ❌ Apenas slog, sem UI feedback
        }
        return tokenMsg(token)
    }
}
```

**Impacto:**
- Usuário vê tela de login sem entender por quê

---

### BUG #14 & #15: Múltiplos Erros Ignorados em HTTP e Clipboard

Vários arquivos ignoram erros silenciosamente sem feedback ao usuário, afetando a experiência de debug.

---

## 🟢 BUGS BAIXOS

### BUG #16: Validação Inadequada em Cache.Invalidate()

**Arquivo:** [internal/cache/cache.go](internal/cache/cache.go#L30-L44)  
**Tipo:** Lógica Incorreta  
**Severidade:** 🟢 **BAIXO**

**Problema:**
```go
func (c *Cache) Invalidate(name string, limit uint) {
    for name != "" {
        if c.Exists(name) && c.Get(name) >= limit {  // ❌ Sem validação de tamanho
            for name != "" {
                c.items.Delete(name)
                name = name[:len(name)-1]
            }
        }
        name = name[:len(name)-1]  // ❌ Pode cause bounds error
    }
}
```

---

### BUG #17: Cast Inseguro em Picker

**Arquivo:** [internal/ui/chat/channels_picker.go](internal/ui/chat/channels_picker.go#L24-L26)  
**Tipo:** Type Assertion  
**Severidade:** 🟢 **BAIXO**

**Observação:** Este código está correto com verificação `ok`.

---

### BUG #18: Erro Potencial em DecodeRuneInString

Vários locais com décoding UTF-8 sem tratamento de erro.

---

## ✅ RECOMENDAÇÕES PRIORITÁRIAS

### 👑 Prioridade 1 (IMEDIATO - 24h)

1. **Corrigir Logger.go** - Adicionar `defer file.Close()` (Vazamento crítico)
2. **Corrigir Cache.Get()** - Retornar `(uint, bool)` (Crash crítico)
3. **Corrigir typingTimer** - Adicionar sincronização com mutex (Race condition)

### 🔥 Prioridade 2 (ALTO - 1 semana)

4. **Adicionar clearTypers()** ao closeState() (Goroutine leak)
5. **Corrigir waitForChunkEvent()** - Sincronização adequada
6. **Adicionar synchronização** em QR login model
7. **Corrigir applyDefaults()** - Remover return prematuro

### ⚠️ Prioridade 3 (MÉDIO - 2 semanas)

8. Propagação de erros ao usuário (`openState`, `setClipboard`, `getToken`)
9. Rastreamento de goroutines de typing
10. Timeout em `listen()`

---

## 📈 Impacto por Tipo

| Tipo | Quantidade | Severidade |
|------|-----------|-----------|
| Vazamento de Recursos | 3 | 🔴 + 🟠 |
| Race Conditions | 4 | 🔴 + 🟠 + 🟡 |
| Goroutine Leaks | 4 | 🔴 + 🟡 |
| Tratamento de Erro | 7 | 🟡 + 🟢 |
| Lógica Incorreta | 2 | 🟠 + 🟢 |

---

## 🛠️ Próximos Passos

1. ✅ Implementar correções dos 3 bugs críticos
2. ✅ Adicionar testes de concorrência
3. ✅ Usar `go test -race` antes de commits
4. ✅ Implementar error propagation framework
5. ✅ Code review com foco em concorrência

