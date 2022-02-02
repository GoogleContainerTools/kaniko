# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "generic/debian10"
  config.vm.synced_folder ".", "/go/src/github.com/GoogleContainerTools/kaniko"
  config.ssh.extra_args = ["-t", "cd /go/src/github.com/GoogleContainerTools/kaniko; bash --login"]

  config.vm.provision "shell", inline: <<-SHELL
    apt-get update && apt-get install -y \
      apt-transport-https \
      ca-certificates \
      curl \
      gnupg-agent \
      html-xml-utils \
      python \
      wget \
      ca-certificates \
      jq \
      software-properties-common
    curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"
    apt-get update
    apt-get install -y docker-ce-cli docker-ce containerd.io
    usermod -a -G docker vagrant

    curl -LO https://storage.googleapis.com/container-diff/latest/container-diff-linux-amd64
    chmod +x container-diff-linux-amd64 && mv container-diff-linux-amd64 /usr/local/bin/container-diff

    wget --quiet https://storage.googleapis.com/pub/gsutil.tar.gz
    mkdir -p /opt/gsutil
    tar xfz gsutil.tar.gz -C /opt/
    rm gsutil.tar.gz
    ln -s /opt/gsutil/gsutil /usr/local/bin

    export GODLURL=https://go.dev/dl/$(curl --silent --show-error https://go.dev/dl/ | hxnormalize -x | hxselect -s "\n" "span, #filename" | grep linux | cut -d '>' -f 2 | cut -d '<' -f 1)
    echo "Downloading go from: $GODLURL"
    wget --quiet $GODLURL
    tar -C /usr/local -xzf go*.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin:/go/bin' > /etc/profile.d/go-path.sh
    echo 'export GOPATH=/go' >> /etc/profile.d/go-path.sh
    chmod a+x /etc/profile.d/go-path.sh
    chown vagrant /go
    chown vagrant /go/bin

    docker run --rm  -d -p 5000:5000 --name registry -e DEBUG=true registry:2
    echo 'export IMAGE_REPO=localhost:5000' > /etc/profile.d/local-registry.sh
    chmod a+x /etc/profile.d/local-registry.sh
    export PATH=$PATH:/usr/local/go/bin:/go/bin
    export GOPATH=/go
    go get github.com/google/go-containerregistry/cmd/crane
  SHELL
end
