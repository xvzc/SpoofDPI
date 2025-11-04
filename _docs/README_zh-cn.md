# SpoofDPI

选择语言: [🇬🇧English](https://github.com/xvzc/SpoofDPI), [🇰🇷한국어](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ko.md), [🇨🇳简体中文](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_zh-cn.md), [🇷🇺Русский](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ru.md), [🇯🇵日本語](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ja.md)



规避**深度包检测**的简单工具

```txt
 ❯ spoofdpi

 .d8888b.                              .d888 8888888b.  8888888b. 8888888
d88P  Y88b                            d88P"  888  "Y88b 888   Y88b  888
Y88b.                                 888    888    888 888    888  888
 "Y888b.   88888b.   .d88b.   .d88b.  888888 888    888 888   d88P  888
    "Y88b. 888 "88b d88""88b d88""88b 888    888    888 8888888P"   888
      "888 888  888 888  888 888  888 888    888    888 888         888
Y88b  d88P 888 d88P Y88..88P Y88..88P 888    888  .d88P 888         888
 "Y8888P"  88888P"   "Y88P"   "Y88P"  888    8888888P"  888       8888888
           888
           888
           888

 • LISTEN_ADDR : 127.0.0.1
 • LISTEN_PORT : 8080
 • DNS_ADDR    : 8.8.8.8
 • DNS_PORT    : 53
 • DEBUG       : false

Press 'CTRL + c' to quit
```

# Installation
See the installation guide for SpoofDPI [here](https://github.com/xvzc/SpoofDPI/blob/main/_docs/INSTALL.md).

<a href="https://repology.org/project/spoofdpi/versions">
    <img src="https://repology.org/badge/vertical-allrepos/spoofdpi.svg?columns=1" alt="Packaging status">
</a>  

# 使用方法

```
Usage: spoofdpi [options...]
  -cache-shards uint
        number of shards to use for ttlcache; it is recommended to set this to be >= the number of CPU cores for optimal performance (default 32)
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
  -listen-addr string
        IP address to listen on (default "127.0.0.1")
  -listen-port value
        port number to listen on (default 8080)
  -pattern value
        bypass DPI only on packets matching this regex pattern; can be given multiple times
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

> 如果你在 Chrome 浏览器使用其他 VPN 扩展比如 Hotspot Shield 请去 设置 > 扩展程序禁用它们

### OSX
运行 `spoofdpi` ，然后它会自动设置自身为代理

### Linux
运行 `spoofdpi` 然后加上代理参数运行你的浏览器
```bash
google-chrome --proxy-server="http://127.0.0.1:8080"
```

# 工作原理

### HTTP

因为世界上许多网站都已支持 HTTPS ，SpoofDPI 不会规避对 HTTP 请求的 DPI，但是它仍会为 HTTP 请求提供代理。

### HTTPS
尽管 TLS 加密了握手的每一步，但是在 Client Hello 中的域名仍然是明文的。因此如果有人看到 Client Hello 包就可以知道你在连接什么网站。这给 DPI 提供了很大方便，我们也看到连接在 Client Hello 之后就会被屏蔽掉。我之前尝试了规避这种审查，并发现，如果把 Client Hello 分包，只有第一个 chunk 会被检测。SpoofDPI 只要在第一个分包发送 1 byte，然后再发送其他部分就能规避。

# 启发

[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel) by @SadeghHayeri  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) by @ValdikSS
