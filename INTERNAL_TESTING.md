# Freelo CLI — Interní testování

Tento dokument popisuje jak si nainstalovat, otestovat a používat Freelo CLI interně, než ho zveřejníme.

## 1. Instalace (pro testera)

### Varianta A: Z repozitáře (doporučeno)

```bash
# Naklonovat repo
git clone git@github.com:freeloapp/freelo-cli.git
cd freelo-cli

# Build + instalace do ~/bin (bez sudo)
make install

# Ověřit
freelo version
```

> Pokud `freelo` příkaz není nalezen, přidej `~/bin` do PATH:
> ```bash
> echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
> source ~/.zshrc
> ```

### Varianta B: Přímý Go install

```bash
go install github.com/freeloapp/freelo-cli/cmd/freelo@latest
```

> Vyžaduje Go 1.21+. Binary se nainstaluje do `$GOPATH/bin/`.

### Varianta C: Stáhnout binary manuálně

1. Jít na https://github.com/freeloapp/freelo-cli/releases
2. Stáhnout archiv pro svůj OS (darwin_arm64 pro Apple Silicon Mac)
3. Rozbalit a přesunout `freelo` do PATH

---

## 2. Přihlášení

Potřebuješ svůj Freelo API klíč — najdeš ho v:
**https://app.freelo.io/profil/nastaveni**

### Produkce (výchozí)
```bash
freelo auth login
# Email: tvuj@email.com
# API Key: [zadej klíč, nebude vidět]
```

### Devel prostředí
```bash
# Nejdřív nastavit URL dev API (dostaneš od team leada)
export FREELO_DEV_URL=https://tvoje-dev-api-url/v1

# Přidat do ~/.zshrc pro trvalé nastavení:
echo 'export FREELO_DEV_URL=https://tvoje-dev-api-url/v1' >> ~/.zshrc

# Přihlásit se na devel
freelo auth login --dev
# Email: tvuj devel email
# API Key: tvuj devel API klíč
```

Credentials jsou uloženy **odděleně** — můžeš být přihlášený do obou zároveň:
- Produkce: `~/.config/freelo/credentials.json`
- Devel: `~/.config/freelo/credentials-dev.json`

Ověření:
```bash
freelo auth status           # → stav produkce
freelo auth status --dev     # → stav develu
```

### Používání s --dev
Stačí přidat `--dev` k jakémukoliv příkazu:
```bash
freelo projects list --dev       # projekty na develu
freelo tasks list --dev          # úkoly na develu
freelo search "test" --dev       # hledání na develu
```

V terminálu se zobrazí indikátor `[DEV]` s URL, aby bylo jasné proti čemu jedete.

### Pro CI/automatizaci (bez interaktivního loginu):
```bash
export FREELO_EMAIL=tvuj@email.com
export FREELO_API_KEY=tvuj-api-klic
freelo projects list
```

---

## 3. Testovací checklist

Projdi tyto příkazy a ověř, že fungují s tvým účtem:

### Základní operace
```bash
# Ověření přihlášení
freelo auth status

# Tvůj profil
freelo users me

# Seznam spolupracovníků
freelo users list
```

### Projekty
```bash
# Výpis aktivních projektů
freelo projects list

# Detail konkrétního projektu (dosáď ID z výpisu výše)
freelo projects show <ID>
```

### Úkoly
```bash
# Všechny tvé úkoly
freelo tasks list

# Úkoly v konkrétním projektu
freelo tasks list --project <ID>

# Hledání úkolů
freelo tasks list --search "klíčové slovo"

# Detail úkolu
freelo tasks show <TASK_ID>
```

### Vyhledávání
```bash
freelo search "test"
freelo search "bug" --type task
```

### Time tracking
```bash
# Zkontrolovat stav (jestli běží časovač)
freelo tracking status

# Spustit (na nějakém úkolu)
freelo tracking start --task <TASK_ID>

# Zastavit
freelo tracking stop
```

### Výkazy práce
```bash
freelo reports list --project <ID>
```

### Komentáře
```bash
freelo comments list --project <ID>
```

### Štítky, notifikace, další
```bash
freelo labels list
freelo notifications list
freelo notifications list --unread
freelo events list --project <ID>
freelo templates list
freelo custom-fields types
```

