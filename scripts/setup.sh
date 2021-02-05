#!/bin/bash
set -euo pipefail

mkdir ~/actions-runner && cd ~/actions-runner
curl -O -L https://github.com/actions/runner/releases/download/v2.276.1/actions-runner-linux-x64-2.276.1.tar.gz
tar xzf ./actions-runner-linux-x64-2.276.1.tar.gz
sudo ./bin/installdependencies.sh
cd bin
for lib in $(find . -name 'System.*'); do
  toFile=$(echo "$lib" | sed -e 's/\.\/System\./.\/libSystem./g');
  if ! [ -f $toFile ]; then
    sudo ln -s $lib $toFile;
  fi;
done