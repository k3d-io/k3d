#!/bin/bash

NVIDIA_DRIVER_VERS="570"
sudo apt-get remove --purge '^nvidia-.*' 'libnvidia-*' -y && \
sudo apt-get autoremove -y && \
sudo apt-get autoclean -y && \
sudo apt update && \
sudo apt-get install -y software-properties-common && \
sudo add-apt-repository -y ppa:graphics-drivers/ppa && \
sudo apt-get update && \
sudo apt install nvidia-driver-${NVIDIA_DRIVER_VERS} -y && \
sudo reboot
# Then install following to have nvidia-smi
# sudo apt install nvidia-utils-570 -y
#
#
# Add NVIDIA GPG key
distribution="ubuntu22.04"
curl -s -L https://nvidia.github.io/libnvidia-container/gpgkey | \
  sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg && \
curl -s -L https://nvidia.github.io/libnvidia-container/$distribution/libnvidia-container.list | \
  sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
  sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list

# Install
sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit

# Configure Docker to use NVIDIA runtime
sudo nvidia-ctk runtime configure --runtime=docker

# Restart Docker
sudo systemctl restart docker



# Add NVIDIA GPG key
curl -s -L https://nvidia.github.io/libnvidia-container/gpgkey | \
  sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg

# Add repo using fake distribution
curl -s -L https://nvidia.github.io/libnvidia-container/$distribution/libnvidia-container.list | \
  sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
  sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list


sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit

sudo nvidia-ctk runtime configure --runtime=docker

sudo systemctl restart docker

docker run --rm --gpus all nvidia/cuda:12.8.1-base-ubuntu24.04 nvidia-smi
