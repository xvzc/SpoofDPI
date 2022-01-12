#!bin/bash

curl "https://api.github.com/repos/xvzc/SpoofDPI/releases/latest" |
    grep '"tag_name":' |                                                
    sed -E 's/.*"([^"]+)".*/\1/' |
    xargs -I {} curl -OL "https://github.com/xvzc/SpoofDPI/releases/download/"\{\}"/spoof-dpi-${1}.tar.gz"

tar -xzvf ./spoof-dpi-${1}.tar.gz && rm -rf ./spoof-dpi-${1}.tar.gz && mv ./spoof-dpi /usr/local/bin && hash -r

