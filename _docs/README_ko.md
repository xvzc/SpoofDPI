# SpoofDPI

다른 언어로 읽기: [🇬🇧English](https://github.com/xvzc/SpoofDPI), [🇰🇷한국어](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ko.md), [🇨🇳简体中文](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_zh-cn.md), [🇷🇺Русский](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ru.md), [🇯🇵日本語](https://github.com/xvzc/SpoofDPI/blob/main/_docs/README_ja.md)

DPI(Deep Packet Inspection) 우회를 위해 고안된 소프트웨어  
  
![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

# Installation
SpoofDPI의 설치과정은 [여기](https://github.com/xvzc/SpoofDPI/blob/main/_docs/INSTALL.md)를 참고바랍니다.


<a href="https://repology.org/project/spoofdpi/versions">
    <img src="https://repology.org/badge/vertical-allrepos/spoofdpi.svg?columns=1" alt="Packaging status">
</a>  

# 사용법
```
Usage: spoofdpi [options...]
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
> 만약 브라우저에서 Hotspot Shield와 같은 크롬 VPN 확장프로그램을 사용중이라면  
  Settings > Extension 으로 이동해 비활성화 해주시기바랍니다.
### OSX
터미널에서 `$ spoofdpi`를 실행합니다. Proxy 설정은 자동으로 수행됩니다.

### Linux
터미널에서 `$ spoofdpi`를 실행하고, 프록시 옵션과 함께 브라우저를 실행합니다.  
`google-chrome --proxy-server="http://127.0.0.1:8080"`

# 원리
### HTTP
최근 대부분의 웹사이트가 HTTPS를 지원하기 때문에, 
SpoofDPI는 HTTP 요청에 대한 DPI 우회는 지원하지 않습니다. 
다만 모든 HTTP 요청에 대한 Proxy 연결은 지원합니다.

### HTTPS
TLS는 모든 Handshake 과정을 암호화 합니다. 하지만, Client hello 패킷의 일부에는 여전히 서버의 도메인 네임이 평문으로 노출되어있습니다. 
다시 말하자면, 누군가가 암호화된 패킷을 본다면 해당 패킷의 목적지가 어딘지 손쉽게 알아차릴 수 있다는 뜻입니다. 
노출된 도메인은 DPI 검열에 매우 유용하게 사용될 수도 있고, 실제로 HTTPS 요청을 보냈을 때 차단이 이루어지는 시점도 Client hello 패킷을 보낸 시점입니다. 
여러가지 방법을 시도해본 결과, Client hello 패킷을 여러 조각으로 나누어 요청을 보냈을 때, 첫번째 조각에 대해서만 도메인 검열이 이루어지는 듯한 동작을 발견했습니다. 따라서 SpoofDPI는 해당 패킷을 두번에 나누어 보냅니다. 자세히 말하자면, 첫번째 1 바이트를 우선적으로 보내고, 나머지를 그 이후에 보내는 동작을 수행합니다.

# 참고
[Green Tunnel](https://github.com/SadeghHayeri/GreenTunnel) by @SadeghHayeri  
[GoodbyeDPI](https://github.com/ValdikSS/GoodbyeDPI) by @ValdikSS


