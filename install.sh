#!bin/bash

curl "https://api.github.com/repos/xvzc/SpoofDPI/releases/latest" |
    grep '"tag_name":' |                                                
    sed -E 's/.*"([^"]+)".*/\1/' |
    xargs -I {} curl -OL "https://github.com/xvzc/SpoofDPI/releases/download/"\{\}"/spoof-dpi-${1}.tar.gz"

mkdir -p ~/.spoof-dpi/bin

tar -xzvf ./spoof-dpi-${1}.tar.gz && \
    rm -rf ./spoof-dpi-${1}.tar.gz && \
    mv ./spoof-dpi ~/.spoof-dpi/bin

if [ $? -ne 0 ]; then
    echo "Error. exiting now"
    exit
fi

export PATH=$PATH:~/.spoof-dpi/bin

echo ""
echo "Successfully installed SpoofDPI."
echo "Please add the line below to your rcfile(.bashrc or .zshrc etc..)"
echo ""
echo ">>    export PATH=\$PATH:~/.spoof-dpi/bin"
