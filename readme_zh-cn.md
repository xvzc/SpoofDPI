# SpoofDPI

选择语言: [English](https://github.com/xvzc/SpoofDPI), [한국어](https://github.com/xvzc/SpoofDPI/blob/main/readme_ko.md), [简体中文](https://github.com/xvzc/SpoofDPI/blob/main/readme_zh-cn.md)

规避**深度包检测**的简单工具
  
![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)


# 依赖

- Go

# 安装
## Binary

SpoofDPI 会被安装在 `~/.spoof-dpi/bin`
要在其他目录下运行，请给 `~/.bashrc || ~/.zshrc || ...` 添加

```
export PATH=$PATH:~/.spoof-dpi/bin
```


### curl
用 curl 安装最新的二进制可执行文件

- OSX
```
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s osx
```
- Linux
```
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux
```
### wget
用 wget 安装最新的二进制可执行文件
- OSX
```
wget -O - https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s osx 
```
- Linux
```
wget -O - https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s linux 
```

## Go
也可以用 **go install** 安装
 
`$ go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi`  
 > 记得确认 $GOPATH 在你的 $PATH 中
 
## Git
You can also build your own  
`$ git clone https://github.com/xvzc/SpoofDPI.git`  
`$ cd SpoofDPI`  
`$ go build ./cmd/...`  

# 使用方法

```
Usage: spoof-dpi [options...]
--dns=<addr>   | default: 8.8.8.8
--port=<port>  | default: 8080
--debug=<bool> | default: false
```

> 如果你在 Chrome 浏览器使用其他 VPN 扩展比如 Hotspot Shield 请去 设置 > 扩展程序禁用它们

### OSX
运行 `$ spoof-dpi` ，然后它会自动设置自身为代理

### Linux
运行 `$ spoof-dpi` 然后加上代理参数运行你的浏览器 

`google-chrome --proxy-server="http://127.0.0.1:8080"`

# 工作原理

### HTTP

因为世界上许多网站都已支持 HTTPS ，SpoofDPI 不会规避对 HTTP 请求的 DPI，但是它仍会为 HTTP 请求提供代理。

### HTTPS
尽管 TLS 1.3加密了握手的每一步，但是在 Client Hello 中的域名仍然是明文的。因此如果有人看到 Client Hello 包就可以知道你在连接什么网站。这给 DPI 提供了很大方便，我们也看到连接在 Client Hello 之后就会被屏蔽掉。我之前尝试了规避这种审查，并发现，如果把 Client Hello 分包，只有第一个 chunk 会被检测。SpoofDPI 只要在第一个分包发送 1 byte，然后再发送其他部分就能规避。
 
 > SpoofDPI 不会解密 HTTPS 请求，所以您无需安装任何 TLS 证书。
 
# 启发

[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel) by @SadeghHayeri  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) by @ValdikSS
