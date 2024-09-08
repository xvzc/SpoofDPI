# SpoofDPI

Можете прочитать на других языках: [🇬🇧English](https://github.com/xvzc/SpoofDPI), [🇰🇷한국어](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ko.md), [🇨🇳简体中文](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_zh-cn.md), [🇷🇺Русский](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ru.md), [🇯🇵日本語](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ja.md)

Простое и быстрое ПО, созданное для обхода **Deep Packet Inspection**

![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

# Installation
Инструкции по установке SpoofDPI вы можете найти [здесь](https://github.com/xvzc/SpoofDPI/blob/main/_docs/INSTALL.md).

<a href="https://repology.org/project/spoofdpi/versions">
    <img src="https://repology.org/badge/vertical-allrepos/spoofdpi.svg?columns=1" alt="Packaging status">
</a>  

# Использование
```
Usage: spoofdpi [опции...]
  -addr string
        listen address (default "127.0.0.1")
  -debug
        enable debug output
  -dns-addr string
        dns address (default "8.8.8.8")
  -dns-ipv4-only
        resolve only version 4 addresses
  -dns-port value
        port number for dns (default 53)
  -enable-doh
        enable 'dns-over-https'
  -pattern value
        bypass DPI only on packets matching this regex pattern; can be given multiple times
  -port value
        port (default 8080)
  -silent
        do not show the banner and server information at start up
  -system-proxy
        enable system-wide proxy (default true)
  -timeout value
        timeout in milliseconds; no timeout when not given
  -v    print spoofdpi's version; this may contain some other relevant information
  -window-size value
        chunk size, in number of bytes, for fragmented client hello,
        try lower values if the default value doesn't bypass the DPI;
        when not given, the client hello packet will be sent in two parts:
        fragmentation for the first data packet and the rest
```
> Если Вы используете любые VPN-расширения по типу Hotspot Shield в браузере
Chrome, зайдите в Настройки > Расширения и отключите их.

### OSX
Выполните команду  `spoofdpi` и прокси будет сконфигурирован автоматически

### Linux
Выполните команду `spoofdpi` и откройте Chrome с параметром прокси:
```bash
google-chrome --proxy-server="http://127.0.0.1:8080"
```

# Как это работает
### HTTP
Поскольку большинство веб-сайтов работают поверх HTTPS, SpoofDPI не обходит Deep Packet Inspection для HTTP запросов, однако он по-прежнему обеспечивает проксирование для всех запросов по HTTP.

### HTTPS
Несмотря на то, что шифрование используется в TLS даже во время установки соединения, имена доменов по-прежнему пересылаются в открытом виде в пакете Client Hello. Другими словами, когда кто-то посторонний смотрит на пакет, он может легко понять, куда этот пакет направляется. Доменное имя может предоставить важную информацию во время обработки DPI, и видно, что соединение блокируется сразу после отправки пакета Client Hello.
Я попробовал несколько способов обойти это и обнаружил, что, похоже, когда мы отправляем пакет Client Hello, разделенный на фрагменты, проверяется только первый фрагмент. Поэтому, чтобы обойти DPI, SpoofDPI отправляет на сервер первый 1 байт запроса, а затем отправляет все остальное.

# Проекты, повлиявшие на SpoofDPI
[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel) от @SadeghHayeri  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) от @ValdikSS
