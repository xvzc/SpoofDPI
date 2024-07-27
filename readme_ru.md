**⭐Pull Request-ы или любые формы вклада будут признательны⭐**

# SpoofDPI

Можете прочитать на других языках: [🇬🇧English](https://github.com/xvzc/SpoofDPI), [🇰🇷한국어](https://github.com/xvzc/SpoofDPI/blob/main/readme_ko.md), [🇨🇳简体中文](https://github.com/xvzc/SpoofDPI/blob/main/readme_zh-cn.md), [🇷🇺Русский](https://github.com/xvzc/SpoofDPI/blob/main/readme_ru.md)

Простое и быстрое программное обеспечение, созданное для обхода **Deep Packet Inspection**  
  
![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

# Установка
## Бинарник
SpoofDPI будет установлен в директорию `~/.spoof-dpi/bin`.  
Чтобы запустить SpoofDPI в любой директории, добавьте строку ниже в `~/.bashrc || ~/.zshrc || ...`
```
export PATH=$PATH:~/.spoof-dpi/bin
```

```bash
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s darwin-amd64
```
```bash
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-amd64
```
```bash
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-arm
```
```bash
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-arm64
```
```bash
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-mips
```
```bash
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-mipsle
```

## Go
Вы также можете установить SpoofDPI с помощью **go install**  
`$ go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi`  
  > Не забудьте, что $GOPATH должен быть установлен в Вашем $PATH

## Git
Вы также можете собрать SpoofDPI

`$ git clone https://github.com/xvzc/SpoofDPI.git`  
`$ cd SpoofDPI`  
`$ go build ./cmd/...`  

# Использование
```
Usage: spoof-dpi [options...]
  -addr string
        listen address (default "127.0.0.1")
  -debug
        enable debug output
  -dns-addr string
        dns address (default "8.8.8.8")
  -dns-port int
        port number for dns (default 53)
  -enable-doh
        enable 'dns over https'
  -no-banner
        disable banner
  -pattern string
        bypass DPI only on packets matching this regex pattern
  -port int
        port (default 8080)
  -timeout int
        timeout in milliseconds (default 2000)
  -url value
        Bypass DPI only on this url, can be passed multiple times
  -v    print spoof-dpi's version. this may contain some other relevant information
  -window-size int
        chunk size, in number of bytes, for fragmented client hello,
        try lower values if the default value doesn't bypass the DPI;
        set to 0 to use old (pre v0.10.0) client hello splitting method:
        fragmentation for the first data packet and the rest (default 50)
```
**Перевод:**
```
Использование: spoof-dpi [параметры...]
--addr=<адрес>       | Адрес. По умолчанию 127.0.0.1
--dns=<адрес>        | Адрес DNS-сервера. По умолчанию 8.8.8.8
--port=<порт>        | Порт. По умолчанию 8080
--debug=<булев>      | Включать ли режим отладки. По умолчанию false
--banner=<булев>     | По умолчанию true
--url=<url>          | Можно использовать несколько раз. Если 
                     | задано, будет применятся
                     | обход только для данного url.
                     | Пример: --url=google.com --url=github.com
--pattern=<regex>    | Если задано, будет применятся обход
                     | только для пакетов, которые соответствуют
                     | этому регулярному выражению.
                     | Пример: --pattern="google|github"
```
> Если Вы используете любые "VPN"-расширения по типу Hotspot Shield в браузере  
  Chrome, зайдите в Настройки > Расширения и отключите их.

### OSX
Выполните `$ spoof-dpi` и прокси автоматически установится

### Linux
Выполните `$ spoof-dpi` и откройте свой любимый браузер с параметром прокси
`google-chrome --proxy-server="http://127.0.0.1:8080"`

# Как это работает
### HTTP
Поскольку большинство веб-сайтов в мире теперь поддерживают HTTPS, SpoofDPI не обходит Deep Packet Inspection для HTTP-запросов, однако он по-прежнему обеспечивает прокси-соединение для всех HTTP-запросов.

### HTTPS
Хотя TLS 1.3 шифрует каждый процесс рукопожатия, имена доменов по-прежнему отображаются в виде открытого текста в пакете Client Hello. Другими словами, когда кто-то другой смотрит на пакет, он может легко догадаться, куда направляется пакет. Доменное имя может предоставлять значительную информацию во время обработки DPI, и мы можем видеть, что соединение блокируется сразу после отправки пакета Client Hello. Я попробовал несколько способов обойти это, и обнаружил, что, похоже, только первый фрагмент проверяется, когда мы отправляем пакет Client Hello, разделенный на фрагменты. Чтобы обойти это, SpoofDPI отправляет на сервер первый 1 байт запроса, а затем отправляет все остальное.
 > SpoofDPI не расшифровывает Ваши HTTPS-запросы, так что нам не нужны SSL-сертификаты.

# Вдохновлено
[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel) от @SadeghHayeri  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) от @ValdikSS
