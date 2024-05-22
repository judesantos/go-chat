#!/bin/bash

set -e

# Function to install Docker and Docker Compose on Ubuntu/Debian
install_docker_ubuntu() {
  echo "Installing Docker on Ubuntu/Debian..."

  sudo apt-get update
  sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common

  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
  sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

  sudo apt-get update
  sudo apt-get install -y docker-ce

  sudo systemctl start docker
  sudo systemctl enable docker

  echo "Installing Docker Compose..."
  sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
  sudo chmod +x /usr/local/bin/docker-compose

  docker-compose --version

  echo "Docker and Docker Compose installed successfully on Ubuntu/Debian"
}

# Function to install Docker and Docker Compose on CentOS/RHEL
install_docker_centos() {
  echo "Installing Docker on CentOS/RHEL..."

  sudo yum install -y yum-utils device-mapper-persistent-data lvm2
  sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

  sudo yum install -y docker-ce

  sudo systemctl start docker
  sudo systemctl enable docker

  echo "Installing Docker Compose..."
  sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
  sudo chmod +x /usr/local/bin/docker-compose

  docker-compose --version

  echo "Docker and Docker Compose installed successfully on CentOS/RHEL"
}

# Function to install Docker and Docker Compose on macOS
install_docker_mac() {
  echo "Installing Docker on macOS..."

  # Install Homebrew if it's not already installed
  if ! command -v brew &>/dev/null; then
    echo "Homebrew not found. Installing Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  fi

  echo "Installing Docker and Docker Compose using Homebrew..."
  brew install --cask docker

  echo "Docker and Docker Compose installed successfully on macOS"
  echo "Please open Docker.app to finish the installation."
}

# Function to install Docker and Docker Compose on Windows
install_docker_windows() {
  echo "Installing Docker on Windows..."

  # Check if Chocolatey is installed
  if ! command -v choco &>/dev/null; then
    echo "Chocolatey not found. Please install Chocolatey first from https://chocolatey.org/install"
    exit 1
  fi

  echo "Installing Docker using Chocolatey..."
  choco install -y docker-desktop

  echo "Docker installed successfully on Windows"
  echo "Please restart your computer to complete the Docker installation."
}

# Function to detect the operating system and install Docker accordingly
detect_os_and_install() {

  # First, check if docker is not already installed
  docker info --format "{{.OperatingSystem}}" | grep -q "Docker"
  if [[ $? -eq 0 ]]; then
    echo "Docker is already installed"
    exit 0
  fi

  OS="$(uname -s)"

  case "$OS" in
    Linux)
      if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
      else
        echo "Cannot detect the Linux distribution."
        exit 1
      fi

      case $OS in
        ubuntu | debian)
          install_docker_ubuntu
          ;;
        centos | rhel | fedora)
          install_docker_centos
          ;;
        *)
          echo "Unsupported Linux distribution: $OS"
          exit 1
          ;;
      esac
      ;;
    Darwin)
      install_docker_mac
      ;;
    CYGWIN*|MINGW*|MSYS*|MINGW32*|MSYS2*)
      install_docker_windows
      ;;
    *)
      echo "Unsupported operating system: $OS"
      exit 1
      ;;
  esac
}

# Run the function to detect the OS and install Docker
detect_os_and_install
