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

## Возможности

- Обход DPI для YouTube и Discord «из коробки» (готовые списки доменов).
- Несколько стратегий десинхронизации: `split`, `disorder`, `fake`, `fakedsplit`.
- Подавление QUIC, чтобы браузер откатывался на TLS-over-TCP (который обходится).
- Запуск в один клик (`.bat`-пресеты) и установка как служба Windows (автозапуск).
- Не трогает посторонний трафик — работает только по целевым доменам.
- **Fail-open**: если что-то пошло не так, пакет отправляется без изменений —
  интернет не «падает».

## Быстрый старт

1. Скачайте архив `skvoz-<версия>-windows-amd64.zip` со страницы
   [Releases](../../releases) и распакуйте.
2. Дважды кликните нужный `.bat` (запросит права администратора):
   - `youtube.bat` — разблокировать YouTube;
   - `discord.bat` — разблокировать Discord;
   - `general.bat` — то и другое сразу.
3. Оставьте окно открытым. Готово — открывайте YouTube/Discord.

Чтобы Skvoz запускался автоматически при загрузке Windows:
`service-install.bat` (удалить — `service-uninstall.bat`).

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
но настоящий сервер собирает всё корректно.

## Использование из командной строки

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

Если один провайдер блокирует иначе — попробуйте другую стратегию, например
`--strategy fake --fake-ttl 6`.

## Сборка из исходников

Нужен Go 1.22+. `skvoz.exe` кросс-компилируется с любой ОС:

```bash
make build-windows          # -> dist/skvoz.exe
make test                   # юнит-тесты (запускаются на любой ОС)
scripts/package.sh v1.0.0   # собрать полный релизный zip (тянет WinDivert)
```

Для запуска локально собранного `skvoz.exe` положите рядом `WinDivert.dll` и
`WinDivert64.sys` (WinDivert 2.2.x) — см. `third_party/windivert/README.md`.

## Ограничения v1

- Только Windows (Linux/NFQUEUE — в планах).
- Голосовой трафик Discord (UDP по IP без SNI) не покрывается.
- Нет GUI (пока только CLI и `.bat`).

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

1. Download `skvoz-<version>-windows-amd64.zip` from [Releases](../../releases)
   and unzip it.
2. Double-click a preset (it will request administrator rights):
   `youtube.bat`, `discord.bat`, or `general.bat` (both).
3. Keep the window open and browse. To auto-start on boot, run
   `service-install.bat` (`service-uninstall.bat` to remove).

### How it works

DPI identifies a blocked site by the host name in the TLS ClientHello (the SNI
field). Skvoz splits or spoofs that packet so DPI cannot reassemble the name,
while the real server still reconstructs the stream correctly. QUIC is suppressed
so browsers fall back to the TLS-over-TCP path Skvoz handles.

### Command line

```
skvoz.exe --lists lists\list-youtube.txt,lists\list-discord.txt
skvoz.exe --strategy fake --fake-ttl 6 --lists lists\list-discord.txt
```

See the flag table above (identical to the Russian section).

### Building

Requires Go 1.22+. `skvoz.exe` cross-compiles from any OS:

```bash
make build-windows
make test
scripts/package.sh v1.0.0
```

### License

Skvoz’s code is [MIT](LICENSE). WinDivert ships under its own LGPLv3/GPLv3
license — see [NOTICE](NOTICE).
