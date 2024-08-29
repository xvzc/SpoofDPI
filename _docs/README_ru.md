**⭐Pull Request-ы или любые формы вклада будут признательны⭐**

# SpoofDPI

Можете прочитать на других языках: [🇬🇧English](https://github.com/xvzc/SpoofDPI), [🇰🇷한국어](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ko.md), [🇨🇳简体中文](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_zh-cn.md), [🇷🇺Русский](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ru.md), [🇯🇵日本語](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ja.md)

Простое и быстрое ПО, созданное для обхода **Deep Packet Inspection**

![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

# Installation
See the installation guide for SpoofDPI [here](https://github.com/xvzc/SpoofDPI/blob/main/_docs/QUICK_START.md).

# Использование
```
Usage: spoofdpi [опции...]
  -addr string
        listen address (default "127.0.0.1")
  -banner
        enable banner (default true)
  -debug
        enable debug output
  -dns-addr string
        dns address (default "8.8.8.8")
  -dns-port int
        port number for dns (default 53)
  -enable-doh
        enable 'dns-over-https'
  -pattern value
        bypass DPI only on packets matching this regex pattern; can be given multiple times
  -port int
        port (default 8080)
  -system-proxy
        enable system-wide proxy (default true)
  -timeout int
        timeout in milliseconds; no timeout when not given
  -v    print spoofdpi's version; this may contain some other relevant information
  -window-size int
        chunk size, in number of bytes, for fragmented client hello,
        try lower values if the default value doesn't bypass the DPI;
        when not given, the client hello packet will be sent in two parts:
        fragmentation for the first data packet and the rest
```
> Если Вы используете любые VPN-расширения по типу Hotspot Shield в браузере
  Chrome, зайдите в Настройки > Расширения и отключите их.

### OSX
Пропишите `spoofdpi` и прокси автоматически установится

### Linux
Пропишите `spoofdpi` и откройте Chrome с параметром прокси
```bash
google-chrome --proxy-server="http://127.0.0.1:8080"
```

# Как это работает
### HTTP
Поскольку большинство веб-сайтов в мире теперь поддерживают HTTPS, SpoofDPI не обходит Deep Packet Inspection для HTTP запросов, однако он по-прежнему обеспечивает прокси-соединение для всех HTTP запросов.

### HTTPS
Хотя TLS шифрует каждый процесс рукопожатия, имена доменов по-прежнему отображаются в виде открытого текста в пакете Client Hello. Другими словами, когда кто-то другой смотрит на пакет, он может легко догадаться, куда направляется пакет. Домен может предоставлять значительную информацию во время обработки DPI, и мы можем видеть, что соединение блокируется сразу после отправки пакета Client Hello.
Я попробовал несколько способов обойти это, и обнаружил, что, похоже, проверяется только первый фрагмент, когда мы отправляем пакет Client Hello, разделенный на фрагменты. Чтобы обойти DPI, SpoofDPI отправляет на сервер первый 1 байт запроса, а затем отправляет все остальное.

# Вдохновение
[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel) от @SadeghHayeri  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) от @ValdikSS
