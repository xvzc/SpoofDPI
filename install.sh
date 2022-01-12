#!bin/bash

curl --silent "https://api.github.com/repos/xvzc/SpoofDPI/releases/latest" |
    grep '"tag_name":' |                                                
    sed -E 's/.*"([^"]+)".*/\1/' |
    xargs -I {} curl -sOL "https://github.com/xvzc/SpoofDPI/releases/download/"\{\}'/spoof-dpi-osx.tar.gz'

tar -xzvf ./spoof-dpi-osx.tar.gz && rm -rf ./spoof-dpi-osx.tar.gz
