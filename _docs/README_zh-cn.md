**⭐PRs or any form of contribution will be appreciated⭐**

# SpoofDPI

选择语言: [🇬🇧English](https://github.com/xvzc/SpoofDPI), [🇰🇷한국어](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ko.md), [🇨🇳简体中文](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_zh-cn.md), [🇷🇺Русский](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ru.md), [🇯🇵日本語](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ja.md)



规避**深度包检测**的简单工具

![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

# 安装
## Binary

SpoofDPI 会被安装在 `~/.spoof-dpi/bin`
要在其他目录下运行，请给 `~/.bashrc || ~/.zshrc || ...` 添加

```
export PATH=$PATH:~/.spoof-dpi/bin
```
---
```bash
# macOS Intel
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s darwin-amd64

# macOS Apple Silicon
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s darwin-arm64

# linux-amd64
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-amd64

# linux-arm
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-arm

# linux-arm64
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-arm64

# linux-mips
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-mips

# linux-mipsle
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux-mipsle
```

## Go
也可以用 `go install` 安装
```bash
$ go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi@latest
```

## Git
You can also build your own
```bash
$ git clone https://github.com/xvzc/SpoofDPI.git
$ cd SpoofDPI
$ go build ./cmd/...
```

# 使用方法

```
Usage: spoof-dpi [options...]
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
  -v    print spoof-dpi's version; this may contain some other relevant information
  -window-size int
        chunk size, in number of bytes, for fragmented client hello,
        try lower values if the default value doesn't bypass the DPI;
        when not given, the client hello packet will be sent in two parts:
        fragmentation for the first data packet and the rest
```

> 如果你在 Chrome 浏览器使用其他 VPN 扩展比如 Hotspot Shield 请去 设置 > 扩展程序禁用它们

### OSX
运行 `spoof-dpi` ，然后它会自动设置自身为代理

### Linux
运行 `spoof-dpi` 然后加上代理参数运行你的浏览器
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
