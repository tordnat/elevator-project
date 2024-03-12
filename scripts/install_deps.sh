#!/bin/bash

echo "Installing dependencies"
sudo apt install golang
sudo apt install dmd

HRA_INSTALL_PATH="elevatorAlgorithm/hra/"
echo "Building project and moving hall_request_assigner to workspace root"
chmod +x Project-resources/cost_fns/hall_request_assigner/build.sh
Project-resources/cost_fns/hall_request_assigner/build.sh
sudo cp Project-resources/cost_fns/hall_request_assigner $HRA_INSTALL_PATH