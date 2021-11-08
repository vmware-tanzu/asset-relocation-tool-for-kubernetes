#!/bin/bash
set -e

### Usage
### ----------------------------------------
### Set RELOCATION_CLI_VERSION env var or pass arg
### Print ls - export VERBOSE=true 
### ./entrypoint.sh "$RELOCATION_CLI_VERSION"
### ----------------------------------------



_ROOT_DIR="${PWD}"
_WORKDIR="${_ROOT_DIR}/vmware-relok8s"
#RELOCATION_CLI_VERSION=${"0.2.4":-$RELOCATION_CLI_VERSION} # Use env or arg

_RELOCATION_CLI_VERSION="0.3.39"

_RLOC_CLI_VERSION=${_RELOCATION_CLI_VERSION}
_DOWNLOAD_URL=""
_DOWNLOAD_FILENAME="relok8s-linux"



msg_log(){
    msg=$1
    echo -e ">> [LOG]: ${msg}"
}


set_workdir(){
    env
    echo "--------------"
    echo $RELOCATION_CLI_VERSION
    echo "${_RLOC_CLI_VERSION}"
    echo "--------------"   
    mkdir -p "${_WORKDIR}"
    cd "${_WORKDIR}"
}

set_download_url(){
    msg_log "Setting _DOWNLOAD_URL"

    msg_log "${_RLOC_CLI_VERSION}"
    
     _DOWNLOAD_URL="https://github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/releases/download/${_RLOC_CLI_VERSION}/relok8s-linux"
     

    msg_log "_DOWNLOAD_URL = ${_DOWNLOAD_URL}"
}    

download_reloc_cli(){
    msg_log "Downloading ..."
    wget "$_DOWNLOAD_URL" &&
    [[ $_VERBOSE = "true" ]] && ls -lah "$_DOWNLOAD_FILENAME"
    chmod +x relok8s-linux
}

install_reloc_cli(){
    msg_log "Installing ..."
    mv relok8s-linux /usr/bin/relok8s
  
}

test_reloc_cli(){
    msg_log "Printing Help"
    relok8s --help
}

#Main
set_workdir
set_download_url
download_reloc_cli
install_reloc_cli
test_reloc_cli