### Agent mode (pro AI agenty)
```bash
# Každý příkaz s --agent vrací čistý JSON
freelo projects list --agent
freelo tasks list --project <ID> --agent

# JSON s obálkou (ok, data, summary, breadcrumbs)
freelo projects list --json
```

### Raw API přístup
```bash
# Libovolný endpoint
freelo api get /projects
freelo api post /search --data '{"search_query":"test"}'
```

---

## 4. Testování s AI agentem

### Claude Code

```bash
# Nainstalovat skill
freelo skill install claude
```

Pak otevři **novou konverzaci** v Claude Code a zkus:
- "Ukaž mi moje projekty ve Freelu"
- "Kolik mám aktivních úkolů?"
- "Najdi všechny úkoly s termínem tento týden"
- "Spusť mi časovač na úkolu 12345"

Claude by měl automaticky použít `freelo` příkazy.

### Codex / OpenCode

```bash
freelo skill install codex
freelo skill install opencode
# nebo
freelo skill install all
```

---

## 5. Co testovat a na co dávat pozor

### Funguje správně?
- [ ] Přihlášení a ověření credentials
- [ ] Výpis projektů (vlastní i přizvané)
- [ ] Výpis úkolů s filtry (projekt, hledání, worker)
- [ ] Detail úkolu (včetně komentářů, štítků, custom fields)
- [ ] Vyhledávání
- [ ] Time tracking (start/stop/status)
- [ ] Výpis výkazů práce
- [ ] Agent mode (--agent) vrací čistý parsovatelný JSON
- [ ] JSON mode (--json) vrací obálku s breadcrumbs
- [ ] Chybové hlášky jsou srozumitelné

### Zápis (POZOR — mění data!)
- [ ] Vytvořit testovací úkol: `freelo tasks create --project <ID> --tasklist <ID> --name "Test CLI"`
- [ ] Dokončit ho: `freelo tasks finish <TASK_ID>`
- [ ] Znovu otevřít: `freelo tasks activate <TASK_ID>`
- [ ] Přidat komentář: `freelo comments create --task <ID> --content "Test komentář z CLI"`
- [ ] Vytvořit výkaz: `freelo reports create --task <ID> --minutes 15 --note "Test"`
- [ ] Smazat testovací data po sobě

### Bezpečnost
- [ ] `~/.config/freelo/credentials.json` má oprávnění 600 (pouze vlastník)
- [ ] Heslo/API klíč se nezobrazuje v terminálu při zadávání
- [ ] `freelo auth logout` smaže credentials

---

## 6. Nahlášení problémů

Když najdeš bug nebo něco nefunguje:

1. Zapiš **příkaz** co jsi spustil
2. Zapiš **výstup** (ideálně s `--agent` pro přesný JSON)
3. Zapiš **co jsi očekával**
4. Vytvoř issue na https://github.com/freeloapp/freelo-cli/issues

Příklad:
```
Příkaz: freelo tasks list --project 123 --agent
Výstup: []
Očekávání: Mělo vrátit 15 úkolů (v projektu jich tolik je)
```

---

## 7. Jak aktualizovat na novou verzi

```bash
cd freelo-cli
git pull
make install
```

---

## 8. Přehled příkazů (quick reference)

```
freelo auth login|logout|status
freelo projects list|show|create|archive|activate|delete
freelo tasks list|show|create|edit|finish|activate|move|description
freelo subtasks list|show|create|finish|activate|delete
freelo tasklists list|show|create
freelo search <query>
freelo comments list|create|edit|delete
freelo labels list|create|edit|delete|add-to-task|remove-from-task|add-to-project|remove-from-project
freelo tracking start|stop|status
freelo reports list|create|edit|delete
freelo notes create|show|edit|delete
freelo files list|download|upload
freelo custom-fields types|list|create|rename|delete|restore|set-value|delete-value|enum-*
freelo templates list|create-project|create-tasklist|create-task
freelo pinned list|create|delete
freelo users me|list
freelo workers list|invite|remove
freelo out-of-office status|enable|disable
freelo notifications list|read|unread
freelo invoices list|show|mark-invoiced
freelo events list
freelo api get|post|put|delete <path>
freelo skill show|install
freelo version
```

Každý příkaz podporuje: `--agent` (čistý JSON) | `--json` (JSON s obálkou) | `--quiet` | `--ids-only` | `--count`
