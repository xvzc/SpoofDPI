# SpoofDPI

他の言語で読む: [🇬🇧English](https://github.com/xvzc/SpoofDPI), [🇰🇷한국어](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ko.md), [🇨🇳简体中文](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_zh-cn.md), [🇷🇺Русский](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ru.md), [🇯🇵日本語](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ja.md)

**Deep Packet Inspection**をバイパスするために設計されたシンプルで高速なソフトウェア  
  

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
> ChromeブラウザでHotspot ShieldなどのVPN拡張機能を使用している場合は、  
  設定 > 拡張機能に移動して無効にしてください。

### OSX
`spoofdpi`を実行すると、自動的にプロキシが設定されます。

### Linux
`spoofdpi`を実行し、プロキシオプションを使用してブラウザを開きます。  
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
