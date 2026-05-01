# Freelo CLI — interní zpráva a testování

Krátký dokument o tom, co jsme postavili, jak to funguje, jak se to bude
samo aktualizovat, jak s tím pracovat a co teď chceme od interních
testerů. Verze CLI v době psaní: **v1.0.0-dev**, zatím v privátním repu
[`freeloio/freelo-cli`](https://github.com/freeloio/freelo-cli).

---

## 1. Co to je

`freelo` je oficiální command-line interface pro Freelo.io. Ovládá celé
Freelo z terminálu — jak ručně z příkazové řádky, tak přes AI agenty
(Claude Code, Codex, OpenCode).

- **27 command skupin**, ~70 endpointů — projekty, tasklisty, úkoly,
  subúkoly, komentáře, výkazy, štítky, custom fields, soubory,
  vyhledávání, audit log, OOO, invoices, …
- **Postaveno na oficiálním Freelo OpenAPI specu** (žádné ručně
  zadrátované URL — stačí říct co chceš a typed klient sestaví request)
- **Funguje pro lidi i pro agenty** v jednom binary — default je hezký
  terminálový výstup, `--agent` přepne na čistý parsovatelný JSON
- **Bundlovaný skill pro AI** — jeden příkaz a Claude Code ví všechno o
  Freelu a používá `freelo` automaticky

## 2. Jak to funguje pod kapotou

```
tvůj příkaz  →  freelo CLI
                    │
                    ▼
            wrapper (auth + User-Agent + rate limit + retry)
                    │
                    ▼
            generovaný klient (z OpenAPI specu)
                    │
                    ▼
            api.freelo.io
```

- **Auth**: env vars `FREELO_EMAIL` + `FREELO_API_KEY` mají přednost
  (CI, agenty), jinak credentials z **OS keyringu** (Keychain na macOS,
  Credential Manager na Windows, Secret Service na Linux desktopu).
  Pro headless prostředí (Docker bez DBus, server bez UI) je escape
  `FREELO_KEYRING=file` → JSON soubor 0600 v `~/.config/freelo/`.
- **Rate limit**: ~2.4 s mezi voláními → nikdy se nedostaneš na 25/min
  Freelo limit
- **Retry**: 429 / 5xx → 3× exponenciální backoff s jitterem,
  respektuje `Retry-After`
- **User-Agent**: `FreeloCLI/<verze>` — backend si tak může měřit, kdo
  CLI používá

## 3. Jak se to bude samo aktualizovat na změny v API

Toto byl jeden z hlavních důvodů celé refaktorizace.

Od v1.1.0 generovaný klient žije v sibling repu
[`github.com/freeloio/freelo-go`](https://github.com/freeloio/freelo-go).
CRON na regeneraci běží **tam**, ne v tomto repu:

```
každé pondělí 6:00 UTC (v repu freelo-go)
        │
        ▼
GitHub Actions: update-api-spec.yml
        │
        ▼
1. curl https://api.freelo.io/docs/v1/freelo-api.yaml
2. patch (Client → BusinessClient — kvůli kolizi názvů)
3. go generate → nový klient
4. patchgen (time.Time → freelotime.Time)
5. go build + go test + go build ./examples/...
        │
        ▼
Pokud spec ≠ vendored:  otevře PR v freelo-go
Pokud spec = stejný:    no-op
```

Po sloučení PR ve freelo-go se otaguje nová verze SDK. Tady (CLI) pak
udělej:

```bash
go get github.com/freeloio/freelo-go@v0.x.y
go mod tidy
make test
```

V pondělí ráno přijde notifikace o PR ve freelo-go, projedeš diff:

| Druh změny v API | Ruční práce ve freelo-go | Ruční práce v CLI |
|---|---|---|
| Nový optional parametr | Smerguj | Bump SDK pin, smerguj |
| Nový endpoint | Smerguj | Pokud chceš command, dopiš ho |
| Přejmenování pole / breaking change | Smerguj (typed builds doplníš) | Bump SDK pin, oprav 2-3 commandy |
| Smazaný endpoint | Smerguj | Smaž odpovídající command |

Frekvence změn ve Freelo specu je nízká, většinou půjde jen o smerge +
SDK bump.

## 4. Instalace

### Pro testera (z repa)

```bash
# Klonovat
git clone git@github.com:freeloio/freelo-cli.git
cd freelo-cli

# Build + instalace do ~/bin
make install

# Ověřit
freelo version
```

Pokud `~/bin` není v PATH:
```bash
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

### Alternativy

- `go install github.com/freeloio/freelo-cli/cmd/freelo@latest` (vyžaduje
  Go 1.24+)
- Stáhnout binary z [GitHub Releases](https://github.com/freeloio/freelo-cli/releases)
  (zatím prázdné, dokud nevyjde v1.0.0)

### Aktualizace na novou verzi
```bash
cd freelo-cli && git pull && make install
```

## 5. Přihlášení

API klíč najdeš na [app.freelo.io/profil/nastaveni](https://app.freelo.io/profil/nastaveni).

```bash
freelo auth login
# Email: tvuj@email.com
# API Key: <vloží se nezobrazí>
```

Credentials se uloží do OS keyringu (na macOS uvidíš Keychain prompt,
to je v pořádku — povol). Stav:
```bash
freelo auth status
```

### Pro CI / agenty / sandboxy
```bash
export FREELO_EMAIL=tvuj@email.com
export FREELO_API_KEY=tvuj-klic
freelo projects list
```

### Headless Linux / Docker bez DBus
```bash
export FREELO_KEYRING=file
freelo auth login
# uloží do ~/.config/freelo/credentials.json (0600)
```

### Dev prostředí (separátní credentials)
```bash
export FREELO_DEV_URL=https://dev-api.example.com/v1
freelo auth login --dev      # zvlášť uložené
freelo projects list --dev   # všechno s --dev jde proti devu
```

## 6. Jak s tím pracovat — quick tour

### Pro lidi
```bash
freelo projects list                              # tvoje projekty
freelo tasks list --project 12345                 # úkoly v projektu
freelo tasks show 29576272                        # detail úkolu
freelo tasks create --project 12345 --tasklist 67890 --name "Něco"
freelo tasks edit 29576272 --priority h --due-date 2026-12-31
freelo tasks finish 29576272
freelo search "PRD"                               # napříč vším
freelo tracking start --task 29576272
freelo tracking stop
```

### Pro AI agenty
```bash
freelo skill install claude        # nebo codex / opencode / all
```

Pak otevři **novou konverzaci** v Claude Code a mluv normálně:

> "Ukaž mi moje aktivní úkoly v projektu Marketing s deadline tento
> týden a vytvoř výkaz na 90 minut na ten nejstarší."

Claude si sám zavolá správné `freelo` příkazy a odpoví linkem.

### Output módy (užitečné pro skripty)
```bash
freelo projects list                  # default (TTY: tabulka, pipe: JSON)
freelo projects list --agent          # raw JSON, nejvhodnější pro agenty
freelo projects list --json           # obálka {ok, data, summary, breadcrumbs}
freelo projects list --ids-only       # jen IDčka, řádek po řádku
freelo projects list --count          # jen počet
freelo projects list --quiet          # minimum textu
```

### Escape hatch pro endpointy bez dedikovaného commandu
```bash
freelo api get /archived-projects --agent
freelo api post /search --data '{"search_query":"X"}' --agent
freelo api delete /task/12345 --agent
```

## 7. Co teď chceme po testerech

Nedávno proběhlo komplexní E2E testování (117 case, 11 kol — viz
[TESTING_REPORT_v1.0.0.md](TESTING_REPORT_v1.0.0.md)). Našli jsme a
opravili **6 bugů**. Teď chceme **lidský feedback** od reálného použití
před public launchem.

### Cíl interního testování
1. Ověřit, že CLI funguje **na vašich účtech** (jiných než `info@byurban.cz`)
2. Najít **edge cases** specifické pro různé typy projektů (velké, malé,
   archivované, různé custom fields, paid-plan funkce, …)
3. Ověřit, že **AI integrace** je užitečná v reálných úlohách
4. Najít cokoliv, co je **nelibné na UX / pojmenování příkazů / dokumentaci**
   ještě před tím, než to půjde ven

### Test plán (cca 30 minut)

#### Fáze 1 — instalace + login (5 min)
- [ ] Klonovat repo, `make install`, `freelo version` vypíše `v1.0.0-dev`
- [ ] `freelo auth login` na svém účtu, OS keyring nepoptá heslo opakovaně
- [ ] `freelo auth status` ukáže tvoje jméno + email

#### Fáze 2 — bezpečné read-only příkazy (10 min)
Projedi tyto a checkni, že to dává smysl:
```bash
freelo users me
freelo projects list
freelo projects show <ID-jednoho-projektu>
freelo tasks list --project <ID>
freelo tasks list --search "klíčové slovo"
freelo tasklists list --project <ID>
freelo workers list --project <ID>
freelo comments list --project <ID>
freelo reports list --project <ID>
freelo notifications list
freelo events list --project <ID>
freelo search "něco"
```

Pozoruj: dává to smysl? Chybí tam něco? Je rychlost OK? Jsou error
hlášky srozumitelné?

#### Fáze 3 — write paths na throwaway projektu (10 min)
**Vytvoř si vlastní testovací projekt** (`freelo projects create --name
"[Test CLI] můj test"`), neboj sahat na produkční data:
```bash
PID=<id-tvého-test-projektu>

# Plný lifecycle úkolu
freelo tasklists create --project $PID --name "TL" --agent
TLID=<vrácené-id>
freelo tasks create --project $PID --tasklist $TLID --name "task 1" --priority h
TASKID=<vrácené-id>
freelo tasks edit $TASKID --due-date 2026-12-31
freelo tasks description $TASKID --set "<p>popis</p>"
freelo tasks finish $TASKID
freelo tasks activate $TASKID

# Komentáře + soubory
freelo comments create --task $TASKID --content "test komentář"
freelo files upload /tmp/cokoliv.txt          # → vrátí UUID
freelo comments create --task $TASKID --content "s přílohou" --file <UUID>
freelo files download <UUID> --output /tmp/zpet.txt

# Time tracking
freelo tracking start --task $TASKID --note "test session"
freelo tracking stop

# Výkaz
freelo reports create --task $TASKID --minutes 30 --note "test"

# Cleanup
freelo projects archive $PID
freelo projects delete $PID
```

#### Fáze 4 — AI integrace (5 min)
```bash
freelo skill install claude
```
Otevři **novou konverzaci** v Claude Code a zkus 2-3 vlastní úlohy:
- "Kolik aktivních úkolů mám v projektu X?"
- "Vytvoř mi úkol 'Y' v projektu Z, prioritu vysokou, deadline příští pátek"
- "Najdi všechny komentáře z minulého týdne, kde se mluví o 'PRD'"

Pozoruj jestli Claude:
- správně používá `freelo` příkazy
- používá `--agent` flag pro JSON
- odkazy na úkoly/projekty jsou klikatelné
- chyby jsou srozumitelné

### Co když najdeš bug

Vytvoř issue na [github.com/freeloio/freelo-cli/issues](https://github.com/freeloio/freelo-cli/issues)
s:
1. **Příkaz** který jsi spustil
2. **Co se stalo** (ideálně výstup s `--agent` pro přesný JSON)
3. **Co jsi čekal**
4. **OS** (macOS / Linux / WSL)

Příklad:
```
Příkaz:    freelo tasks list --project 591279 --agent
Výstup:    [] (prázdno)
Očekávání: V projektu mám 23 aktivních úkolů, viděl jsem ve webu
OS:        macOS Apple Silicon
```

### Destruktivní operace — opatrně

Tyto **neodzkoušej** na produkčních datech (skutečně mění stav nebo
posílají emaily):
- `freelo workers invite` → odešle pozvánkový email
- `freelo workers remove` → odebere uživatele z projektu
- `freelo invoices mark-invoiced` → ovlivňuje fakturační artefakty
- `freelo projects delete` → mazání je nevratné

## 8. Známé limitace

- **Custom fields lifecycle** vyžaduje placený Freelo plán (free tier
  vrátí 402 "Payment required").
- **Některé subcommandy z legacy v0.1.0 jsou pryč** protože je Freelo
  API nikdy nepodporovalo (subtasks `show/finish/activate/delete`,
  `comments delete`, edit/delete tasklistů). Vrací 404 z API, takže CLI
  je teď nenabízí.
- **Notes neumí přílohy** (Freelo backend `files` field na poznámkách
  silently ignoruje — známý quirk).

## 9. Přehled skupin příkazů

```
auth          login | logout | status
users         me | list
workers       list | invite | remove
projects      list | show | create | archive | activate | delete
tasklists     list | show | create
tasks         list | show | create | edit | finish | activate |
              move | description
subtasks      list | create
comments      list | create | edit
labels        list | create | add-to-task | remove-from-task |
              add-to-project | remove-from-project | edit | delete
custom-fields types | list | create | rename | delete | restore |
              set-value | delete-value | enum-* | set-enum-value
notes         create | show | edit | delete
tracking      start | stop | status
reports       list | create | edit | delete
files         list | upload | download
templates     list | create-project | create-tasklist | create-task
pinned        list | create | delete
notifications list | read | unread
events        list
out-of-office status | enable | disable
invoices      list | show | mark-invoiced
search        <query> [--type ...] [--project ...]
api           get | post | put | delete <path>
skill         show | install
version
```

Detaily: `freelo <skupina> --help` a `freelo <skupina> <command> --help`.

---

Díky za testování. Pošli prosím feedback do issues nebo přímo Markovi
([marek.urban@freelo.io](mailto:marek.urban@freelo.io)).
