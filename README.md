# Skvoz

**Skvoz** («сквозь») — свободный инструмент обхода DPI-блокировок для Windows.
Он восстанавливает доступ к **YouTube** и **Discord** без VPN и прокси: трафик
никуда не туннелируется и не шифруется — Skvoz лишь на лету изменяет ваши
исходящие пакеты так, что система DPI перестаёт их распознавать и блокировать.

Идейный наследник `zapret` / `winws`, написанный с нуля на Go под лицензией MIT.

> ⚠️ Skvoz — инструмент против цензуры для доступа к легальным сервисам.
> Вы сами отвечаете за соблюдение законов вашей страны.

*(English version below — [see “English”](#english))*

---

## Быстрый старт

1. Скачайте **один файл** `skvoz.exe` со страницы
   [Releases](../../releases).
2. Дважды кликните по нему.
   - Windows покажет синее окно **«Windows защитила ваш компьютер»** —
     это потому, что файл не подписан платным сертификатом. Нажмите
     **«Подробнее» → «Всё равно запустить»**. Предупреждение бывает один раз.
   - Разрешите права администратора (**«Да»** в окне UAC) — они нужны, чтобы
     перехватывать пакеты.
3. В системном трее (у часов) появится значок Skvoz. Всё — открывайте
   YouTube/Discord. Skvoz сам подберёт рабочую стратегию обхода.

Никакой распаковки и никаких `.bat` — драйвер и списки доменов уже внутри exe.

### Значок в трее

Правый клик по значку открывает меню:

```
● Skvoz — работает ✓ (fakedsplit)
────────────────
☑ YouTube
☑ Discord
────────────────
☐ Запускать с Windows
Проверить сейчас
────────────────
Выход
```

- **YouTube / Discord** — что разблокировать. Изменения применяются сразу.
- **Запускать с Windows** — автозапуск при входе в систему (по умолчанию
  выключен). Включается одной галкой, повторный UAC при загрузке не требуется.
- **Проверить сейчас** — заново подобрать рабочую стратегию, если провайдер
  поменял способ блокировки.

## Как это работает

```
Исходящий пакет ─▶ WinDivert перехватывает
        ─▶ это TLS ClientHello (TCP 443/80) или QUIC Initial (UDP 443)?
        ─▶ достаём SNI (имя сайта) из пакета
        ─▶ SNI в списке целей (youtube/discord)?
             ├─ да  ─▶ применяем стратегию десинхронизации ─▶ реинжект
             └─ нет ─▶ отправляем как есть
```

DPI распознаёт заблокированный сайт по имени в TLS ClientHello (поле SNI).
Skvoz разрезает или «подделывает» этот пакет так, что DPI не может собрать имя,
но настоящий сервер собирает всё корректно. При старте Skvoz пробует несколько
стратегий (`fakedsplit`, `fake`, `split`, `disorder`) и оставляет ту, что
реально открывает доступ.

## Почему Windows ругается на файл

Skvoz не подписан платным code-signing сертификатом (проект бесплатный), поэтому
SmartScreen и часть антивирусов могут показывать предупреждение. Код открыт —
можете собрать `skvoz.exe` сами (см. ниже) и сверить контрольную сумму с
`skvoz.exe.sha256` со страницы Releases.

<details>
<summary><b>Расширенно: командная строка и служба</b></summary>

Тот же `skvoz.exe`, запущенный **с флагами**, работает в режиме CLI (без трея):

```
skvoz.exe --lists lists\list-youtube.txt,lists\list-discord.txt
```

| Флаг | Значение | По умолчанию |
|---|---|---|
| `--strategy` | `split` \| `disorder` \| `fake` \| `fakedsplit` | `fakedsplit` |
| `--fake-ttl` | TTL для «фейковых» пакетов | `8` |
| `--lists` | списки доменов через запятую | — |
| `--quic` | `drop` (гасить QUIC) \| `off` | `drop` |
| `--split` | точка разреза: `sni` \| `middle` | `sni` |
| `--ports` | целевые TCP-порты | `80,443` |
| `--service` | `install` \| `uninstall` \| `run` | — |

В CLI-режиме нужны файлы списков рядом с exe (в репозитории — папка `lists/`).
`--service install` регистрирует Skvoz как службу Windows.

</details>

## Сборка из исходников

Нужен Go 1.22+. `skvoz.exe` кросс-компилируется с любой ОС:

```bash
make test                   # юнит-тесты (запускаются на любой ОС)
scripts/package.sh v2.0.0   # -> dist/skvoz.exe (+ .sha256); тянет WinDivert
```

`scripts/package.sh` скачивает драйвер WinDivert, встраивает его вместе со
списками доменов в один самодостаточный `skvoz.exe` и считает контрольную сумму.

## Ограничения

- Только Windows (Linux/NFQUEUE — в планах).
- Голосовой трафик Discord (UDP по IP без SNI) не покрывается.
- Файл не подписан — возможны предупреждения SmartScreen/антивируса.

## Лицензия

Код Skvoz — [MIT](LICENSE). WinDivert поставляется под своей лицензией
(LGPLv3/GPLv3), см. [NOTICE](NOTICE).

---

## English

**Skvoz** (Russian for “through”) is a free, open-source DPI-bypass tool for
Windows that restores access to **YouTube** and **Discord**. It is not a VPN or
proxy — no traffic is tunneled or encrypted. Skvoz rewrites your own outbound
packets on the fly so DPI can no longer classify and block them. A from-scratch,
MIT-licensed spiritual successor to `zapret` / `winws`.

> ⚠️ Skvoz is an anti-censorship tool for accessing lawful services. You are
> responsible for complying with your local laws.

### Quick start

1. Download the single file **`skvoz.exe`** from [Releases](../../releases).
2. Double-click it.
   - Windows will show a blue **“Windows protected your PC”** dialog because the
     file isn’t signed with a paid certificate. Click **“More info” → “Run
     anyway”** (shown once).
   - Approve the administrator prompt (**“Yes”** in UAC) — required to capture
     packets.
3. A Skvoz icon appears in the system tray. Done — open YouTube/Discord. Skvoz
   auto-selects a working bypass strategy.

No unzip and no `.bat` files — the driver and domain lists are already inside
the exe.

### Tray menu

Right-click the tray icon:

- **YouTube / Discord** — which services to unblock (applied instantly).
- **Start with Windows** — auto-start at logon (off by default; one checkbox, no
  extra UAC prompt on boot).
- **Check now** — re-select a working strategy if your ISP changed its blocking.

### Why Windows warns about the file

Skvoz isn’t signed with a paid code-signing certificate (it’s a free project), so
SmartScreen and some antivirus tools may warn. The code is open — build
`skvoz.exe` yourself and verify it against the `skvoz.exe.sha256` on the Releases
page.

### Advanced: CLI & service

Run the same `skvoz.exe` **with flags** for headless CLI mode; see the flags
table in the Russian section above. `--service install` registers it as a
Windows service.

### Build from source

Go 1.22+. `scripts/package.sh v2.0.0` fetches WinDivert, embeds it with the
domain lists into one self-contained `dist/skvoz.exe`, and writes a checksum.
`make test` runs the unit tests on any OS.

### License

Skvoz’s code is [MIT](LICENSE). WinDivert ships under its own LGPLv3/GPLv3
license — see [NOTICE](NOTICE).
