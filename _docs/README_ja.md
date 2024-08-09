**⭐PRs or any form of contribution will be appreciated⭐**

# SpoofDPI

他の言語で読む: [🇬🇧English](https://github.com/xvzc/SpoofDPI), [🇰🇷한국어](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ko.md), [🇨🇳简体中文](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_zh-cn.md), [🇷🇺Русский](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ru.md), [🇯🇵日本語](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ja.md)

**Deep Packet Inspection**をバイパスするために設計されたシンプルで高速なソフトウェア  
  
![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

# インストール
## バイナリ
SpoofDPIは`~/.spoof-dpi/bin`にインストールされます。  
任意のディレクトリでSpoofDPIを実行するには、以下の行を`~/.bashrc || ~/.zshrc || ...`に追加してください。
```
export PATH=$PATH:~/.spoof-dpi/bin
```
---
```bash
# OSX
curl -fsSL https://raw.githubusercontent.com/xvzc/SpoofDPI/main/install.sh | bash -s darwin-amd64

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
`go install` でインストールすることもできます  
```bash
$ go install github.com/xvzc/SpoofDPI/cmd/spoof-dpi@latest
```

## Git
自分でビルドすることもできます  
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
  -debug
        enable debug output
  -dns-addr string
        dns address (default "8.8.8.8")
  -dns-port int
        port number for dns (default 53)
  -enable-doh
        enable 'dns-over-https'
  -no-banner
        disable banner
  -pattern value
        bypass DPI only on packets matching this regex pattern; can be given multiple times
  -pattern-file path
        bypass DPI only on packets matching regex patterns provided in a file (one per line)
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
> ChromeブラウザでHotspot ShieldなどのVPN拡張機能を使用している場合は、  
  設定 > 拡張機能に移動して無効にしてください。

### OSX
`spoof-dpi`を実行すると、自動的にプロキシが設定されます。

### Linux
`spoof-dpi`を実行し、プロキシオプションを使用してブラウザを開きます。  
```bash
google-chrome --proxy-server="http://127.0.0.1:8080"
```

# 仕組み
### HTTP
世界中のほとんどのウェブサイトがHTTPSをサポートしているため、SpoofDPIはHTTPリクエストのDeep Packet Inspectionをバイパスしませんが、すべてのHTTPリクエストに対してプロキシ接続を提供します。

### HTTPS
TLS はすべてのハンドシェイクプロセスを暗号化しますが、Client helloパケットには依然としてドメイン名がプレーンテキストで表示されます。 
つまり、他の誰かがパケットを見た場合、パケットがどこに向かっているのかを簡単に推測することができます。 
ドメイン名はDPIが処理されている間に重要な情報を提供することができ、実際にClient helloパケットを送信した直後に接続がブロックされることがわかります。
これをバイパスするためにいくつかの方法を試してみましたが、Client helloパケットをチャンクに分割して送信すると、最初のチャンクだけが検査されるように見えることがわかりました。 
SpoofDPIがこれをバイパスするために行うことは、リクエストの最初の1バイトをサーバーに送信し、その後に残りを送信することです。

# インスピレーション
[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel) by @SadeghHayeri  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) by @ValdikSS
